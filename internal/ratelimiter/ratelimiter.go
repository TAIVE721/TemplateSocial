// internal/ratelimiter/ratelimiter.go
package ratelimiter

import "time"

// Limiter es la interfaz que nuestros limitadores deben cumplir.
type Limiter interface {
	Allow(ip string) (bool, time.Duration)
}

// Config contiene los ajustes para el rate limiter.
type Config struct {
	RequestsPerTimeFrame int
	TimeFrame            time.Duration
	Enabled              bool
}
