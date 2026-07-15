package browser

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/chromedp/chromedp"
)

type Client struct {
	allocCtx context.Context
}

func NewClient() (*Client, error) {
	opts := []chromedp.ExecAllocatorOption{
		chromedp.NoSandbox,
		chromedp.DisableGPU,
		chromedp.Flag("headless", "new"),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("no-zygote", true),
	}

	// Use CHROME_BIN env var or default paths
	chromePath := os.Getenv("CHROME_BIN")
	if chromePath != "" {
		opts = append(opts, chromedp.ExecPath(chromePath))
	}

	allocCtx, _ := chromedp.NewExecAllocator(context.Background(), opts...)
	return &Client{allocCtx: allocCtx}, nil
}

func (c *Client) FetchHTML(url string, waitVisible string) (string, error) {
	ctx, cancel := chromedp.NewContext(c.allocCtx)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 40*time.Second)
	defer cancel()

	var html string
	tasks := []chromedp.Action{
		chromedp.Navigate(url),
		chromedp.Sleep(3 * time.Second),
	}

	if waitVisible != "" {
		tasks = append(tasks, chromedp.WaitVisible(waitVisible, chromedp.ByQuery))
	}

	tasks = append(tasks,
		chromedp.Sleep(2*time.Second),
		chromedp.OuterHTML("html", &html),
	)

	if err := chromedp.Run(ctx, tasks...); err != nil {
		return "", err
	}

	log.Printf("[browser] fetched %s (%d bytes)", url, len(html))
	return html, nil
}
