package httpclient

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"net/http"
	"time"
)

const (
	userAgent    = "Bounce/0.2 (+https://github.com/mefrraz/bounce)"
	maxRetries   = 3
	rateInterval = 1100 * time.Millisecond
	reqTimeout   = 20 * time.Second
)

type Client struct {
	http    *http.Client
	limiter *time.Ticker
}

func New() *Client {
	return &Client{
		http: &http.Client{
			Timeout: reqTimeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 100,
				IdleConnTimeout:     90 * time.Second,
			},
		},
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

func (c *Client) doWithRetry(fetch func() (*http.Response, error)) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		<-c.limiter.C
		resp, err := fetch()
		if err != nil {
			// Don't retry on context/timeout errors — the server is overloaded
			if isTimeout(err) {
				return nil, err
			}
			lastErr = err
			time.Sleep(time.Duration(attempt+1) * 500 * time.Millisecond)
			continue
		}
		defer resp.Body.Close()
		// Read with timeout (fixes io.ReadAll hanging forever)
		body, err := readWithTimeout(resp.Body, reqTimeout)
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

func readWithTimeout(r io.Reader, timeout time.Duration) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	type result struct {
		data []byte
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		data, err := io.ReadAll(r)
		ch <- result{data, err}
	}()

	select {
	case res := <-ch:
		return res.data, res.err
	case <-ctx.Done():
		return nil, fmt.Errorf("read timeout after %v", timeout)
	}
}

func (c *Client) Stop() {
	c.limiter.Stop()
}

func isTimeout(err error) bool {
	if os.IsTimeout(err) { return true }
	if ne, ok := err.(net.Error); ok && ne.Timeout() { return true }
	return false
}

// FastMode switches to no rate limiter for bulk scraping.
func (c *Client) FastMode() {
	c.limiter.Stop()
	c.limiter = time.NewTicker(1 * time.Nanosecond)
}
