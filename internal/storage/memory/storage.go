package memory

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/ozosanek/ozon_task/internal/domain"
	"github.com/ozosanek/ozon_task/internal/pagination"
	"github.com/ozosanek/ozon_task/internal/storage"
)

type Storage struct {
	mu            sync.RWMutex
	posts         map[string]domain.Post
	comments      map[string]domain.Comment
	nextPostID    int64
	nextCommentID int64
	now           func() time.Time
}

func New() *Storage {
	return &Storage{
		posts:    make(map[string]domain.Post),
		comments: make(map[string]domain.Comment),
		now:      time.Now,
	}
}

func (s *Storage) Close() error {
	return nil
}

func (s *Storage) CreatePost(_ context.Context, params storage.CreatePostParams) (domain.Post, error) {
	return withLock(&s.mu, func() (domain.Post, error) {
		s.nextPostID++
		now := s.now().UTC()
		post := domain.Post{
			ID:              strconv.FormatInt(s.nextPostID, 10),
			AuthorName:      params.AuthorName,
			Title:           params.Title,
			Body:            params.Body,
			CommentsEnabled: params.CommentsEnabled,
			CreatedAt:       now,
			UpdatedAt:       now,
		}
		s.posts[post.ID] = post

		return post, nil
	})
}

func (s *Storage) ListPosts(_ context.Context, opts domain.ListOptions) (domain.Page[domain.Post], error) {
	return withLock(s.mu.RLocker(), func() (domain.Page[domain.Post], error) {
		items := make([]domain.Post, 0, len(s.posts))
		for _, post := range s.posts {
			items = append(items, post)
		}

		sort.Slice(items, func(i, j int) bool {
			if items[i].CreatedAt.Equal(items[j].CreatedAt) {
				return numericID(items[i].ID) > numericID(items[j].ID)
			}
			return items[i].CreatedAt.After(items[j].CreatedAt)
		})

		if opts.After != nil {
			cursor, err := pagination.DecodeCursor(*opts.After)
			if err != nil {
				return domain.Page[domain.Post]{}, err
			}
			items, err = filterPostsAfter(items, cursor)
			if err != nil {
				return domain.Page[domain.Post]{}, err
			}
		}

		return pagePosts(items, opts.First), nil
	})
}

func (s *Storage) GetPost(_ context.Context, id string) (domain.Post, error) {
	return withLock(s.mu.RLocker(), func() (domain.Post, error) {
		post, ok := s.posts[id]
		if !ok {
			return domain.Post{}, domain.ErrNotFound
		}

		return post, nil
	})
}

func (s *Storage) UpdatePostCommentsEnabled(_ context.Context, postID string, enabled bool) (domain.Post, error) {
	return withLock(&s.mu, func() (domain.Post, error) {
		post, ok := s.posts[postID]
		if !ok {
			return domain.Post{}, domain.ErrNotFound
		}

		post.CommentsEnabled = enabled
		post.UpdatedAt = s.now().UTC()
		s.posts[postID] = post

		return post, nil
	})
}

func (s *Storage) CreateComment(_ context.Context, params storage.CreateCommentParams) (domain.Comment, error) {
	return withLock(&s.mu, func() (domain.Comment, error) {
		if _, ok := s.posts[params.PostID]; !ok {
			return domain.Comment{}, domain.ErrNotFound
		}
		if params.ParentID != nil {
			parent, ok := s.comments[*params.ParentID]
			if !ok || parent.PostID != params.PostID {
				return domain.Comment{}, domain.ErrNotFound
			}
		}

		s.nextCommentID++
		comment := domain.Comment{
			ID:         strconv.FormatInt(s.nextCommentID, 10),
			PostID:     params.PostID,
			ParentID:   params.ParentID,
			AuthorName: params.AuthorName,
			Body:       params.Body,
			CreatedAt:  s.now().UTC(),
		}
		s.comments[comment.ID] = comment

		return comment, nil
	})
}

