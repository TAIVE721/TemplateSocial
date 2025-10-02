// cmd/api/middleware.go
package main

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/golang-jwt/jwt/v5"

	"GopherSocial/internal/store"

	"github.com/go-chi/chi/v5"
)

type userKey string

const userCtxKey userKey = "user"

func (app *application) AuthTokenMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			app.unauthorizedErrorResponse(w, r) // Necesitaremos este helper
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			app.unauthorizedErrorResponse(w, r)
			return
		}

		tokenString := parts[1]
		token, err := app.authenticator.ValidateToken(tokenString)
		if err != nil || !token.Valid {
			app.unauthorizedErrorResponse(w, r)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			app.unauthorizedErrorResponse(w, r)
			return
		}

		// Extraemos el ID de usuario del token
		userIDStr, err := claims.GetSubject()
		if err != nil {
			app.unauthorizedErrorResponse(w, r)
			return
		}

		userID, _ := strconv.ParseInt(userIDStr, 10, 64)

		// --- LÓGICA DE CACHING ---
		// 1. Intentamos obtener el usuario de la caché.
		user, err := app.cacheStorage.Users.Get(r.Context(), userID)
		if err != nil {
			app.internalServerError(w, r, err)
			return
		}

		// 2. Si no está en la caché (cache miss), lo buscamos en la DB.
		if user == nil {
			user, err = app.store.Users.GetByID(r.Context(), userID)
			if err != nil {
				app.unauthorizedErrorResponse(w, r)
				return
			}
			// 3. Y lo guardamos en la caché para la próxima vez.
			if err := app.cacheStorage.Users.Set(r.Context(), user); err != nil {
				app.internalServerError(w, r, err)
				return
			}
		}

		// TODO: Buscar el usuario en la BD y añadirlo al contexto.
		// Por ahora, solo pasamos el ID para mantenerlo simple.
		ctx := context.WithValue(r.Context(), userCtxKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type postKey string

const postCtxKey postKey = "post"

// postsContextMiddleware carga un post basado en el postID de la URL
// y lo guarda en el contexto de la petición.
func (app *application) postsContextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "postID"), 10, 64)
		if err != nil {
			app.notFoundResponse(w, r)
			return
		}

		post, err := app.store.Posts.GetByID(r.Context(), id)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				app.notFoundResponse(w, r)
				return
			}
			app.internalServerError(w, r, err)
			return
		}

		ctx := context.WithValue(r.Context(), postCtxKey, post)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (app *application) checkPermission(requiredLevel int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := r.Context().Value(userCtxKey).(*store.User)
			post := r.Context().Value(postCtxKey).(*store.Post) // Asumimos que el post está en el contexto

			// Un usuario siempre puede modificar sus propios posts
			if post.UserID == user.ID {
				next.ServeHTTP(w, r)
				return
			}

			// Si no es su post, verificamos si su nivel es suficiente
			if user.Role.Level >= requiredLevel {
				next.ServeHTTP(w, r)
				return
			}

			// Si no cumple ninguna condición, acceso denegado.
			app.forbiddenResponse(w, r)
		})
	}
}

func (app *application) RateLimiterMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Usamos RealIP para obtener la IP verdadera, incluso si hay un proxy.
		ip := r.RemoteAddr

		if allow, _ := app.rateLimiter.Allow(ip); !allow {
			app.rateLimitExceededResponse(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}
