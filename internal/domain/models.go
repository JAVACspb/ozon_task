package domain

import "time"

type Post struct {
	ID              string
	AuthorName      string
	Title           string
	Body            string
	CommentsEnabled bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type Comment struct {
	ID         string
	PostID     string
	ParentID   *string
	AuthorName string
	Body       string
	CreatedAt  time.Time
}

type Page[T any] struct {
	Items       []T
	HasNextPage bool
	EndCursor   *string
}

type ListOptions struct {
	First int
	After *string
}
