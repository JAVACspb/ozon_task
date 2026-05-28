package postgres

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ozosanek/ozon_task/internal/domain"
	"github.com/ozosanek/ozon_task/internal/pagination"
	"github.com/ozosanek/ozon_task/internal/storage"
)

type Storage struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, dsn string) (*Storage, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("create pg pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return &Storage{pool: pool}, nil
}

func (s *Storage) Close() error {
	s.pool.Close()
	return nil
}

func (s *Storage) CreatePost(ctx context.Context, params storage.CreatePostParams) (domain.Post, error) {
	const query = `
		INSERT INTO posts (author_name, title, body, comments_enabled)
		VALUES ($1, $2, $3, $4)
		RETURNING id, author_name, title, body, comments_enabled, created_at, updated_at`

	return scanPost(s.pool.QueryRow(ctx, query, params.AuthorName, params.Title, params.Body, params.CommentsEnabled))
}

func (s *Storage) ListPosts(ctx context.Context, opts domain.ListOptions) (domain.Page[domain.Post], error) {
	createdAt, id, err := decodeAfter(opts.After)
	if err != nil {
		return domain.Page[domain.Post]{}, err
	}

	const query = `
		SELECT id, author_name, title, body, comments_enabled, created_at, updated_at
		FROM posts
		WHERE ($1::timestamptz IS NULL OR (created_at, id) < ($1::timestamptz, $2::bigint))
		ORDER BY created_at DESC, id DESC
		LIMIT $3`

	rows, err := s.pool.Query(ctx, query, createdAt, id, opts.First+1)
	if err != nil {
		return domain.Page[domain.Post]{}, fmt.Errorf("query posts: %w", err)
	}
	defer rows.Close()

	var posts []domain.Post
	for rows.Next() {
		post, err := scanPost(rows)
		if err != nil {
			return domain.Page[domain.Post]{}, err
		}
		posts = append(posts, post)
	}
	if err := rows.Err(); err != nil {
		return domain.Page[domain.Post]{}, fmt.Errorf("read posts rows: %w", err)
	}

	return pagePosts(posts, opts.First), nil
}

func (s *Storage) GetPost(ctx context.Context, id string) (domain.Post, error) {
	postID, err := parseID(id)
	if err != nil {
		return domain.Post{}, err
	}

	const query = `
		SELECT id, author_name, title, body, comments_enabled, created_at, updated_at
		FROM posts
		WHERE id = $1`

	return scanPost(s.pool.QueryRow(ctx, query, postID))
}

func (s *Storage) UpdatePostCommentsEnabled(ctx context.Context, postID string, enabled bool) (domain.Post, error) {
	id, err := parseID(postID)
	if err != nil {
		return domain.Post{}, err
	}

	const query = `
		UPDATE posts
		SET comments_enabled = $2, updated_at = now()
		WHERE id = $1
		RETURNING id, author_name, title, body, comments_enabled, created_at, updated_at`

	return scanPost(s.pool.QueryRow(ctx, query, id, enabled))
}

func (s *Storage) CreateComment(ctx context.Context, params storage.CreateCommentParams) (domain.Comment, error) {
	postID, err := parseID(params.PostID)
	if err != nil {
		return domain.Comment{}, err
	}

	var parentID any
	if params.ParentID != nil {
		id, err := parseID(*params.ParentID)
		if err != nil {
			return domain.Comment{}, err
		}
		parentID = id
	}

	const query = `
		INSERT INTO comments (post_id, parent_id, author_name, body)
		VALUES ($1, $2, $3, $4)
		RETURNING id, post_id, parent_id, author_name, body, created_at`

	comment, err := scanComment(s.pool.QueryRow(ctx, query, postID, parentID, params.AuthorName, params.Body))
	if err != nil {
		return domain.Comment{}, mapPgError(err)
	}

	return comment, nil
}

func (s *Storage) ListComments(ctx context.Context, postID string, parentID *string, opts domain.ListOptions) (domain.Page[domain.Comment], error) {
	post, err := parseID(postID)
	if err != nil {
		return domain.Page[domain.Comment]{}, err
	}

	var parent any
	if parentID != nil {
		id, err := parseID(*parentID)
		if err != nil {
			return domain.Page[domain.Comment]{}, err
		}
		parent = id
	}

	createdAt, id, err := decodeAfter(opts.After)
	if err != nil {
		return domain.Page[domain.Comment]{}, err
	}

	const query = `
		SELECT id, post_id, parent_id, author_name, body, created_at
		FROM comments
		WHERE post_id = $1
			AND (($2::bigint IS NULL AND parent_id IS NULL) OR parent_id = $2::bigint)
			AND ($3::timestamptz IS NULL OR (created_at, id) > ($3::timestamptz, $4::bigint))
		ORDER BY created_at ASC, id ASC
		LIMIT $5`

	rows, err := s.pool.Query(ctx, query, post, parent, createdAt, id, opts.First+1)
	if err != nil {
		return domain.Page[domain.Comment]{}, fmt.Errorf("query comments: %w", err)
	}
	defer rows.Close()

	var comments []domain.Comment
	for rows.Next() {
		comment, err := scanComment(rows)
		if err != nil {
			return domain.Page[domain.Comment]{}, err
		}
		comments = append(comments, comment)
	}
	if err := rows.Err(); err != nil {
		return domain.Page[domain.Comment]{}, fmt.Errorf("read comments rows: %w", err)
	}

	return pageComments(comments, opts.First), nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanPost(row rowScanner) (domain.Post, error) {
	var post domain.Post
	var id int64

	err := row.Scan(&id, &post.AuthorName, &post.Title, &post.Body, &post.CommentsEnabled, &post.CreatedAt, &post.UpdatedAt)
	if err != nil {
		return domain.Post{}, mapPgError(err)
	}

	post.ID = strconv.FormatInt(id, 10)
	return post, nil
}

func scanComment(row rowScanner) (domain.Comment, error) {
	var comment domain.Comment
	var id int64
	var postID int64
	var parentID *int64

	err := row.Scan(&id, &postID, &parentID, &comment.AuthorName, &comment.Body, &comment.CreatedAt)
	if err != nil {
		return domain.Comment{}, mapPgError(err)
	}

	comment.ID = strconv.FormatInt(id, 10)
	comment.PostID = strconv.FormatInt(postID, 10)
	if parentID != nil {
		parent := strconv.FormatInt(*parentID, 10)
		comment.ParentID = &parent
	}

	return comment, nil
}

func pagePosts(posts []domain.Post, first int) domain.Page[domain.Post] {
	hasNext := len(posts) > first
	if hasNext {
		posts = posts[:first]
	}

	var endCursor *string
	if len(posts) > 0 {
		cursor := pagination.EncodeCursor(posts[len(posts)-1].CreatedAt, posts[len(posts)-1].ID)
		endCursor = &cursor
	}

	return domain.Page[domain.Post]{
		Items:       posts,
		HasNextPage: hasNext,
		EndCursor:   endCursor,
	}
}

func pageComments(comments []domain.Comment, first int) domain.Page[domain.Comment] {
	hasNext := len(comments) > first
	if hasNext {
		comments = comments[:first]
	}

	var endCursor *string
	if len(comments) > 0 {
		cursor := pagination.EncodeCursor(comments[len(comments)-1].CreatedAt, comments[len(comments)-1].ID)
		endCursor = &cursor
	}

	return domain.Page[domain.Comment]{
		Items:       comments,
		HasNextPage: hasNext,
		EndCursor:   endCursor,
	}
}

func decodeAfter(after *string) (*time.Time, *int64, error) {
	if after == nil {
		return nil, nil, nil
	}

	cursor, err := pagination.DecodeCursor(*after)
	if err != nil {
		return nil, nil, err
	}

	id, err := parseID(cursor.ID)
	if err != nil {
		return nil, nil, err
	}

	return &cursor.CreatedAt, &id, nil
}

func parseID(id string) (int64, error) {
	value, err := strconv.ParseInt(id, 10, 64)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%w: invalid id", domain.ErrInvalidInput)
	}

	return value, nil
}

func mapPgError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23503" {
		return domain.ErrNotFound
	}

	return err
}
