package client

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
)

// ArrTransport is a http.RoundTripper that adds authentication to requests
type ArrTransport struct {
	inner http.RoundTripper
	auth  AuthConfig
}

type AuthConfig struct {
	Username string
	Password string
	ApiKey   string
}

func NewArrTransport(auth AuthConfig, inner http.RoundTripper) *ArrTransport {
	return &ArrTransport{
		inner: &GzipTransport{inner},
		auth:  auth,
	}
}
func (t *ArrTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.auth.Username != "" && t.auth.Password != "" {
		req.SetBasicAuth(t.auth.Username, t.auth.Password)
	}
	req.Header.Add("X-Api-Key", t.auth.ApiKey)

	resp, err := t.inner.RoundTrip(req)

	if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
		return resp, nil
	}
	if resp.StatusCode >= 400 && resp.StatusCode <= 499 {
		return nil, fmt.Errorf("Received Client Error Status Code: %d", resp.StatusCode)
	}
	if err != nil || resp.StatusCode >= 500 {
		retries := 2
		for i := 0; i < retries; i++ {
			resp, err = t.inner.RoundTrip(req)
			if err == nil && resp.StatusCode < 500 {
				return resp, nil
			}
		}
	}
	if err != nil {
		return nil, err
	} else {
		return nil, fmt.Errorf("Received Server Error Status Code: %d", resp.StatusCode)
	}
}

// GzipTransport is a http.RoundTripper that adds gzip compression to requests
type GzipTransport struct {
	inner http.RoundTripper
}

func (t *GzipTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("Accept-Encoding", "gzip")
	resp, err := t.inner.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	var reader io.ReadCloser
	if resp.Header.Get("Content-Encoding") == "gzip" {
		reader, err = gzip.NewReader(resp.Body)
		if err != nil {
			return nil, err
		}
		resp.Body = reader
	}

	return resp, nil
}
