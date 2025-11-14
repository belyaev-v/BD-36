package main

import (
	"context"
	"database/sql"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	_ "github.com/lib/pq"

	"github.com/belyaev-v/task36/internal/api"
	"github.com/belyaev-v/task36/internal/app"
	"github.com/belyaev-v/task36/internal/config"
	"github.com/belyaev-v/task36/internal/rss"
	"github.com/belyaev-v/task36/internal/storage"
)

func main() {
	var (
		configPath = flag.String("config", "config.json", "path to configuration file")
		webDir     = flag.String("web", "webapp", "directory with web assets")
	)
	flag.Parse()

	logger := log.New(os.Stdout, "gonews ", log.LstdFlags|log.Lmsgprefix)

	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Fatalf("load config: %v", err)
	}

	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		logger.Fatalf("open database: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := db.PingContext(ctx); err != nil {
		cancel()
		logger.Fatalf("ping database: %v", err)
	}
	cancel()

	store := storage.NewPostgres(db)
	fetcher := rss.NewFetcher(nil)
	aggregator := app.New(cfg.RSS, cfg.RequestPeriod, fetcher, store, logger)

	staticDir, err := filepath.Abs(*webDir)
	if err != nil {
		logger.Fatalf("resolve web directory: %v", err)
	}
	fs := http.FileServer(http.Dir(staticDir))
	apiServer := api.New(store, fs)

	srv := &http.Server{
		Addr:    cfg.APIHost,
		Handler: apiServer.Handler(),
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go aggregator.Run(ctx)

	go func() {
		logger.Printf("HTTP server listening on %s", cfg.APIHost)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("http server: %v", err)
		}
	}()

	<-ctx.Done()
	logger.Println("shutting down")

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Printf("server shutdown: %v", err)
	}
	if err := db.Close(); err != nil {
		logger.Printf("close db: %v", err)
	}
}
