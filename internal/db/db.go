// internal/db/db.go
package db

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/lib/pq" // El driver de postgres
)

func New(addr string) (*sql.DB, error) {
	db, err := sql.Open("postgres", addr)
	if err != nil {
		return nil, err
	}

	// Configuración del pool de conexiones
	db.SetMaxOpenConns(30)
	db.SetMaxIdleConns(30)
	db.SetConnMaxIdleTime(15 * time.Minute)

	// Verificamos que la conexión sea exitosa
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err = db.PingContext(ctx); err != nil {
		return nil, err
	}

	return db, nil
}
