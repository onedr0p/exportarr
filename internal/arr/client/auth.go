// Package client provides the authenticated HTTP client for *arr APIs.
package client

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/onedr0p/exportarr/internal/arr/config"
	"github.com/onedr0p/exportarr/internal/client"
)

// Client is the shared exportarr HTTP client.
type Client = client.Client

// QueryParams holds URL query parameters.
type QueryParams = client.QueryParams

// Get fetches an endpoint from the *arr API and decodes the response into T.
func Get[T any](c *Client, endpoint string, queryParams ...QueryParams) (T, error) {
	return client.Get[T](c, endpoint, queryParams...)
}

// NewClient builds an authenticated client for the configured *arr instance.
func NewClient(config *config.ArrConfig) (*Client, error) {
	auth, err := NewAuth(config)
	if err != nil {
		return nil, err
	}
	return client.NewClient(config.BaseURL(), config.DisableSSLVerify, config.RequestTimeout, auth)
}

// NewAuth selects the authenticator (form, basic, or API key) for the config.
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
			APIKey:      config.APIKey,
			AuthBaseURL: u,
			Transport:   client.BaseTransport(config.DisableSSLVerify),
			Timeout:     config.RequestTimeout,
		}
	} else {
		auth = &APIKeyAuth{
			APIKey: config.APIKey,
		}
	}
	return auth, nil
}

// APIKeyAuth authenticates requests with the X-Api-Key header.
type APIKeyAuth struct {
	APIKey string
}

// Auth adds the X-Api-Key header.
func (a *APIKeyAuth) Auth(req *http.Request) error {
	req.Header.Add("X-Api-Key", a.APIKey)
	return nil
}

// FormAuth authenticates via the *arr login form, caching the session cookie.
// Safe for concurrent use by collectors sharing one client.
type FormAuth struct {
	Username    string
	Password    string
	APIKey      string
	AuthBaseURL *url.URL
	Transport   http.RoundTripper
	Timeout     time.Duration

	mu     sync.Mutex
	cookie *http.Cookie
}

// Auth attaches the cached session cookie (renewing it via the login form
// when needed) and the X-Api-Key header.
func (a *FormAuth) Auth(req *http.Request) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	// Renew when missing or expiring within the next five minutes. A zero
	// Expires is a session cookie: cache it for the life of the process
	// instead of logging in on every request.
	if a.cookie == nil || (!a.cookie.Expires.IsZero() && a.cookie.Expires.Before(time.Now().Add(5*time.Minute))) {
		form := url.Values{
			"username":   {a.Username},
			"password":   {a.Password},
			"rememberMe": {"on"},
		}

		u := a.AuthBaseURL.JoinPath("login")
		vals := u.Query()
		vals.Add("ReturnUrl", "/general/settings")
		u.RawQuery = vals.Encode()

		// Inherit the API request's context so cancellation covers the login
		// round-trip too.
		authReq, err := http.NewRequestWithContext(req.Context(), http.MethodPost, u.String(), strings.NewReader(form.Encode()))
		if err != nil {
			return fmt.Errorf("failed to renew FormAuth Cookie: %w", err)
		}

		authReq.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		authReq.Header.Add("Content-Length", fmt.Sprintf("%d", len(form.Encode())))

		client := &http.Client{Transport: a.Transport, Timeout: a.Timeout, CheckRedirect: func(req *http.Request, _ []*http.Request) error {
			if req.URL.Query().Get("loginFailed") == "true" {
				return fmt.Errorf("failed to renew FormAuth Cookie: Login Failed")
			}
			return http.ErrUseLastResponse
		}}

		authResp, err := client.Do(authReq)
		if err != nil {
			return fmt.Errorf("failed to renew FormAuth Cookie: %w", err)
		}
		defer func() { _ = authResp.Body.Close() }()

		if authResp.StatusCode != http.StatusFound {
			return fmt.Errorf("failed to renew FormAuth Cookie: Received Status Code %d", authResp.StatusCode)
		}

		found := false
		for _, cookie := range authResp.Cookies() {
			if strings.HasSuffix(cookie.Name, "arrAuth") {
				cookieCopy := *cookie
				a.cookie = &cookieCopy
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("failed to renew FormAuth Cookie: No Cookie with suffix 'arrAuth' found")
		}
	}

	req.AddCookie(a.cookie)
	req.Header.Add("X-Api-Key", a.APIKey)

	return nil
}
