// cmd/api/auth.go
package main

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"time"

	// <--- Importamos el mailer
	"GopherSocial/internal/store"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type RegisterUserPayload struct {
	Username string `json:"username" validate:"required,max=100"`
	Email    string `json:"email" validate:"required,email,max=255"`
	Password string `json:"password" validate:"required,min=3"`
}

type CreateUserTokenPayload struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// UserWithToken no se usa en este handler, pero es bueno tenerla para referencia futura
type UserWithToken struct {
	*store.User
	Token string `json:"token"`
}

func (app *application) registerUserHandler(w http.ResponseWriter, r *http.Request) {
	var payload RegisterUserPayload
	if err := app.readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	user := &store.User{
		Username: payload.Username,
		Email:    payload.Email,
	}

	if err := user.Password.Set(payload.Password); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	// 1. Generamos un token único para la activación
	plainToken := uuid.New().String()
	hash := sha256.Sum256([]byte(plainToken))
	tokenHash := hash[:]

	// 2. Usamos `CreateAndInvite`
	err := app.store.Users.CreateAndInvite(r.Context(), user, tokenHash, time.Hour*24*3) // Expira en 3 días
	if err != nil {
		switch err {
		case store.ErrDuplicateEmail, store.ErrDuplicateUsername:
			app.conflictResponse(w, r, err)
		default:
			app.internalServerError(w, r, err)
		}
		return
	}

	// 3. Preparamos los datos para la plantilla del correo
	activationURL := fmt.Sprintf("http://localhost:8080/v1/users/activate/%s", plainToken)
	data := struct {
		Username      string
		ActivationURL string
	}{
		Username:      user.Username,
		ActivationURL: activationURL,
	}

	// 4. ¡Enviamos el correo!
	_, err = app.mailer.Send("user_invitation.tmpl", user.Username, user.Email, data)
	if err != nil {
		app.logger.Printf("error sending welcome email: %s", err)
		app.internalServerError(w, r, err)
		return
	}

	app.jsonResponse(w, http.StatusCreated, user)
}

func (app *application) createTokenHandler(w http.ResponseWriter, r *http.Request) {
	var payload CreateUserTokenPayload
	if err := app.readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	user, err := app.store.Users.GetByEmail(r.Context(), payload.Email)
	if err != nil {
		app.unauthorizedErrorResponse(w, r)
		return
	}

	if err := user.Password.Compare(payload.Password); err != nil {
		app.unauthorizedErrorResponse(w, r)
		return
	}

	claims := jwt.RegisteredClaims{
		Subject:   fmt.Sprintf("%d", user.ID),
		Issuer:    "gophersocial",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}

	token, err := app.authenticator.GenerateToken(claims)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	app.jsonResponse(w, http.StatusCreated, map[string]string{"token": token})
}
