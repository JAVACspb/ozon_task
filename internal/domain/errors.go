package domain

import "errors"

var (
	ErrNotFound         = errors.New("not found")
	ErrCommentsDisabled = errors.New("comments are disabled")
	ErrInvalidInput     = errors.New("invalid input")
	ErrForbidden        = errors.New("forbidden")
)

const MaxCommentBodyLen = 2000
