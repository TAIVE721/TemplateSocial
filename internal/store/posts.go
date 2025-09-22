// internal/store/posts.go
package store

import (
	"context"
	"database/sql"
	"errors"
)

type Post struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	UserID    int64  `json:"user_id"`
	CreatedAt string `json:"created_at"`
	Version   int    `json:"version"`
	User      User   `json:"user"` // ¡Campo nuevo!
}

type PostStore struct {
	db *sql.DB
}

// GetByID recupera una publicación por su ID.
func (s *PostStore) GetByID(ctx context.Context, id int64) (*Post, error) {
	query := `SELECT id, title, content, user_id, created_at, version FROM posts WHERE id = $1` // Añadir version
	var post Post
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	err := s.db.QueryRowContext(ctx, query, id).Scan(&post.ID, &post.Title, &post.Content, &post.UserID, &post.CreatedAt, &post.Version) // Añadir version
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &post, nil
}

// Create inserta una nueva publicación en la base de datos.
func (s *PostStore) Create(ctx context.Context, post *Post) error {
	query := `
		INSERT INTO posts (title, content, user_id)
		VALUES ($1, $2, $3)
		RETURNING id, created_at`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	return s.db.QueryRowContext(ctx, query, post.Title, post.Content, post.UserID).Scan(&post.ID, &post.CreatedAt)
}

func (s *PostStore) Update(ctx context.Context, post *Post) error {
	query := `
		UPDATE posts SET title = $1, content = $2, version = version + 1
		WHERE id = $3 AND version = $4
		RETURNING version`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	// Usamos la versión para asegurarnos de que no estamos actualizando un post obsoleto.
	err := s.db.QueryRowContext(ctx, query, post.Title, post.Content, post.ID, post.Version).Scan(&post.Version)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound // Puede ser que el post no exista o la versión sea incorrecta
		}
		return err
	}
	return nil
}

func (s *PostStore) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM posts WHERE id = $1`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	res, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// PostWithMetadata es una nueva struct que incluye el Post y un contador de comentarios.
// La usaremos para nuestro feed.
type PostWithMetadata struct {
	Post
	CommentsCount int  `json:"comments_count"`
	User          User `json:"user,omitempty"`
}

// GetUserFeed recupera las publicaciones de los usuarios que sigue el userID.
func (s *PostStore) GetUserFeed(ctx context.Context, userID int64) ([]PostWithMetadata, error) {
	query := `
		SELECT
			p.id, p.user_id, p.title, p.content, p.created_at, p.version,
			u.username
		FROM posts p
		JOIN users u ON p.user_id = u.id
		JOIN followers f ON f.user_id = p.user_id
		WHERE f.follower_id = $1 OR p.user_id = $1
		GROUP BY p.id, u.username
		ORDER BY p.created_at DESC
		LIMIT 20`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var feed []PostWithMetadata
	for rows.Next() {
		var p PostWithMetadata
		err := rows.Scan(
			&p.ID,
			&p.UserID,
			&p.Title,
			&p.Content,
			&p.CreatedAt,
			&p.Version,
			&p.User.Username, // <-- Ahora SÍ existe p.User.Username
		)
		if err != nil {
			return nil, err
		}

		// Asignamos el ID del usuario de la publicación
		p.User.ID = p.UserID
		feed = append(feed, p)
	}

	return feed, nil
}
