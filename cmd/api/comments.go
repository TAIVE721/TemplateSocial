// cmd/api/comments.go
package main

import (
	"GopherSocial/internal/store"
	"net/http"
)

type CreateCommentPayload struct {
	Content string `json:"content" validate:"required,max=1000"`
}

func (app *application) createCommentHandler(w http.ResponseWriter, r *http.Request) {
	// Obtenemos el post del contexto (nuestro middleware 'postsContextMiddleware' ya lo cargó)
	post := r.Context().Value(postCtxKey).(*store.Post)
	// Obtenemos al usuario que está comentando
	user := r.Context().Value(userCtxKey).(*store.User)

	var payload CreateCommentPayload
	if err := app.readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	if err := Validate.Struct(payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	comment := &store.Comment{
		Content: payload.Content,
		PostID:  post.ID,
		UserID:  user.ID,
	}

	if err := app.store.Comments.Create(r.Context(), comment); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	app.jsonResponse(w, http.StatusCreated, comment)
}
