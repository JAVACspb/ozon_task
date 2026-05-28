package memory

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ozosanek/ozon_task/internal/domain"
	"github.com/ozosanek/ozon_task/internal/pagination"
	"github.com/ozosanek/ozon_task/internal/storage"
	"github.com/stretchr/testify/require"
)

func TestListPostsUsesCursorPagination(t *testing.T) {
	ctx := context.Background()
	store := New()
	now := time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC)
	store.now = func() time.Time {
		now = now.Add(time.Second)
		return now
	}

	first, err := store.CreatePost(ctx, storage.CreatePostParams{
		AuthorName:      "author",
		Title:           "first",
		Body:            "body",
		CommentsEnabled: true,
	})
	require.NoError(t, err)
	second, err := store.CreatePost(ctx, storage.CreatePostParams{
		AuthorName:      "author",
		Title:           "second",
		Body:            "body",
		CommentsEnabled: true,
	})
	require.NoError(t, err)
	third, err := store.CreatePost(ctx, storage.CreatePostParams{
		AuthorName:      "author",
		Title:           "third",
		Body:            "body",
		CommentsEnabled: true,
	})
	require.NoError(t, err)

	page, err := store.ListPosts(ctx, domain.ListOptions{First: 2})
	require.NoError(t, err)
	require.True(t, page.HasNextPage)
	require.Len(t, page.Items, 2)
	require.Equal(t, []string{third.ID, second.ID}, []string{page.Items[0].ID, page.Items[1].ID})
	require.NotNil(t, page.EndCursor)

	nextPage, err := store.ListPosts(ctx, domain.ListOptions{First: 2, After: page.EndCursor})
	require.NoError(t, err)
	require.False(t, nextPage.HasNextPage)
	require.Len(t, nextPage.Items, 1)
	require.Equal(t, first.ID, nextPage.Items[0].ID)
}

func TestListCommentsSeparatesRootsAndReplies(t *testing.T) {
	ctx := context.Background()
	store := New()

	post, err := store.CreatePost(ctx, storage.CreatePostParams{
		AuthorName:      "author",
		Title:           "post",
		Body:            "body",
		CommentsEnabled: true,
	})
	require.NoError(t, err)

	root, err := store.CreateComment(ctx, storage.CreateCommentParams{
		PostID:     post.ID,
		AuthorName: "reader",
		Body:       "root",
	})
	require.NoError(t, err)

	reply, err := store.CreateComment(ctx, storage.CreateCommentParams{
		PostID:     post.ID,
		ParentID:   &root.ID,
		AuthorName: "reader",
		Body:       "reply",
	})
	require.NoError(t, err)

	roots, err := store.ListComments(ctx, post.ID, nil, domain.ListOptions{First: 10})
	require.NoError(t, err)
	require.Len(t, roots.Items, 1)
	require.Equal(t, root.ID, roots.Items[0].ID)

	replies, err := store.ListComments(ctx, post.ID, &root.ID, domain.ListOptions{First: 10})
	require.NoError(t, err)
	require.Len(t, replies.Items, 1)
	require.Equal(t, reply.ID, replies.Items[0].ID)
}

func TestListPostsRejectsInvalidCursor(t *testing.T) {
	ctx := context.Background()
	store := New()

	_, err := store.CreatePost(ctx, storage.CreatePostParams{
		AuthorName:      "author",
		Title:           "post",
		Body:            "body",
		CommentsEnabled: true,
	})
	require.NoError(t, err)

	cursor := pagination.EncodeCursor(time.Now(), "not-number")
	_, err = store.ListPosts(ctx, domain.ListOptions{First: 10, After: &cursor})
	require.True(t, errors.Is(err, domain.ErrInvalidInput))
}
