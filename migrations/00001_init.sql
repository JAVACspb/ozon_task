-- +goose Up
CREATE TABLE posts (
    id BIGSERIAL PRIMARY KEY,
    author_name TEXT NOT NULL,
    title TEXT NOT NULL,
    body TEXT NOT NULL,
    comments_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE comments (
    id BIGSERIAL PRIMARY KEY,
    post_id BIGINT NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    parent_id BIGINT,
    author_name TEXT NOT NULL,
    body TEXT NOT NULL CHECK (char_length(body) <= 2000),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT comments_post_id_id_unique UNIQUE (post_id, id),
    CONSTRAINT comments_parent_same_post_fk
        FOREIGN KEY (post_id, parent_id)
        REFERENCES comments(post_id, id)
        ON DELETE CASCADE
);

CREATE INDEX posts_created_at_id_idx
    ON posts (created_at DESC, id DESC);

CREATE INDEX comments_roots_page_idx
    ON comments (post_id, created_at ASC, id ASC)
    WHERE parent_id IS NULL;

CREATE INDEX comments_replies_page_idx
    ON comments (post_id, parent_id, created_at ASC, id ASC);

-- +goose Down
DROP TABLE IF EXISTS comments;
DROP TABLE IF EXISTS posts;
