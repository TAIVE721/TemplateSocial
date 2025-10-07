// internal/store/comments.go
package store

import (
	"context"
	"database/sql"
)

type Comment struct {
	ID        int64  `json:"id"`
	Content   string `json:"content"`
	PostID    int64  `json:"post_id"`
	UserID    int64  `json:"user_id"`
	CreatedAt string `json:"created_at"`
	User      User   `json:"user"` // Para mostrar quién comentó
}

type CommentStore struct {
	db *sql.DB
}

// Create inserta un nuevo comentario.
func (s *CommentStore) Create(ctx context.Context, comment *Comment) error {
	query := `
		INSERT INTO comments (post_id, user_id, content)
		VALUES ($1, $2, $3)
		RETURNING id, created_at`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	return s.db.QueryRowContext(ctx, query, comment.PostID, comment.UserID, comment.Content).Scan(&comment.ID, &comment.CreatedAt)
}

// GetByPostID recupera todos los comentarios de una publicación.
func (s *CommentStore) GetByPostID(ctx context.Context, postID int64) ([]Comment, error) {
	query := `
		SELECT c.id, c.content, c.post_id, c.user_id, c.created_at, u.username, u.id
		FROM comments c
		JOIN users u on u.id = c.user_id
		WHERE c.post_id = $1
		ORDER BY c.created_at DESC`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	rows, err := s.db.QueryContext(ctx, query, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []Comment
	for rows.Next() {
		var c Comment
		err := rows.Scan(&c.ID, &c.Content, &c.PostID, &c.UserID, &c.CreatedAt, &c.User.Username, &c.User.ID)
		if err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}

	return comments, nil
}
