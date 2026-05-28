package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/ozosanek/ozon_task/internal/domain"
	"github.com/ozosanek/ozon_task/internal/storage"
	"github.com/ozosanek/ozon_task/internal/storage/memory"
	"github.com/stretchr/testify/require"
)

func TestCreateCommentRejectsDisabledPost(t *testing.T) {
	ctx := context.Background()
	svc := New(memory.New())

	post, err := svc.CreatePost(ctx, storage.CreatePostParams{
		AuthorName:      "author",
		Title:           "post",
		Body:            "body",
		CommentsEnabled: false,
	})
	require.NoError(t, err)

	_, err = svc.CreateComment(ctx, storage.CreateCommentParams{
		PostID:     post.ID,
		AuthorName: "reader",
		Body:       "comment",
	})
	require.ErrorIs(t, err, domain.ErrCommentsDisabled)
}

func TestCreateCommentRejectsLongBody(t *testing.T) {
	ctx := context.Background()
	svc := New(memory.New())

	post, err := svc.CreatePost(ctx, storage.CreatePostParams{
		AuthorName:      "author",
		Title:           "post",
		Body:            "body",
		CommentsEnabled: true,
	})
	require.NoError(t, err)

	_, err = svc.CreateComment(ctx, storage.CreateCommentParams{
		PostID:     post.ID,
		AuthorName: "reader",
		Body:       strings.Repeat("a", domain.MaxCommentBodyLen+1),
	})
	require.True(t, errors.Is(err, domain.ErrInvalidInput))
}

func TestCreateCommentTrimsInput(t *testing.T) {
	ctx := context.Background()
	svc := New(memory.New())

	post, err := svc.CreatePost(ctx, storage.CreatePostParams{
		AuthorName:      " author ",
		Title:           " post ",
		Body:            " body ",
		CommentsEnabled: true,
	})
	require.NoError(t, err)
	require.Equal(t, "author", post.AuthorName)

	comment, err := svc.CreateComment(ctx, storage.CreateCommentParams{
		PostID:     post.ID,
		AuthorName: " reader ",
		Body:       " comment ",
	})
	require.NoError(t, err)
	require.Equal(t, "reader", comment.AuthorName)
	require.Equal(t, "comment", comment.Body)
}

func TestUpdatePostCommentsEnabledRequiresAuthor(t *testing.T) {
	ctx := context.Background()
	svc := New(memory.New())

	post, err := svc.CreatePost(ctx, storage.CreatePostParams{
		AuthorName:      "author",
		Title:           "post",
		Body:            "body",
		CommentsEnabled: true,
	})
	require.NoError(t, err)

	_, err = svc.UpdatePostCommentsEnabled(ctx, post.ID, "other", false)
	require.ErrorIs(t, err, domain.ErrForbidden)

	updated, err := svc.UpdatePostCommentsEnabled(ctx, post.ID, " author ", false)
	require.NoError(t, err)
	require.False(t, updated.CommentsEnabled)
}

func TestSubscribeCommentsReceivesCreatedComment(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	svc := New(memory.New())
	post, err := svc.CreatePost(ctx, storage.CreatePostParams{
		AuthorName:      "author",
		Title:           "post",
		Body:            "body",
		CommentsEnabled: true,
	})
	require.NoError(t, err)

	comments, err := svc.SubscribeComments(ctx, post.ID)
	require.NoError(t, err)

	created, err := svc.CreateComment(ctx, storage.CreateCommentParams{
		PostID:     post.ID,
		AuthorName: "reader",
		Body:       "comment",
	})
	require.NoError(t, err)

	select {
	case got := <-comments:
		require.NotNil(t, got)
		require.Equal(t, created.ID, got.ID)
		require.Equal(t, post.ID, got.PostID)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for comment")
	}
}
