package storage

import (
	"context"
	"io"

	"github.com/ozosanek/ozon_task/internal/domain"
)

type CreatePostParams struct {
	AuthorName      string
	Title           string
	Body            string
	CommentsEnabled bool
}

type CreateCommentParams struct {
	PostID     string
	ParentID   *string
	AuthorName string
	Body       string
}

type Storage interface {
	io.Closer

	CreatePost(ctx context.Context, params CreatePostParams) (domain.Post, error)
	ListPosts(ctx context.Context, opts domain.ListOptions) (domain.Page[domain.Post], error)
	GetPost(ctx context.Context, id string) (domain.Post, error)
	UpdatePostCommentsEnabled(ctx context.Context, postID string, enabled bool) (domain.Post, error)

	CreateComment(ctx context.Context, params CreateCommentParams) (domain.Comment, error)
	ListComments(ctx context.Context, postID string, parentID *string, opts domain.ListOptions) (domain.Page[domain.Comment], error)
}
