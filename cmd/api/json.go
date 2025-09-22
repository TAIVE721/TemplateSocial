// cmd/api/json.go
package main

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-playground/validator/v10"
)

var Validate *validator.Validate

func init() {
	Validate = validator.New(validator.WithRequiredStructEnabled())
}

// writeJSONError es un helper para errores.
func (app *application) writeJSONError(w http.ResponseWriter, status int, message string) {
	type envelope struct {
		Error string `json:"error"`
	}
	writeJSON(w, status, envelope{Error: message})
}

// Movemos los helpers que ya teníamos de main.go aquí.
func (app *application) jsonResponse(w http.ResponseWriter, status int, data any) error {
	type envelope struct {
		Data any `json:"data"`
	}
	return writeJSON(w, status, envelope{Data: data})
}

func writeJSON(w http.ResponseWriter, status int, data any) error {
	js, err := json.Marshal(data)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(js)
	return nil
}

// ¡Nuevo helper para leer JSON de forma segura!
func (app *application) readJSON(w http.ResponseWriter, r *http.Request, data any) error {
	maxBytes := 1_048_578 // 1MB
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields() // No permite campos desconocidos en el JSON

	err := decoder.Decode(data)
	if err != nil {
		// Aquí se podrían manejar errores más específicos de JSON
		return err
	}

	// Verificamos que no haya más de un objeto JSON en el body
	err = decoder.Decode(&struct{}{})
	if err.Error() != "EOF" {
		return errors.New("el cuerpo debe contener un único objeto JSON")
	}

	return nil
}
