package app

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/belyaev-v/task36/internal/rss"
	"github.com/belyaev-v/task36/internal/storage"
)

type App struct {
	feeds   []string
	period  time.Duration
	fetcher *rss.Fetcher
	store   storage.Store
	logger  *log.Logger
}

func New(feeds []string, period time.Duration, fetcher *rss.Fetcher, store storage.Store, logger *log.Logger) *App {
	return &App{
		feeds:   feeds,
		period:  period,
		fetcher: fetcher,
		store:   store,
		logger:  logger,
	}
}
func (a *App) Run(ctx context.Context) {
	if len(a.feeds) == 0 {
		a.logger.Println("no feeds configured, aggregator will not run")
		<-ctx.Done()
		return
	}

	a.logger.Printf("starting aggregator for %d feeds every %s", len(a.feeds), a.period)
	a.runOnce(ctx)

	ticker := time.NewTicker(a.period)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			a.logger.Println("aggregator stopped")
			return
		case <-ticker.C:
			a.runOnce(ctx)
		}
	}
}

func (a *App) runOnce(ctx context.Context) {
	postsCh := make(chan storage.Post)
	errCh := make(chan error)

	var wg sync.WaitGroup
	wg.Add(len(a.feeds))

	for _, feedURL := range a.feeds {
		go func(url string) {
			defer wg.Done()

			items, err := a.fetcher.Fetch(ctx, url)
			if err != nil {
				select {
				case errCh <- fmt.Errorf("fetch %s: %w", url, err):
				case <-ctx.Done():
				}
				return
			}

			for _, item := range items {
				post := storage.Post{
					Title:       item.Title,
					Description: item.Description,
					Link:        item.Link,
					PublishedAt: item.PublishedAt,
				}
				select {
				case postsCh <- post:
				case <-ctx.Done():
					return
				}
			}
		}(feedURL)
	}

	go func() {
		wg.Wait()
		close(postsCh)
		close(errCh)
	}()

	a.collect(ctx, postsCh, errCh)
}

func (a *App) collect(ctx context.Context, postsCh <-chan storage.Post, errCh <-chan error) {
	const batchSize = 25
	batch := make([]storage.Post, 0, batchSize)

	flush := func() {
		if len(batch) == 0 {
			return
		}
		if err := a.store.SavePosts(ctx, batch); err != nil {
			a.logger.Printf("save posts: %v", err)
		}
		batch = batch[:0]
	}

	for postsCh != nil || errCh != nil {
		select {
		case post, ok := <-postsCh:
			if !ok {
				postsCh = nil
				flush()
				continue
			}
			batch = append(batch, post)
			if len(batch) >= batchSize {
				flush()
			}
		case err, ok := <-errCh:
			if !ok {
				errCh = nil
				continue
			}
			a.logger.Printf("fetch feed: %v", err)
		case <-ctx.Done():
			return
		}
	}
}
