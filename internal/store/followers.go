// internal/store/followers.go
package store

import (
	"context"
	"database/sql"

	"github.com/lib/pq" // ¡Importante para manejar errores específicos de Postgres!
)

type FollowerStore struct {
	db *sql.DB
}

// Follow crea una nueva relación de seguimiento.
func (s *FollowerStore) Follow(ctx context.Context, userID, followerID int64) error {
	// userID es a quién siguen
	// followerID es quién sigue
	query := `
		INSERT INTO followers (user_id, follower_id) VALUES ($1, $2)`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	_, err := s.db.ExecContext(ctx, query, userID, followerID)
	if err != nil {
		// Verificamos si el error es porque ya existe la relación (error de clave primaria duplicada)
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return ErrConflict // Usamos nuestro error personalizado
		}
		return err
	}

	return nil
}

// Unfollow elimina una relación de seguimiento.
func (s *FollowerStore) Unfollow(ctx context.Context, userID, followerID int64) error {
	query := `
		DELETE FROM followers
		WHERE user_id = $1 AND follower_id = $2`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	_, err := s.db.ExecContext(ctx, query, userID, followerID)
	return err
}
