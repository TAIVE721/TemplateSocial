// internal/store/users.go
package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrDuplicateEmail    = errors.New("a user with that email already exists")
	ErrDuplicateUsername = errors.New("a user with that username already exists")
)

// User define nuestro modelo de datos.
type User struct {
	ID        int64    `json:"id"`
	Username  string   `json:"username"`
	Email     string   `json:"email"`
	Password  password `json:"-"`
	CreatedAt string   `json:"created_at"`
	IsActive  bool     `json:"is_active"`
	RoleID    int64    `json:"-"`    // ID del rol, no lo exponemos en el JSON
	Role      Role     `json:"role"` // Struct anidada con la info del rol
}

type Role struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Level int    `json:"level"`
}

// password es un tipo custom para manejar el hash de forma segura.
type password struct {
	text *string
	hash []byte
}

// UserStore encapsula la conexión a la DB.
type UserStore struct {
	db *sql.DB
}

// --- HELPERS ---

// withTx es una función auxiliar para manejar transacciones de base de datos.
// Se asegura de que si alguna operación dentro de la transacción falla, todo se revierte.
func withTx(db *sql.DB, ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

// Set hashea una contraseña en texto plano.
func (p *password) Set(text string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(text), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	p.text = &text
	p.hash = hash
	return nil
}

// Compare verifica si una contraseña en texto plano coincide con el hash.
func (p *password) Compare(text string) error {
	return bcrypt.CompareHashAndPassword(p.hash, []byte(text))
}

// --- MÉTODOS PRINCIPALES ---

// CreateAndInvite es el punto de entrada para el registro.
// Orquesta la creación del usuario y su token de invitación dentro de una sola transacción.
func (s *UserStore) CreateAndInvite(ctx context.Context, user *User, tokenHash []byte, exp time.Duration) error {
	return withTx(s.db, ctx, func(tx *sql.Tx) error {
		if err := s.Create(ctx, tx, user); err != nil {
			return err
		}
		if err := s.createUserInvitation(ctx, tx, tokenHash, user.ID, exp); err != nil {
			return err
		}
		return nil
	})
}

// Activate es el punto de entrada para la activación de la cuenta.
// Orquesta la verificación, activación y limpieza dentro de una sola transacción.
func (s *UserStore) Activate(ctx context.Context, tokenHash []byte) error {
	return withTx(s.db, ctx, func(tx *sql.Tx) error {
		user, err := s.getUserFromInvitation(ctx, tx, tokenHash)
		if err != nil {
			return err
		}

		user.IsActive = true
		if err := s.update(ctx, tx, user); err != nil {
			return err
		}

		if err := s.deleteUserInvitations(ctx, tx, user.ID); err != nil {
			return err
		}

		return nil
	})
}

// GetByID busca un usuario por su ID.
func (s *UserStore) GetByID(ctx context.Context, id int64) (*User, error) {
	query := `SELECT id, username, email, password, created_at, is_active FROM users WHERE id = $1`
	var user User
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	err := s.db.QueryRowContext(ctx, query, id).Scan(&user.ID, &user.Username, &user.Email, &user.Password.hash, &user.CreatedAt, &user.IsActive)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

// GetByEmail busca un usuario (que debe estar activo) por su email.
func (s *UserStore) GetByEmail(ctx context.Context, email string) (*User, error) {
	query := `SELECT id, username, email, password, created_at FROM users WHERE email = $1 AND is_active = TRUE`
	var user User
	err := s.db.QueryRowContext(ctx, query, email).Scan(&user.ID, &user.Username, &user.Email, &user.Password.hash, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

// --- FUNCIONES INTERNAS (NO EXPUESTAS) ---

// create es la función interna que realmente inserta el usuario. Se llama desde CreateAndInvite.
func (s *UserStore) Create(ctx context.Context, tx *sql.Tx, user *User) error {
	query := `
		INSERT INTO users (username, password, email) VALUES ($1, $2, $3)
    	RETURNING id, created_at, is_active`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	err := tx.QueryRowContext(ctx, query, user.Username, user.Password.hash, user.Email).Scan(&user.ID, &user.CreatedAt, &user.IsActive)
	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"`:
			return ErrDuplicateEmail
		case err.Error() == `pq: duplicate key value violates unique constraint "users_username_key"`:
			return ErrDuplicateUsername
		default:
			return err
		}
	}
	return nil
}

// createUserInvitation inserta el token de activación.
func (s *UserStore) createUserInvitation(ctx context.Context, tx *sql.Tx, tokenHash []byte, userID int64, exp time.Duration) error {
	query := `INSERT INTO user_invitations (token, user_id, expiry) VALUES ($1, $2, $3)`
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()
	_, err := tx.ExecContext(ctx, query, tokenHash, userID, time.Now().Add(exp))
	return err
}

// getUserFromInvitation busca un usuario a partir de un token de invitación válido.
func (s *UserStore) getUserFromInvitation(ctx context.Context, tx *sql.Tx, tokenHash []byte) (*User, error) {
	query := `
        SELECT u.id, u.username, u.email, u.created_at, u.is_active
        FROM users u
        JOIN user_invitations ui ON u.id = ui.user_id
        WHERE ui.token = $1 AND ui.expiry > $2`

	var user User
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	err := tx.QueryRowContext(ctx, query, tokenHash, time.Now()).Scan(&user.ID, &user.Username, &user.Email, &user.CreatedAt, &user.IsActive)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

// update actualiza los datos de un usuario, en este caso, el campo is_active.
func (s *UserStore) update(ctx context.Context, tx *sql.Tx, user *User) error {
	query := `UPDATE users SET is_active = $1 WHERE id = $2`
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()
	_, err := tx.ExecContext(ctx, query, user.IsActive, user.ID)
	return err
}

// deleteUserInvitations elimina todos los tokens de un usuario después de que uno ha sido usado.
func (s *UserStore) deleteUserInvitations(ctx context.Context, tx *sql.Tx, userID int64) error {
	query := `DELETE FROM user_invitations WHERE user_id = $1`
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()
	_, err := tx.ExecContext(ctx, query, userID)
	return err
}
