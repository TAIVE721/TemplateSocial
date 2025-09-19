// internal/store/storage.go
package store

import (
	"database/sql"
	"errors"
	"time"
)

var (
	ErrNotFound          = errors.New("resource not found")
	QueryTimeoutDuration = time.Second * 5
)

// Storage agrupará todos nuestros tipos de store (Users, Posts, etc.)
type Storage struct {
	Users *UserStore
	Posts *PostStore
}

func NewStorage(db *sql.DB) Storage {
	return Storage{
		Users: &UserStore{db: db},
		Posts: &PostStore{db: db},
	}
}
