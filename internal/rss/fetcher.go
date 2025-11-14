package rss

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Item struct {
	Title       string
	Description string
	Link        string
	PublishedAt time.Time
}
type Fetcher struct {
	client *http.Client
}

func NewFetcher(client *http.Client) *Fetcher {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return &Fetcher{client: client}
}
func (f *Fetcher) Fetch(ctx context.Context, url string) ([]Item, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request feed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	var feed rssFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, fmt.Errorf("parse rss: %w", err)
	}

	items := make([]Item, 0, len(feed.Channel.Items))
	for _, it := range feed.Channel.Items {
		pubDate, _ := parsePubDate(it.PubDate)
		items = append(items, Item{
			Title:       strings.TrimSpace(it.Title),
			Description: strings.TrimSpace(it.Description),
			Link:        strings.TrimSpace(it.Link),
			PublishedAt: pubDate,
		})
	}

	return items, nil
}

type rssFeed struct {
	Channel struct {
		Items []rssItem `xml:"item"`
	} `xml:"channel"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Description string `xml:"description"`
	Link        string `xml:"link"`
	PubDate     string `xml:"pubDate"`
}

func parsePubDate(value string) (time.Time, error) {
	layouts := []string{
		time.RFC1123Z,
		time.RFC1123,
		time.RFC822Z,
		time.RFC822,
		time.RFC3339,
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, value); err == nil {
			return t, nil
		}
	}
	return time.Now().UTC(), fmt.Errorf("unknown pubDate format: %s", value)
}