func (s *Storage) ListComments(_ context.Context, postID string, parentID *string, opts domain.ListOptions) (domain.Page[domain.Comment], error) {
	return withLock(s.mu.RLocker(), func() (domain.Page[domain.Comment], error) {
		if _, ok := s.posts[postID]; !ok {
			return domain.Page[domain.Comment]{}, domain.ErrNotFound
		}

		items := make([]domain.Comment, 0)
		for _, comment := range s.comments {
			if comment.PostID == postID && sameParent(comment.ParentID, parentID) {
				items = append(items, comment)
			}
		}

		sort.Slice(items, func(i, j int) bool {
			if items[i].CreatedAt.Equal(items[j].CreatedAt) {
				return numericID(items[i].ID) < numericID(items[j].ID)
			}
			return items[i].CreatedAt.Before(items[j].CreatedAt)
		})

		if opts.After != nil {
			cursor, err := pagination.DecodeCursor(*opts.After)
			if err != nil {
				return domain.Page[domain.Comment]{}, err
			}
			items, err = filterCommentsAfter(items, cursor)
			if err != nil {
				return domain.Page[domain.Comment]{}, err
			}
		}

		return pageComments(items, opts.First), nil
	})
}

type locker interface {
	Lock()
	Unlock()
}

func withLock[T any](locker locker, fn func() (T, error)) (T, error) {
	locker.Lock()
	defer locker.Unlock()

	return fn()
}

func pagePosts(items []domain.Post, first int) domain.Page[domain.Post] {
	hasNext := len(items) > first
	if hasNext {
		items = items[:first]
	}

	var endCursor *string
	if len(items) > 0 {
		cursor := pagination.EncodeCursor(items[len(items)-1].CreatedAt, items[len(items)-1].ID)
		endCursor = &cursor
	}

	return domain.Page[domain.Post]{
		Items:       items,
		HasNextPage: hasNext,
		EndCursor:   endCursor,
	}
}

func pageComments(items []domain.Comment, first int) domain.Page[domain.Comment] {
	hasNext := len(items) > first
	if hasNext {
		items = items[:first]
	}

	var endCursor *string
	if len(items) > 0 {
		cursor := pagination.EncodeCursor(items[len(items)-1].CreatedAt, items[len(items)-1].ID)
		endCursor = &cursor
	}

	return domain.Page[domain.Comment]{
		Items:       items,
		HasNextPage: hasNext,
		EndCursor:   endCursor,
	}
}

func filterPostsAfter(items []domain.Post, cursor pagination.Cursor) ([]domain.Post, error) {
	cursorID, err := parseNumericID(cursor.ID)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid cursor", domain.ErrInvalidInput)
	}

	for i, post := range items {
		postID, err := parseNumericID(post.ID)
		if err != nil {
			return nil, err
		}
		if post.CreatedAt.Before(cursor.CreatedAt) ||
			(post.CreatedAt.Equal(cursor.CreatedAt) && postID < cursorID) {
			return items[i:], nil
		}
	}

	return nil, nil
}

func filterCommentsAfter(items []domain.Comment, cursor pagination.Cursor) ([]domain.Comment, error) {
	cursorID, err := parseNumericID(cursor.ID)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid cursor", domain.ErrInvalidInput)
	}

	for i, comment := range items {
		commentID, err := parseNumericID(comment.ID)
		if err != nil {
			return nil, err
		}
		if comment.CreatedAt.After(cursor.CreatedAt) ||
			(comment.CreatedAt.Equal(cursor.CreatedAt) && commentID > cursorID) {
			return items[i:], nil
		}
	}

	return nil, nil
}

func sameParent(a *string, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}

	return *a == *b
}

func numericID(id string) int64 {
	value, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return 0
	}

	return value
}

func parseNumericID(id string) (int64, error) {
	value, err := strconv.ParseInt(id, 10, 64)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%w: invalid id", domain.ErrInvalidInput)
	}

	return value, nil
}
