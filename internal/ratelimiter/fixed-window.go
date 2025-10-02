// internal/ratelimiter/fixed_window.go
package ratelimiter

import (
	"sync"
	"time"
)

// FixedWindowRateLimiter implementa nuestro Limiter.
type FixedWindowRateLimiter struct {
	sync.RWMutex
	clients map[string]int
	limit   int
	window  time.Duration
}

// NewFixedWindowLimiter crea una nueva instancia de nuestro limitador.
func NewFixedWindowLimiter(limit int, window time.Duration) *FixedWindowRateLimiter {
	limiter := &FixedWindowRateLimiter{
		clients: make(map[string]int),
		limit:   limit,
		window:  window,
	}
	return limiter
}

// Allow comprueba si una IP tiene permiso para hacer una petición.
func (rl *FixedWindowRateLimiter) Allow(ip string) (bool, time.Duration) {
	rl.Lock()
	defer rl.Unlock()

	count, exists := rl.clients[ip]

	if !exists {
		// Si es la primera vez que vemos esta IP, la añadimos y le damos permiso.
		rl.clients[ip] = 1
		// Programamos que se reinicie su contador después de que pase la ventana de tiempo.
		go rl.resetCount(ip)
		return true, 0
	}

	if count < rl.limit {
		// Si aún no ha llegado al límite, incrementamos su contador y le damos permiso.
		rl.clients[ip]++
		return true, 0
	}

	// Si ha llegado al límite, le denegamos el permiso.
	return false, rl.window
}

// resetCount borra el contador de una IP después de que la ventana de tiempo haya pasado.
func (rl *FixedWindowRateLimiter) resetCount(ip string) {
	time.Sleep(rl.window)
	rl.Lock()
	defer rl.Unlock()
	delete(rl.clients, ip)
}
