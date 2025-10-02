// cmd/api/errors.go
package main

import (
	"log"
	"net/http"
)

func (app *application) internalServerError(w http.ResponseWriter, r *http.Request, err error) {
	log.Println(err)
	app.writeJSONError(w, http.StatusInternalServerError, "el servidor encontró un problema")
}

func (app *application) badRequestResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.writeJSONError(w, http.StatusBadRequest, err.Error())
}

func (app *application) conflictResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.writeJSONError(w, http.StatusConflict, err.Error())
}

func (app *application) unauthorizedErrorResponse(w http.ResponseWriter, r *http.Request) {
	app.writeJSONError(w, http.StatusUnauthorized, "credenciales inválidas o token no autorizado")
}

// cmd/api/errors.go
func (app *application) notFoundResponse(w http.ResponseWriter, r *http.Request) {
	app.writeJSONError(w, http.StatusNotFound, "el recurso solicitado no fue encontrado")
}

func (app *application) forbiddenResponse(w http.ResponseWriter, r *http.Request) {
	app.writeJSONError(w, http.StatusForbidden, "no tienes permiso para realizar esta acción")
}

func (app *application) rateLimitExceededResponse(w http.ResponseWriter, r *http.Request) {
	message := "límite de peticiones excedido"
	app.writeJSONError(w, http.StatusTooManyRequests, message)
}
