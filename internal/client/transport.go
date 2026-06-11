package client

import (
	"context"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"time"
)

// Authenticator decorates an outgoing request with authentication.
type Authenticator interface {
	Auth(req *http.Request) error
}

const (
	// maxRetries is how many times a failed request is re-sent.
	maxRetries = 2
	// retryBaseBackoff scales the linear wait between retry attempts;
	// retryJitter desynchronizes concurrent collectors retrying together.
	retryBaseBackoff = 250 * time.Millisecond
	retryJitter      = 100 * time.Millisecond
)

// defaultBackoff returns the wait before retry attempt n (1-based): linear
// growth gives a struggling target breathing room instead of back-to-back
// hits, and jitter keeps concurrent collectors from retrying in lockstep.
func defaultBackoff(attempt int) time.Duration {
	return time.Duration(attempt)*retryBaseBackoff + rand.N(retryJitter) //nolint:gosec // retry jitter, not cryptographic
}

// sleepContext waits for d or until ctx is canceled, whichever comes first.
func sleepContext(ctx context.Context, d time.Duration) {
	if d <= 0 {
		return
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}

// ExportarrTransport is an http.RoundTripper that authenticates requests and
// retries server errors.
type ExportarrTransport struct {
	inner http.RoundTripper
	auth  Authenticator
	// Backoff returns the wait before retry attempt n (1-based). Nil means
	// defaultBackoff; tests inject shorter schedules.
	Backoff func(attempt int) time.Duration
}

// NewExportarrTransport wraps inner with authentication and retries.
func NewExportarrTransport(inner http.RoundTripper, auth Authenticator) *ExportarrTransport {
	return &ExportarrTransport{
		inner: inner,
		auth:  auth,
	}
}

// RoundTrip implements http.RoundTripper. Server errors are retried twice
// with backoff; discarded responses are drained and closed so connections
// return to the pool instead of leaking.
func (t *ExportarrTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// RoundTrippers must not modify the caller's request; auth decorates a clone.
	req = req.Clone(req.Context())
	if t.auth != nil {
		if err := t.auth.Auth(req); err != nil {
			return nil, fmt.Errorf("error authenticating request: %w", err)
		}
	}

	backoff := t.Backoff
	if backoff == nil {
		backoff = defaultBackoff
	}
	resp, err := t.inner.RoundTrip(req)
	for attempt := 1; (err != nil || resp.StatusCode >= 500) && attempt <= maxRetries; attempt++ {
		drainBody(resp)
		sleepContext(req.Context(), backoff(attempt))
		if req.Context().Err() != nil {
			break
		}
		resp, err = t.inner.RoundTrip(req)
	}
	if err != nil {
		return nil, fmt.Errorf("error sending HTTP Request: %w", err)
	}
	switch {
	case resp.StatusCode >= 500:
		drainBody(resp)
		return nil, fmt.Errorf("received Server Error Status Code: %d", resp.StatusCode)
	case resp.StatusCode >= 400:
		drainBody(resp)
		return nil, fmt.Errorf("received Client Error Status Code: %d", resp.StatusCode)
	case resp.StatusCode >= 300:
		location, lerr := resp.Location()
		drainBody(resp)
		if lerr == nil {
			return nil, fmt.Errorf("received Redirect Status Code: %d, Location: %s", resp.StatusCode, location.String())
		}
		return nil, fmt.Errorf("received Redirect Status Code: %d, ", resp.StatusCode)
	}
	return resp, nil
}

// drainBody discards and closes a response body so the underlying connection
// can be reused.
func drainBody(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	_ = resp.Body.Close()
}
