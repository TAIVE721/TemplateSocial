// internal/store/roles.go
package store

import (
	"context"
	"database/sql"
)

type RoleStore struct {
	db *sql.DB
}

// GetByName busca un rol por su nombre (ej. "admin").
func (s *RoleStore) GetByName(ctx context.Context, name string) (*Role, error) {
	query := `SELECT id, name, level FROM roles WHERE name = $1`

	var role Role
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	err := s.db.QueryRowContext(ctx, query, name).Scan(&role.ID, &role.Name, &role.Level)
	if err != nil {
		return nil, err
	}

	return &role, nil
}
