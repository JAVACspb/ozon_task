package pagination

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Cursor struct {
	CreatedAt time.Time
	ID        string
}

func EncodeCursor(createdAt time.Time, id string) string {
	raw := fmt.Sprintf("%d:%s", createdAt.UTC().UnixNano(), id)
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func DecodeCursor(cursor string) (Cursor, error) {
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return Cursor{}, fmt.Errorf("decode cursor: %w", err)
	}

	createdAtNano, id, ok := strings.Cut(string(raw), ":")
	if !ok || id == "" {
		return Cursor{}, fmt.Errorf("invalid cursor")
	}

	nano, err := strconv.ParseInt(createdAtNano, 10, 64)
	if err != nil {
		return Cursor{}, fmt.Errorf("parse cursor time: %w", err)
	}

	return Cursor{
		CreatedAt: time.Unix(0, nano).UTC(),
		ID:        id,
	}, nil
}
