// internal/env/env.go
package env

import (
	"os"
	"strconv"
)

// GetString lee una variable de entorno como string.
// Si no existe, devuelve un valor por defecto (fallback).
func GetString(key, fallback string) string {
	val, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	return val
}

// GetInt lee una variable de entorno como número entero.
// Si no existe o no es un número válido, devuelve el valor por defecto.
func GetInt(key string, fallback int) int {
	val, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}

	valAsInt, err := strconv.Atoi(val)
	if err != nil {
		return fallback
	}
	return valAsInt
}

// GetBool lee una variable de entorno como booleano.
func GetBool(key string, fallback bool) bool {
	val, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}

	boolVal, err := strconv.ParseBool(val)
	if err != nil {
		return fallback
	}
	return boolVal
}
