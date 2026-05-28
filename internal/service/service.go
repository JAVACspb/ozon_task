package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ozosanek/ozon_task/internal/domain"
	"github.com/ozosanek/ozon_task/internal/storage"
)

const defaultPageSize = 20

type Service struct {
	storage       storage.Storage
	commentBroker *CommentBroker
}

func New(storage storage.Storage) *Service {
	return &Service{
		storage:       storage,
		commentBroker: NewCommentBroker(),
	}
}

func (s *Service) CreatePost(ctx context.Context, params storage.CreatePostParams) (domain.Post, error) {
	params.AuthorName = strings.TrimSpace(params.AuthorName)
	params.Title = strings.TrimSpace(params.Title)
	params.Body = strings.TrimSpace(params.Body)

	if params.AuthorName == "" || params.Title == "" || params.Body == "" {
		return domain.Post{}, fmt.Errorf("%w: author, title and body are required", domain.ErrInvalidInput)
	}

	return s.storage.CreatePost(ctx, params)
}

func (s *Service) ListPosts(ctx context.Context, opts domain.ListOptions) (domain.Page[domain.Post], error) {
	opts = normalizeListOptions(opts)
	return s.storage.ListPosts(ctx, opts)
}

func (s *Service) GetPost(ctx context.Context, id string) (domain.Post, error) {
	if strings.TrimSpace(id) == "" {
		return domain.Post{}, fmt.Errorf("%w: post id is required", domain.ErrInvalidInput)
	}

	return s.storage.GetPost(ctx, id)
}

func (s *Service) UpdatePostCommentsEnabled(ctx context.Context, postID string, authorName string, enabled bool) (domain.Post, error) {
	if strings.TrimSpace(postID) == "" {
		return domain.Post{}, fmt.Errorf("%w: post id is required", domain.ErrInvalidInput)
	}
	authorName = strings.TrimSpace(authorName)
	if authorName == "" {
		return domain.Post{}, fmt.Errorf("%w: author is required", domain.ErrInvalidInput)
	}

	post, err := s.storage.GetPost(ctx, postID)
	if err != nil {
		return domain.Post{}, err
	}
	if post.AuthorName != authorName {
		return domain.Post{}, fmt.Errorf("%w: only post author can change comments settings", domain.ErrForbidden)
	}

	return s.storage.UpdatePostCommentsEnabled(ctx, postID, enabled)
}

func (s *Service) CreateComment(ctx context.Context, params storage.CreateCommentParams) (domain.Comment, error) {
	params.PostID = strings.TrimSpace(params.PostID)
	params.AuthorName = strings.TrimSpace(params.AuthorName)
	params.Body = strings.TrimSpace(params.Body)

	if params.PostID == "" || params.AuthorName == "" || params.Body == "" {
		return domain.Comment{}, fmt.Errorf("%w: post id, author and body are required", domain.ErrInvalidInput)
	}
	if len([]rune(params.Body)) > domain.MaxCommentBodyLen {
		return domain.Comment{}, fmt.Errorf("%w: comment body is longer than %d chars", domain.ErrInvalidInput, domain.MaxCommentBodyLen)
	}

	post, err := s.storage.GetPost(ctx, params.PostID)
	if err != nil {
		return domain.Comment{}, err
	}
	if !post.CommentsEnabled {
		return domain.Comment{}, domain.ErrCommentsDisabled
	}

	comment, err := s.storage.CreateComment(ctx, params)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.Comment{}, fmt.Errorf("%w: parent comment not found", domain.ErrNotFound)
		}
		return domain.Comment{}, err
	}

	s.commentBroker.Publish(comment)

	return comment, nil
}

func (s *Service) ListComments(ctx context.Context, postID string, parentID *string, opts domain.ListOptions) (domain.Page[domain.Comment], error) {
	if strings.TrimSpace(postID) == "" {
		return domain.Page[domain.Comment]{}, fmt.Errorf("%w: post id is required", domain.ErrInvalidInput)
	}

	opts = normalizeListOptions(opts)
	return s.storage.ListComments(ctx, postID, parentID, opts)
}

func (s *Service) SubscribeComments(ctx context.Context, postID string) (<-chan *domain.Comment, error) {
	postID = strings.TrimSpace(postID)
	if postID == "" {
		return nil, fmt.Errorf("%w: post id is required", domain.ErrInvalidInput)
	}
	if _, err := s.storage.GetPost(ctx, postID); err != nil {
		return nil, err
	}

	return s.commentBroker.Subscribe(postID), nil
}

func normalizeListOptions(opts domain.ListOptions) domain.ListOptions {
	if opts.First <= 0 {
		opts.First = defaultPageSize
	}
	if opts.First > 100 {
		opts.First = 100
	}

	return opts
}
