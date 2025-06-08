package client

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/shamelin/exportarr/internal/arr/config"
	"github.com/shamelin/exportarr/internal/client"
)

type Client = client.Client
type QueryParams = client.QueryParams

func NewClient(config *config.ArrConfig) (*Client, error) {
	auth, err := NewAuth(config)
	if err != nil {
		return nil, err
	}
	return client.NewClient(config.BaseURL(), config.DisableSSLVerify, auth)
}

func NewAuth(config *config.ArrConfig) (client.Authenticator, error) {
	var auth client.Authenticator

	if config.UseFormAuth() {
		u, err := url.Parse(config.URL)
		if err != nil {
			return nil, err
		}
		auth = &FormAuth{
			Username:    config.AuthUsername,
			Password:    config.AuthPassword,
			ApiKey:      config.ApiKey,
			AuthBaseURL: u,
			Transport:   client.BaseTransport(config.DisableSSLVerify),
		}
	} else if config.UseBasicAuth() {
		auth = &BasicAuth{
			Username: config.AuthUsername,
			Password: config.AuthPassword,
			ApiKey:   config.ApiKey,
		}
	} else {
		auth = &ApiKeyAuth{
			ApiKey: config.ApiKey,
		}
	}
	return auth, nil
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
		vals := u.Query()
		vals.Add("ReturnUrl", "/general/settings")
		u.RawQuery = vals.Encode()

		authReq, err := http.NewRequest("POST", u.String(), strings.NewReader(form.Encode()))
		if err != nil {
			return fmt.Errorf("failed to renew FormAuth Cookie: %w", err)
		}

		authReq.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		authReq.Header.Add("Content-Length", fmt.Sprintf("%d", len(form.Encode())))

		client := &http.Client{Transport: a.Transport, CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if req.URL.Query().Get("loginFailed") == "true" {
				return fmt.Errorf("failed to renew FormAuth Cookie: Login Failed")
			}
			return http.ErrUseLastResponse
		}}

		authResp, err := client.Do(authReq)
		if err != nil {
			return fmt.Errorf("failed to renew FormAuth Cookie: %w", err)
		}

		if authResp.StatusCode != 302 {
			return fmt.Errorf("failed to renew FormAuth Cookie: Received Status Code %d", authResp.StatusCode)
		}

		found := false
		for _, cookie := range authResp.Cookies() {
			if strings.HasSuffix(cookie.Name, "arrAuth") {
				copy := *cookie
				a.cookie = &copy
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("failed to renew FormAuth Cookie: No Cookie with suffix 'arrAuth' found")
		}
	}

	req.AddCookie(a.cookie)
	req.Header.Add("X-Api-Key", a.ApiKey)

	return nil
}
