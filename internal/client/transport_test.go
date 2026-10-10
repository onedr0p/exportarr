package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/onedr0p/exportarr/internal/assert"
)

// scriptedTransport returns canned status codes in order, recording the time
// of every attempt; the last status repeats once the script is exhausted.
type scriptedTransport struct {
	mu       sync.Mutex
	statuses []int
	attempts []time.Time
}

func (s *scriptedTransport) RoundTrip(*http.Request) (*http.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	status := s.statuses[min(len(s.attempts), len(s.statuses)-1)]
	s.attempts = append(s.attempts, time.Now())
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader("{}")),
		Header:     make(http.Header),
	}, nil
}

func TestRoundTrip_BacksOffBetweenRetries(t *testing.T) {
	inner := &scriptedTransport{statuses: []int{http.StatusInternalServerError, http.StatusOK}}
	transport := NewExportarrTransport(inner, nil)
	const wait = 25 * time.Millisecond
	transport.Backoff = func(int) time.Duration { return wait }

	req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
	assert.NoError(t, err)
	resp, err := transport.RoundTrip(req)
	assert.NoError(t, err)
	assert.Equal(t, resp.StatusCode, http.StatusOK)
	_ = resp.Body.Close()

	assert.Len(t, inner.attempts, 2)
	assert.GreaterOrEqual(t, inner.attempts[1].Sub(inner.attempts[0]), wait)
}

func TestRoundTrip_BackoffRespectsContextCancel(t *testing.T) {
	inner := &scriptedTransport{statuses: []int{http.StatusInternalServerError}}
	transport := NewExportarrTransport(inner, nil)
	transport.Backoff = func(int) time.Duration { return time.Hour }

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://example.com", nil)
	assert.NoError(t, err)

	start := time.Now()
	_, err = transport.RoundTrip(req)
	assert.Error(t, err)
	assert.Len(t, inner.attempts, 1, "no retry should happen after cancellation")
	assert.True(t, time.Since(start) < time.Minute, "canceled backoff should return promptly")
}

func TestDefaultBackoff_GrowsWithAttempts(t *testing.T) {
	for attempt := 1; attempt <= maxRetries; attempt++ {
		d := defaultBackoff(attempt)
		assert.GreaterOrEqual(t, d, time.Duration(attempt)*retryBaseBackoff)
		assert.True(t, d < time.Duration(attempt)*retryBaseBackoff+retryJitter, "jitter exceeds bound")
	}
}

type trackingBody struct {
	io.Reader
	mu     sync.Mutex
	closed bool
}

func (b *trackingBody) Read(p []byte) (n int, err error) {
	return b.Reader.Read(p)
}

func (b *trackingBody) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.closed = true
	return nil
}

func (b *trackingBody) isClosed() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.closed
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestRoundTrip_ClosesBodyOn4xx(t *testing.T) {
	statusCodes := []int{
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusNotFound,
	}

	for _, code := range statusCodes {
		t.Run(fmt.Sprintf("status_%d", code), func(t *testing.T) {
			body := &trackingBody{Reader: strings.NewReader(`{"error":"client error"}`)}
			inner := roundTripperFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: code,
					Body:       body,
					Header:     make(http.Header),
				}, nil
			})
			transport := NewExportarrTransport(inner, nil)
			req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
			assert.NoError(t, err)

			resp, err := transport.RoundTrip(req)
			assert.Error(t, err)
			assert.Nil(t, resp)
			assert.True(t, body.isClosed(), "response body must be closed on 4xx")
		})
	}
}

func TestRoundTrip_ClosesBodyOn5xx(t *testing.T) {
	statusCodes := []int{
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
	}

	for _, code := range statusCodes {
		t.Run(fmt.Sprintf("status_%d", code), func(t *testing.T) {
			var mu sync.Mutex
			var bodies []*trackingBody
			inner := roundTripperFunc(func(*http.Request) (*http.Response, error) {
				mu.Lock()
				defer mu.Unlock()
				b := &trackingBody{Reader: strings.NewReader(`{"error":"server error"}`)}
				bodies = append(bodies, b)
				return &http.Response{
					StatusCode: code,
					Body:       b,
					Header:     make(http.Header),
				}, nil
			})
			transport := NewExportarrTransport(inner, nil)
			transport.Backoff = func(int) time.Duration { return 0 }
			req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
			assert.NoError(t, err)

			resp, err := transport.RoundTrip(req)
			assert.Error(t, err)
			assert.Nil(t, resp)

			mu.Lock()
			defer mu.Unlock()
			assert.Equal(t, len(bodies), maxRetries+1)
			for i, b := range bodies {
				assert.True(t, b.isClosed(), fmt.Sprintf("response body %d must be closed", i))
			}
		})
	}
}

