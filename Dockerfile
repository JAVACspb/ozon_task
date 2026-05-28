FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go install github.com/pressly/goose/v3/cmd/goose@v3.27.1
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/ozon-task ./cmd/server

FROM alpine:3.22

WORKDIR /app

COPY --from=builder /bin/ozon-task /app/ozon-task
COPY --from=builder /go/bin/goose /app/goose
COPY config/config.yaml /app/config/config.yaml
COPY migrations /app/migrations

EXPOSE 8080

ENTRYPOINT ["/app/ozon-task"]
CMD ["-config", "/app/config/config.yaml"]
