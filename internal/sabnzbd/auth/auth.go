// Package auth implements SABnzbd API authentication.
package auth

import (
	"net/http"
)

// APIKeyAuth authenticates SABnzbd requests via the apikey query parameter.
type APIKeyAuth struct {
	APIKey string
}

// Auth adds the API key and JSON output mode to the request's query string.
func (a APIKeyAuth) Auth(req *http.Request) error {
	q := req.URL.Query()
	q.Add("apikey", a.APIKey)
	q.Add("output", "json")
	req.URL.RawQuery = q.Encode()
	return nil
}
