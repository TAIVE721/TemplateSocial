// internal/store/storage.go
package store

import (
	"database/sql"
	"errors"
	"time"
)

var (
	ErrNotFound          = errors.New("resource not found")
	ErrConflict          = errors.New("resource already exists")
	QueryTimeoutDuration = time.Second * 5
)

type Storage struct {
	Users     *UserStore
	Posts     *PostStore
	Followers *FollowerStore
	Roles     *RoleStore
	Comments  *CommentStore // <-- AÑADE ESTO
}

func NewStorage(db *sql.DB) Storage {
	return Storage{
		Users:     &UserStore{db: db},
		Posts:     &PostStore{db: db},
		Followers: &FollowerStore{db: db},
		Roles:     &RoleStore{db: db},
		Comments:  &CommentStore{db: db}, // <-- Y ESTO
	}
}
