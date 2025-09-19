// cmd/api/auth.go
package main

import (
	"fmt"
	"net/http"
	"time"

	"GopherSocial/internal/store"

	"github.com/golang-jwt/jwt/v5"
)

type RegisterUserPayload struct {
	Username string `json:"username" validate:"required,max=100"`
	Email    string `json:"email" validate:"required,email,max=255"`
	Password string `json:"password" validate:"required,min=3"`
}

func (app *application) registerUserHandler(w http.ResponseWriter, r *http.Request) {
	var payload RegisterUserPayload

	// Usamos nuestro nuevo helper para leer el JSON de forma segura.
	err := app.readJSON(w, r, &payload)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// TODO: Añadir validación aquí

	user := &store.User{
		Username: payload.Username,
		Email:    payload.Email,
	}

	if err := user.Password.Set(payload.Password); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	err = app.store.Users.Create(r.Context(), user)
	if err != nil {
		switch err {
		case store.ErrDuplicateEmail, store.ErrDuplicateUsername:
			app.conflictResponse(w, r, err)
		default:
			app.internalServerError(w, r, err)
		}
		return
	}

	app.jsonResponse(w, http.StatusCreated, user)
}

type CreateUserTokenPayload struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
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

	// ¡Las credenciales son correctas! Creamos el token.
	claims := jwt.RegisteredClaims{
		Subject:   fmt.Sprintf("%d", user.ID), // 'sub' es el ID del usuario
		Issuer:    "gophersocial",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24)), // Expira en 24h
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}

	token, err := app.authenticator.GenerateToken(claims)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	app.jsonResponse(w, http.StatusCreated, map[string]string{"token": token})
}
