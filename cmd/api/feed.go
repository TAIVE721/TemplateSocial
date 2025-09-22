// cmd/api/feed.go
package main

import (
	"GopherSocial/internal/store"
	"net/http"
)

func (app *application) getUserFeedHandler(w http.ResponseWriter, r *http.Request) {
	// 1. Obtenemos al usuario logueado desde el contexto.
	user := r.Context().Value(userCtxKey).(*store.User)

	// 2. Llamamos a nuestra nueva función del store.
	feed, err := app.store.Posts.GetUserFeed(r.Context(), user.ID)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	// 3. Devolvemos el feed como JSON.
	app.jsonResponse(w, http.StatusOK, feed)
}
