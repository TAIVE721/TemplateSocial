// internal/store/cache/storage.go
package cache

import (
	"context"

	"GopherSocial/internal/store" // Reemplaza con tu ruta

	"github.com/go-redis/redis/v8"
)

type Storage struct {
	Users UserCacher
}

// Definimos una interfaz para que nuestro código sea testeable.
type UserCacher interface {
	Get(context.Context, int64) (*store.User, error)
	Set(context.Context, *store.User) error
	Delete(context.Context, int64)
}

func NewRedisStorage(rdb *redis.Client) Storage {
	return Storage{
		Users: &UserStore{rdb: rdb},
	}
}
