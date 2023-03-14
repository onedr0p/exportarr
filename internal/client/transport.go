package client

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

type Authenticator interface {
	Auth(req *http.Request) error
}

// ArrTransport is a http.RoundTripper that adds authentication to requests
type ArrTransport struct {
	inner http.RoundTripper
	auth  Authenticator
}

func NewArrTransport(auth Authenticator, inner http.RoundTripper) *ArrTransport {
	return &ArrTransport{
		inner: inner,
		auth:  auth,
	}
}

func (t *ArrTransport) RoundTrip(req *http.Request) (*http.Response, error) {
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

type ApiKeyAuth struct {
	ApiKey string
}

func (a *ApiKeyAuth) Auth(req *http.Request) error {
	req.Header.Add("X-Api-Key", a.ApiKey)
	return nil
}

type BasicAuth struct {
	Username string
	Password string
	ApiKey   string
}

func (a *BasicAuth) Auth(req *http.Request) error {
	req.SetBasicAuth(a.Username, a.Password)
	req.Header.Add("X-Api-Key", a.ApiKey)
	return nil
}

type FormAuth struct {
	Username    string
	Password    string
	ApiKey      string
	AuthBaseURL *url.URL
	Transport   http.RoundTripper
	cookie      *http.Cookie
}

func (a *FormAuth) Auth(req *http.Request) error {
	if a.cookie == nil || a.cookie.Expires.Before(time.Now().Add(-5*time.Minute)) {
		form := url.Values{
			"username":   {a.Username},
			"password":   {a.Password},
			"rememberMe": {"on"},
		}

		u := a.AuthBaseURL.JoinPath("login")
		u.Query().Add("ReturnUrl", "/general/settings")

		authReq, err := http.NewRequest("POST", u.String(), strings.NewReader(form.Encode()))
		if err != nil {
			return fmt.Errorf("Failed to renew FormAuth Cookie: %w", err)
		}

		authReq.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		authReq.Header.Add("Content-Length", fmt.Sprintf("%d", len(form.Encode())))

		client := &http.Client{Transport: a.Transport, CheckRedirect: func(req *http.Request, via []*http.Request) error {
			log.Infof("Redirect: %s", req.URL.String())
			log.Infof("URL: %+v", req.URL)
			log.Infof("Query: %s", req.URL.Query())

			if req.URL.Query().Get("loginFailed") == "true" {
				return fmt.Errorf("Failed to renew FormAuth Cookie: Login Failed")
			}
			return http.ErrUseLastResponse
		}}

		authResp, err := client.Do(authReq)
		if err != nil {
			return fmt.Errorf("Failed to renew FormAuth Cookie: %w", err)
		}

		if authResp.StatusCode != 302 {
			return fmt.Errorf("Failed to renew FormAuth Cookie: Received Status Code %d", authResp.StatusCode)
		}

		for _, cookie := range authResp.Cookies() {
			if strings.HasSuffix(cookie.Name, "arrAuth") {
				copy := *cookie
				a.cookie = &copy
				break
			}
			return fmt.Errorf("Failed to renew FormAuth Cookie: No Cookie with suffix 'arrAuth' found")
		}
	}

	req.AddCookie(a.cookie)
	req.Header.Add("X-Api-Key", a.ApiKey)

	return nil
}
