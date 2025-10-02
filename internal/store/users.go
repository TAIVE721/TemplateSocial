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

// Role define el modelo de datos para un rol.
type Role struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Level int    `json:"level"`
}

// User define nuestro modelo de datos.
type User struct {
	ID        int64    `json:"id"`
	Username  string   `json:"username"`
	Email     string   `json:"email"`
	Password  password `json:"-"`
	CreatedAt string   `json:"created_at"`
	IsActive  bool     `json:"is_active"`
	RoleID    int64    `json:"-"`    // No lo exponemos en el JSON
	Role      Role     `json:"role"` // Struct anidada con la info del rol
}

type password struct {
	text *string
	hash []byte
}

type UserStore struct {
	db *sql.DB
}

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

func (p *password) Set(text string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(text), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	p.text = &text
	p.hash = hash
	return nil
}

func (p *password) Compare(text string) error {
	return bcrypt.CompareHashAndPassword(p.hash, []byte(text))
}

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

func (s *UserStore) GetByID(ctx context.Context, id int64) (*User, error) {
	query := `
		SELECT u.id, u.username, u.email, u.password, u.created_at, u.is_active,
		       r.id, r.name, r.level
		FROM users u
		JOIN roles r ON u.role_id = r.id
		WHERE u.id = $1`

	var user User
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID, &user.Username, &user.Email, &user.Password.hash, &user.CreatedAt, &user.IsActive,
		&user.Role.ID, &user.Role.Name, &user.Role.Level,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (s *UserStore) GetByEmail(ctx context.Context, email string) (*User, error) {
	query := `
		SELECT u.id, u.username, u.email, u.password, u.created_at, u.is_active,
		       r.id, r.name, r.level
		FROM users u
		JOIN roles r ON u.role_id = r.id
		WHERE u.email = $1 AND u.is_active = TRUE`

	var user User
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	err := s.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID, &user.Username, &user.Email, &user.Password.hash, &user.CreatedAt, &user.IsActive,
		&user.Role.ID, &user.Role.Name, &user.Role.Level,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (s *UserStore) Create(ctx context.Context, tx *sql.Tx, user *User) error {
	query := `
		INSERT INTO users (username, password, email, role_id)
		VALUES ($1, $2, $3, (SELECT id FROM roles WHERE name = 'user'))
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

func (s *UserStore) createUserInvitation(ctx context.Context, tx *sql.Tx, tokenHash []byte, userID int64, exp time.Duration) error {
	query := `INSERT INTO user_invitations (token, user_id, expiry) VALUES ($1, $2, $3)`
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()
	_, err := tx.ExecContext(ctx, query, tokenHash, userID, time.Now().Add(exp))
	return err
}

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

func (s *UserStore) update(ctx context.Context, tx *sql.Tx, user *User) error {
	query := `UPDATE users SET is_active = $1 WHERE id = $2`
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()
	_, err := tx.ExecContext(ctx, query, user.IsActive, user.ID)
	return err
}

func (s *UserStore) deleteUserInvitations(ctx context.Context, tx *sql.Tx, userID int64) error {
	query := `DELETE FROM user_invitations WHERE user_id = $1`
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()
	_, err := tx.ExecContext(ctx, query, userID)
	return err
}
