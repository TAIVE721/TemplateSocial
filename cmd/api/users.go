// cmd/api/users.go
package main

import (
	"net/http"
	"strconv"

	"GopherSocial/internal/store" // Asegúrate que la ruta sea correcta

	"github.com/go-chi/chi/v5"
)

// followUserHandler permite al usuario autenticado seguir a otro usuario.
func (app *application) followUserHandler(w http.ResponseWriter, r *http.Request) {
	// 1. Obtenemos al usuario que quiere seguir (el que está logueado) desde el contexto.
	followerUser := r.Context().Value(userCtxKey).(*store.User)

	// 2. Obtenemos el ID del usuario que será seguido desde la URL.
	followedID, err := strconv.ParseInt(chi.URLParam(r, "userID"), 10, 64)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// 3. Llamamos a nuestra lógica del store.
	err = app.store.Followers.Follow(r.Context(), followedID, followerUser.ID)
	if err != nil {
		if err == store.ErrConflict {
			app.conflictResponse(w, r, err)
			return
		}
		app.internalServerError(w, r, err)
		return
	}

	// 4. Respondemos con éxito.
	w.WriteHeader(http.StatusNoContent)
}

// unfollowUserHandler permite al usuario autenticado dejar de seguir a otro.
func (app *application) unfollowUserHandler(w http.ResponseWriter, r *http.Request) {
	// El flujo es muy similar al de 'follow'
	followerUser := r.Context().Value(userCtxKey).(*store.User)

	unfollowedID, err := strconv.ParseInt(chi.URLParam(r, "userID"), 10, 64)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	err = app.store.Followers.Unfollow(r.Context(), unfollowedID, followerUser.ID)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
