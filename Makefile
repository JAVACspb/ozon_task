APP_NAME := ozon-task
CONFIG ?= config/config.yaml
POSTGRES_DSN ?= postgres://postgres:postgres@localhost:5432/ozon_task?sslmode=disable

.PHONY: build run run-memory run-postgres test generate tidy docker-build docker-up docker-down migrate-up migrate-down dump-tables

build:
	go build -o bin/$(APP_NAME) ./cmd/server

run:
	go run ./cmd/server -config $(CONFIG)

run-memory:
	OZON_TASK_STORAGE_TYPE=memory go run ./cmd/server -config $(CONFIG)

run-postgres:
	OZON_TASK_STORAGE_TYPE=postgres OZON_TASK_POSTGRES_DSN="$(POSTGRES_DSN)" go run ./cmd/server -config $(CONFIG)

test:
	go test ./...

generate:
	go run github.com/99designs/gqlgen generate

tidy:
	go mod tidy

docker-build:
	docker build -t $(APP_NAME):latest .

docker-up:
	docker compose up --build

docker-down:
	docker compose down

migrate-up:
	go run github.com/pressly/goose/v3/cmd/goose -dir migrations postgres "$(POSTGRES_DSN)" up

migrate-down:
	go run github.com/pressly/goose/v3/cmd/goose -dir migrations postgres "$(POSTGRES_DSN)" down

dump-tables:
	sudo docker compose exec postgres psql -U postgres -d ozon_task -c "select id, title, comments_enabled from posts;"
	sudo docker compose exec postgres psql -U postgres -d ozon_task -c "select id, post_id, parent_id, body from comments order by id;"
