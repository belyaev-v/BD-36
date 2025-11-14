package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type Post struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Link        string    `json:"link"`
	PublishedAt time.Time `json:"published_at"`
	CreatedAt   time.Time `json:"created_at"`
}
type Store interface {
	SavePosts(ctx context.Context, posts []Post) error
	Latest(ctx context.Context, limit int) ([]Post, error)
}
type Postgres struct {
	db *sql.DB
}

func NewPostgres(db *sql.DB) *Postgres {
	return &Postgres{db: db}
}
func (p *Postgres) SavePosts(ctx context.Context, posts []Post) error {
	if len(posts) == 0 {
		return nil
	}

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	stmt, err := tx.PrepareContext(ctx, `
        INSERT INTO posts (title, description, link, published_at)
        VALUES ($1, $2, $3, $4)
        ON CONFLICT (link) DO UPDATE
        SET title = EXCLUDED.title,
            description = EXCLUDED.description,
            published_at = EXCLUDED.published_at
    `)
	if err != nil {
		return fmt.Errorf("prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, post := range posts {
		if post.Link == "" {
			continue
		}
		if _, err = stmt.ExecContext(ctx, post.Title, post.Description, post.Link, post.PublishedAt); err != nil {
			return fmt.Errorf("insert post: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}
func (p *Postgres) Latest(ctx context.Context, limit int) ([]Post, error) {
	if limit <= 0 {
		return nil, errors.New("limit must be greater than zero")
	}

	rows, err := p.db.QueryContext(ctx, `
        SELECT id, title, description, link, published_at, created_at
        FROM posts
        ORDER BY published_at DESC
        LIMIT $1
    `, limit)
	if err != nil {
		return nil, fmt.Errorf("query latest: %w", err)
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var post Post
		if err := rows.Scan(&post.ID, &post.Title, &post.Description, &post.Link, &post.PublishedAt, &post.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan post: %w", err)
		}
		posts = append(posts, post)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return posts, nil
}