func TestRoundTrip_RetriesClosePreviousBodyBeforeNextAttempt(t *testing.T) {
	var mu sync.Mutex
	var bodies []*trackingBody
	var closedBeforeNextAttempt []bool

	inner := roundTripperFunc(func(*http.Request) (*http.Response, error) {
		mu.Lock()
		defer mu.Unlock()
		if len(bodies) > 0 {
			prev := bodies[len(bodies)-1]
			closedBeforeNextAttempt = append(closedBeforeNextAttempt, prev.isClosed())
		}
		body := &trackingBody{Reader: strings.NewReader(`{"error":"server error"}`)}
		bodies = append(bodies, body)
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       body,
			Header:     make(http.Header),
		}, nil
	})

	transport := NewExportarrTransport(inner, nil)
	transport.Backoff = func(int) time.Duration { return 0 }

	req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
	assert.NoError(t, err)

	resp, err := transport.RoundTrip(req)
	assert.Error(t, err)
	assert.Nil(t, resp)

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, len(bodies), maxRetries+1)
	assert.Equal(t, len(closedBeforeNextAttempt), maxRetries)
	for i, closed := range closedBeforeNextAttempt {
		assert.True(t, closed, fmt.Sprintf("attempt %d response body was not closed before next attempt", i+1))
	}
	for i, b := range bodies {
		assert.True(t, b.isClosed(), fmt.Sprintf("body %d was not closed", i+1))
	}
}

func TestRoundTrip_RetrySuccessClosesPreviousBodies(t *testing.T) {
	var mu sync.Mutex
	var bodies []*trackingBody
	var closedBeforeNextAttempt []bool

	inner := roundTripperFunc(func(*http.Request) (*http.Response, error) {
		mu.Lock()
		defer mu.Unlock()
		if len(bodies) > 0 {
			prev := bodies[len(bodies)-1]
			closedBeforeNextAttempt = append(closedBeforeNextAttempt, prev.isClosed())
		}
		body := &trackingBody{Reader: strings.NewReader(`{}`)}
		bodies = append(bodies, body)
		status := http.StatusInternalServerError
		if len(bodies) == 2 {
			status = http.StatusOK
		}
		return &http.Response{
			StatusCode: status,
			Body:       body,
			Header:     make(http.Header),
		}, nil
	})

	transport := NewExportarrTransport(inner, nil)
	transport.Backoff = func(int) time.Duration { return 0 }

	req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
	assert.NoError(t, err)

	resp, err := transport.RoundTrip(req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, resp.StatusCode, http.StatusOK)

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, len(bodies), 2)
	assert.Equal(t, len(closedBeforeNextAttempt), 1)
	assert.True(t, closedBeforeNextAttempt[0], "failed attempt body must be closed before retry")
	assert.True(t, bodies[0].isClosed(), "failed attempt body must be closed")
	assert.False(t, bodies[1].isClosed(), "successful response body must remain open for caller")
	_ = resp.Body.Close()
	assert.True(t, bodies[1].isClosed(), "successful response body closed by caller")
}

func TestRoundTrip_ClosesBodyOnRedirect(t *testing.T) {
	body := &trackingBody{Reader: strings.NewReader("redirect")}
	inner := roundTripperFunc(func(*http.Request) (*http.Response, error) {
		hdr := make(http.Header)
		hdr.Set("Location", "http://example.com/target")
		return &http.Response{
			StatusCode: http.StatusMovedPermanently,
			Body:       body,
			Header:     hdr,
		}, nil
	})

	transport := NewExportarrTransport(inner, nil)
	req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
	assert.NoError(t, err)

	resp, err := transport.RoundTrip(req)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, body.isClosed(), "redirect response body must be closed")
}

func TestRoundTrip_ClosesBodyOnError(t *testing.T) {
	body := &trackingBody{Reader: strings.NewReader("error body")}
	inner := roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       body,
		}, fmt.Errorf("network error")
	})

	transport := NewExportarrTransport(inner, nil)
	transport.Backoff = func(int) time.Duration { return 0 }
	req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
	assert.NoError(t, err)

	resp, err := transport.RoundTrip(req)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, body.isClosed(), "response body must be closed when inner returns error")
}
