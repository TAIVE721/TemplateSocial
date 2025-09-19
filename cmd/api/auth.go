// cmd/api/auth.go
package main

import (
	"crypto/sha256" // <--- Necesario para el token
	"fmt"
	"net/http"
	"time"

	// <--- ¡AQUÍ ESTÁ EL IMPORT QUE FALTABA!
	"GopherSocial/internal/store"

	"github.com/go-chi/chi/v5" // <--- Importante para leer parámetros de URL
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

	plainToken := uuid.New().String()
	hash := sha256.Sum256([]byte(plainToken))
	tokenHash := hash[:]

	err := app.store.Users.CreateAndInvite(r.Context(), user, tokenHash, time.Hour*72)
	if err != nil {
		switch err {
		case store.ErrDuplicateEmail, store.ErrDuplicateUsername:
			app.conflictResponse(w, r, err)
		default:
			app.internalServerError(w, r, err)
		}
		return
	}

	// ¡MUY IMPORTANTE! La URL de activación debe apuntar a tu frontend,
	// no al backend. El frontend luego hará la llamada a la API.
	// Por ahora, lo dejamos apuntando al backend para probar.
	activationURL := fmt.Sprintf("http://localhost:8080/v1/users/activate/%s", plainToken)
	data := map[string]string{
		"ActivationURL": activationURL,
		"Username":      user.Username,
	}

	_, err = app.mailer.Send("user_invitation.tmpl", user.Username, user.Email, data)
	if err != nil {
		app.logger.Printf("ERROR: could not send welcome email: %s", err)
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

// ¡NUEVO HANDLER!
// activateUserHandler maneja el token que viene en el enlace del correo.
func (app *application) activateUserHandler(w http.ResponseWriter, r *http.Request) {
	// 1. Leemos el token de la URL (ej: /v1/users/activate/ESTE_TOKEN)
	tokenPlainText := chi.URLParam(r, "token")

	// 2. Hasheamos el token para buscarlo en la base de datos.
	hash := sha256.Sum256([]byte(tokenPlainText))
	tokenHash := hash[:]

	// 3. Llamamos al método del store para activar al usuario.
	err := app.store.Users.Activate(r.Context(), tokenHash)
	if err != nil {
		switch err {
		case store.ErrNotFound:
			app.notFoundResponse(w, r)
		default:
			app.internalServerError(w, r, err)
		}
		return
	}

	// 4. Respondemos que todo salió bien.
	app.jsonResponse(w, http.StatusOK, map[string]string{"message": "¡Cuenta activada exitosamente!"})
}
