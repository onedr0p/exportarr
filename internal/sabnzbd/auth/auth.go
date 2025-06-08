package auth

import (
	"net/http"
)

type ApiKeyAuth struct {
	ApiKey string
}

func (a ApiKeyAuth) Auth(req *http.Request) error {
	q := req.URL.Query()
	q.Add("apikey", a.ApiKey)
	q.Add("output", "json")
	req.URL.RawQuery = q.Encode()
	return nil
}

type BasicAuth struct {
	Username string
	Password string
	ApiKey   string
}

func (a *BasicAuth) Auth(req *http.Request) error {
	req.SetBasicAuth(a.Username, a.Password)
	if req.URL.Query().Get("apikey") == "" {
		q := req.URL.Query()
		q.Add("apikey", a.ApiKey)
		req.URL.RawQuery = q.Encode()
	}
	return nil
}
