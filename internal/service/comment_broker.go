package service

import (
	"sync"

	"github.com/ozosanek/ozon_task/internal/domain"
)

type CommentBroker struct {
	mu          sync.RWMutex
	subscribers map[string]map[chan *domain.Comment]struct{}
}

func NewCommentBroker() *CommentBroker {
	return &CommentBroker{
		subscribers: make(map[string]map[chan *domain.Comment]struct{}),
	}
}

func (b *CommentBroker) Subscribe(postID string) <-chan *domain.Comment {
	ch := make(chan *domain.Comment, 1)

	b.mu.Lock()
	if b.subscribers[postID] == nil {
		b.subscribers[postID] = make(map[chan *domain.Comment]struct{})
	}
	b.subscribers[postID][ch] = struct{}{}
	b.mu.Unlock()

	return ch
}

func (b *CommentBroker) Publish(comment domain.Comment) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for ch := range b.subscribers[comment.PostID] {
		item := comment
		select {
		case ch <- &item:
		default:
		}
	}
}
