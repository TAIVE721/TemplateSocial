// cmd/api/posts.go
package main

import (
	"errors"
	"net/http"

	"GopherSocial/internal/store" // Reemplaza con tu ruta
)

// getPostHandler maneja la recuperación de una publicación.
func (app *application) getPostHandler(w http.ResponseWriter, r *http.Request) {
	post := r.Context().Value(postCtxKey).(*store.Post)
	app.jsonResponse(w, http.StatusOK, post)
}

type CreatePostPayload struct {
	Title   string `json:"title"   validate:"required,max=100"`
	Content string `json:"content" validate:"required"`
}

// createPostHandler maneja la creación de una nueva publicación.
func (app *application) createPostHandler(w http.ResponseWriter, r *http.Request) {
	var payload CreatePostPayload
	if err := app.readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := Validate.Struct(payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// Extraemos el ID de usuario del contexto que nuestro middleware inyectó.
	user, ok := r.Context().Value(userCtxKey).(*store.User)
	if !ok {
		app.internalServerError(w, r, errors.New("usuario no encontrado en el contexto"))
		return
	}

	post := &store.Post{
		Title:   payload.Title,
		Content: payload.Content,
		UserID:  user.ID,
	}

	if err := app.store.Posts.Create(r.Context(), post); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	app.jsonResponse(w, http.StatusCreated, post)
}

type UpdatePostPayload struct {
	Title   *string `json:"title"` // Usamos punteros para detectar si el campo fue enviado o no
	Content *string `json:"content"`
}

func (app *application) updatePostHandler(w http.ResponseWriter, r *http.Request) {
	post := r.Context().Value(postCtxKey).(*store.Post)

	var payload UpdatePostPayload
	if err := app.readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// Actualizamos solo los campos que se enviaron
	if payload.Title != nil {
		post.Title = *payload.Title
	}
	if payload.Content != nil {
		post.Content = *payload.Content
	}

	if err := app.store.Posts.Update(r.Context(), post); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	app.jsonResponse(w, http.StatusOK, post)
}

func (app *application) deletePostHandler(w http.ResponseWriter, r *http.Request) {
	post := r.Context().Value(postCtxKey).(*store.Post)

	if err := app.store.Posts.Delete(r.Context(), post.ID); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent) // 204 No Content
}
