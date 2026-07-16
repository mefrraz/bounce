package httpclient

import (
	"fmt"
	"io"
	"strings"
	"net/http"
	"time"
)

const (
	userAgent   = "Bounce/0.2 (+https://github.com/mefrraz/bounce)"
	maxRetries  = 3
	rateInterval = 1100 * time.Millisecond
)

type Client struct {
	http    *http.Client
	limiter *time.Ticker
}

func New() *Client {
	return &Client{
		http:    &http.Client{Timeout: 15 * time.Second},
		limiter: time.NewTicker(rateInterval),
	}
}

func (c *Client) Get(url string) ([]byte, error) {
	return c.doWithRetry(func() (*http.Response, error) {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", userAgent)
		req.Header.Set("Accept", "application/json, text/html")
		req.Header.Set("Referer", "https://www.fpb.pt/")
		return c.http.Do(req)
	})
}

func (c *Client) Post(url, body string) ([]byte, error) {
	return c.doWithRetry(func() (*http.Response, error) {
		req, err := http.NewRequest("POST", url, strings.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", userAgent)
		req.Header.Set("Accept", "application/json, text/html")
		req.Header.Set("Referer", "https://www.fpb.pt/")
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		return c.http.Do(req)
	})
}

// PostFast sends a POST without rate limiting (for internal score fetching).
func (c *Client) PostFast(url, body string) ([]byte, error) {
	req, err := http.NewRequest("POST", url, strings.NewReader(body))
	if err != nil { return nil, err }
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Referer", "https://www.fpb.pt/")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.http.Do(req)
	if err != nil { return nil, err }
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func (c *Client) doWithRetry(fetch func() (*http.Response, error)) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		<-c.limiter.C
		resp, err := fetch()
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(attempt+1) * 500 * time.Millisecond)
			continue
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(attempt+1) * 500 * time.Millisecond)
			continue
		}
		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("server error %d", resp.StatusCode)
			time.Sleep(time.Duration(attempt+1) * time.Second)
			continue
		}
		if resp.StatusCode == 404 {
			return nil, fmt.Errorf("404 not found")
		}
		return body, nil
	}
	return nil, fmt.Errorf("after %d retries: %w", maxRetries, lastErr)
}

func (c *Client) Stop() {
	c.limiter.Stop()
}
