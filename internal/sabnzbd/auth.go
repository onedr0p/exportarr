package sabnzbd

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
