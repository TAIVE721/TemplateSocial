// internal/auth/jwt.go
package auth

import (
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

type JWTAuthenticator struct {
	secret string
	iss    string // Issuer (quién emite el token)
	aud    string // Audience (para quién es el token)
}

func NewJWTAuthenticator(secret, iss, aud string) *JWTAuthenticator {
	return &JWTAuthenticator{secret: secret, iss: iss, aud: aud}
}

// GenerateToken crea un nuevo token JWT con los claims dados.
func (a *JWTAuthenticator) GenerateToken(claims jwt.Claims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Firmamos el token con nuestro secreto. ¡Esta es la parte crucial!
	return token.SignedString([]byte(a.secret))
}

// ValidateToken verifica la firma y validez de un token.
func (a *JWTAuthenticator) ValidateToken(tokenString string) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("método de firma inesperado: %v", token.Header["alg"])
		}
		return []byte(a.secret), nil
	})
}
