package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gorilla/websocket"
	"github.com/ozosanek/ozon_task/internal/config"
	"github.com/ozosanek/ozon_task/internal/graph"
	"github.com/ozosanek/ozon_task/internal/graph/generated"
	"github.com/ozosanek/ozon_task/internal/service"
	"github.com/ozosanek/ozon_task/internal/storage"
	"github.com/ozosanek/ozon_task/internal/storage/memory"
	"github.com/ozosanek/ozon_task/internal/storage/postgres"
	"go.uber.org/zap"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "server failed: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	configPath := flag.String("config", "config/config.yaml", "path to yaml config")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}

	logger, err := config.NewLogger(cfg.Logger)
	if err != nil {
		return err
	}
	defer logger.Sync()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	store, err := newStorage(ctx, cfg)
	if err != nil {
		return err
	}
	defer store.Close()

	svc := service.New(store)
	resolver := graph.NewResolver(svc)

	mux := http.NewServeMux()
	mux.Handle("/query", newGraphQLHandler(resolver))
	if cfg.Server.Playground {
		mux.Handle("/", playground.Handler("GraphQL playground", "/query"))
	}

	server := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return serve(ctx, server, logger)
}

func newGraphQLHandler(resolver *graph.Resolver) http.Handler {
	srv := handler.New(generated.NewExecutableSchema(generated.Config{Resolvers: resolver}))
	srv.AddTransport(transport.Websocket{
		KeepAlivePingInterval: 10 * time.Second,
		Upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	})
	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.MultipartForm{})
	srv.Use(extension.Introspection{})
	srv.Use(extension.FixedComplexityLimit(1000))

	return srv
}

func newStorage(ctx context.Context, cfg config.Config) (storage.Storage, error) {
	switch cfg.Storage.Type {
	case "memory":
		return memory.New(), nil
	case "postgres":
		return postgres.New(ctx, cfg.Postgres.DSN)
	default:
		return nil, fmt.Errorf("unknown storage type %q", cfg.Storage.Type)
	}
}

func serve(ctx context.Context, server *http.Server, logger *zap.Logger) error {
	errCh := make(chan error, 1)

	var wg sync.WaitGroup
	wg.Go(func() {
		logger.Info("starting http server", zap.String("addr", server.Addr))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	})

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown server: %w", err)
	}
	wg.Wait()
	logger.Info("http server stopped")

	return nil
}
