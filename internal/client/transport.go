// Package client provides HTTP client functionality for the API Gateway REST API.
package client

import (
	"fmt"
	"net/http"
	"time"

	"golang.org/x/time/rate"
)

// rateLimitTransport wraps an [http.RoundTripper] with rate limiting.
type rateLimitTransport struct {
	base    http.RoundTripper
	limiter *rate.Limiter
}

// newRateLimitTransport creates a new rate-limited transport.
func newRateLimitTransport(base http.RoundTripper, rps int) *rateLimitTransport {
	if base == nil {
		base = http.DefaultTransport
	}
	return &rateLimitTransport{
		base:    base,
		limiter: rate.NewLimiter(rate.Limit(rps), rps),
	}
}

// RoundTrip implements [http.RoundTripper] with rate limiting.
func (t *rateLimitTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Wait for rate limiter
	if err := t.limiter.Wait(req.Context()); err != nil {
		return nil, fmt.Errorf("rate limit wait: %w", err)
	}
	return t.base.RoundTrip(req)
}

// userAgentTransport wraps an [http.RoundTripper] to add User-Agent header.
type userAgentTransport struct {
	base      http.RoundTripper
	userAgent string
}

// newUserAgentTransport creates a new transport that adds User-Agent header.
func newUserAgentTransport(base http.RoundTripper, version string) *userAgentTransport {
	if base == nil {
		base = http.DefaultTransport
	}
	return &userAgentTransport{
		base:      base,
		userAgent: fmt.Sprintf("agwctl/%s", version),
	}
}

// RoundTrip implements [http.RoundTripper] with User-Agent injection.
func (t *userAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone request to avoid modifying original
	req = req.Clone(req.Context())
	req.Header.Set("User-Agent", t.userAgent)
	return t.base.RoundTrip(req)
}

// authTransport wraps an [http.RoundTripper] to add Basic Authentication.
type authTransport struct {
	base     http.RoundTripper
	username string
	password string
}

// newAuthTransport creates a new transport that adds Basic Auth.
func newAuthTransport(base http.RoundTripper, username, password string) *authTransport {
	if base == nil {
		base = http.DefaultTransport
	}
	return &authTransport{
		base:     base,
		username: username,
		password: password,
	}
}

// RoundTrip implements [http.RoundTripper] with Basic Authentication.
func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone request to avoid modifying original
	req = req.Clone(req.Context())
	req.SetBasicAuth(t.username, t.password)
	return t.base.RoundTrip(req)
}

// newOptimizedTransport creates a high-performance HTTP transport.
func newOptimizedTransport() *http.Transport {
	return &http.Transport{
		// Connection pooling
		MaxIdleConns:        64,
		MaxIdleConnsPerHost: 64,
		MaxConnsPerHost:     64,

		// Timeouts
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,

		// Keep-alive
		DisableKeepAlives: false,

		// Compression
		DisableCompression: false,

		// Context-aware dialing
		ForceAttemptHTTP2: true,
	}
}

// newTransport creates a layered transport with all decorators.
func newTransport(username, password, version string, rps int) http.RoundTripper {
	// Start with optimized base transport
	base := newOptimizedTransport()

	// Layer 1: Rate limiting (innermost)
	var transport http.RoundTripper = newRateLimitTransport(base, rps)

	// Layer 2: Authentication
	transport = newAuthTransport(transport, username, password)

	// Layer 3: User-Agent (outermost)
	return newUserAgentTransport(transport, version)
}

// Made with Bob
