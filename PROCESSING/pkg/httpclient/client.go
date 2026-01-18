// Package httpclient provides a reusable HTTP client with retry logic.
package httpclient

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// Client is an HTTP client with retry and rate limiting capabilities.
type Client struct {
	client      *http.Client
	rateLimiter *RateLimiter
	maxRetries  int
	retryDelay  time.Duration
	logger      *zap.Logger
}

// Config holds HTTP client configuration.
type Config struct {
	Timeout     time.Duration
	RateLimit   int           // requests per second
	MaxRetries  int
	RetryDelay  time.Duration
}

// NewClient creates a new HTTP client with the given configuration.
func NewClient(cfg Config, logger *zap.Logger) *Client {
	return &Client{
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
		rateLimiter: NewRateLimiter(cfg.RateLimit),
		maxRetries:  cfg.MaxRetries,
		retryDelay:  cfg.RetryDelay,
		logger:      logger,
	}
}

// Get performs a GET request with retry logic.
func (c *Client) Get(ctx context.Context, url string) ([]byte, error) {
	return c.doWithRetry(ctx, http.MethodGet, url, nil)
}

// Post performs a POST request with retry logic.
func (c *Client) Post(ctx context.Context, url string, body io.Reader) ([]byte, error) {
	return c.doWithRetry(ctx, http.MethodPost, url, body)
}

// doWithRetry performs an HTTP request with retry logic.
func (c *Client) doWithRetry(ctx context.Context, method, url string, body io.Reader) ([]byte, error) {
	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			c.logger.Info("retrying request",
				zap.String("url", url),
				zap.Int("attempt", attempt),
			)
			time.Sleep(c.retryDelay * time.Duration(attempt))
		}

		// Wait for rate limiter
		if err := c.rateLimiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limiter error: %w", err)
		}

		data, err := c.do(ctx, method, url, body)
		if err == nil {
			return data, nil
		}

		lastErr = err
		c.logger.Warn("request failed",
			zap.String("url", url),
			zap.Int("attempt", attempt),
			zap.Error(err),
		)
	}

	return nil, fmt.Errorf("all retries exhausted: %w", lastErr)
}

// do performs a single HTTP request.
func (c *Client) do(ctx context.Context, method, url string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("User-Agent", "Pandora-PROCESSING/1.0")
	req.Header.Set("Accept", "application/json,application/xml")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	return data, nil
}

// RateLimiter implements a simple token bucket rate limiter.
type RateLimiter struct {
	tokens   chan struct{}
	interval time.Duration
}

// NewRateLimiter creates a new rate limiter with the specified rate.
func NewRateLimiter(requestsPerSecond int) *RateLimiter {
	if requestsPerSecond <= 0 {
		requestsPerSecond = 1
	}

	rl := &RateLimiter{
		tokens:   make(chan struct{}, requestsPerSecond),
		interval: time.Second / time.Duration(requestsPerSecond),
	}

	// Start token refiller
	go rl.refill()

	return rl
}

// Wait blocks until a token is available.
func (rl *RateLimiter) Wait(ctx context.Context) error {
	select {
	case <-rl.tokens:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// refill continuously adds tokens to the bucket.
func (rl *RateLimiter) refill() {
	ticker := time.NewTicker(rl.interval)
	defer ticker.Stop()

	for range ticker.C {
		select {
		case rl.tokens <- struct{}{}:
		default:
			// Bucket is full
		}
	}
}
