package client

import (
	"fmt"
	"net/http"
)

type Authenticator interface {
	Auth(req *http.Request) error
}

// ArrTransport is a http.RoundTripper that adds authentication to requests
type ExportarrTransport struct {
	inner http.RoundTripper
	auth  Authenticator
}

func NewExportarrTransport(inner http.RoundTripper, auth Authenticator) *ExportarrTransport {
	return &ExportarrTransport{
		inner: inner,
		auth:  auth,
	}
}

func (t *ExportarrTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	err := t.auth.Auth(req)
	if err != nil {
		return nil, fmt.Errorf("Error authenticating request: %w", err)
	}

	resp, err := t.inner.RoundTrip(req)
	if err != nil || resp.StatusCode >= 500 {
		retries := 2
		for i := 0; i < retries; i++ {
			resp, err = t.inner.RoundTrip(req)
			if err == nil && resp.StatusCode < 500 {
				return resp, nil
			}
		}
		if err != nil {
			return nil, fmt.Errorf("Error sending HTTP Request: %w", err)
		} else {
			return nil, fmt.Errorf("Received Server Error Status Code: %d", resp.StatusCode)
		}
	}
	if resp.StatusCode >= 400 && resp.StatusCode <= 499 {
		return nil, fmt.Errorf("Received Client Error Status Code: %d", resp.StatusCode)
	}
	if resp.StatusCode >= 300 && resp.StatusCode <= 399 {
		if location, err := resp.Location(); err == nil {
			return nil, fmt.Errorf("Received Redirect Status Code: %d, Location: %s", resp.StatusCode, location.String())
		} else {
			return nil, fmt.Errorf("Received Redirect Status Code: %d, ", resp.StatusCode)
		}
	}
	return resp, nil
}
