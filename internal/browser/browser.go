// Package browser provides headless Chrome helpers for scraping
// JavaScript-rendered pages from FPB.
package browser

import (
	"context"
	"log"
	"time"

	"github.com/chromedp/chromedp"
)

// Client wraps a chromedp context for fetching JS-rendered pages.
type Client struct {
	allocCtx context.Context
}

// NewClient creates a headless Chrome client.
func NewClient() (*Client, error) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.NoSandbox,
		chromedp.DisableGPU,
		chromedp.Flag("headless", true),
	)

	allocCtx, _ := chromedp.NewExecAllocator(context.Background(), opts...)
	return &Client{allocCtx: allocCtx}, nil
}

// FetchHTML loads a URL in headless Chrome and returns the fully rendered HTML.
func (c *Client) FetchHTML(url string, waitVisible string) (string, error) {
	ctx, cancel := chromedp.NewContext(c.allocCtx)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	var html string
	tasks := []chromedp.Action{
		chromedp.Navigate(url),
		chromedp.Sleep(2 * time.Second), // wait for JS to render
	}

	if waitVisible != "" {
		tasks = append(tasks, chromedp.WaitVisible(waitVisible, chromedp.ByQuery))
	} else {
		tasks = append(tasks, chromedp.Sleep(2*time.Second))
	}

	tasks = append(tasks, chromedp.OuterHTML("html", &html))

	if err := chromedp.Run(ctx, tasks...); err != nil {
		return "", err
	}

	log.Printf("[browser] fetched %s (%d bytes)", url, len(html))
	return html, nil
}
