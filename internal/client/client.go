// Package client provides the shared authenticated HTTP client used by every
// exporter.
package client

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// defaultRequestTimeout caps each HTTP request so a hung instance cannot pin
// scrape goroutines forever; large bazarr/lidarr history payloads can take
// tens of seconds, so the default is generous and overridable via config.
const defaultRequestTimeout = 60 * time.Second

// Client struct is an *Arr client.
type Client struct {
	httpClient http.Client
	URL        url.URL
}

// QueryParams holds URL query parameters.
type QueryParams = url.Values

// NewClient method initializes a new *Arr client.
func NewClient(baseURL string, insecureSkipVerify bool, timeout time.Duration, auth Authenticator) (*Client, error) {
	if timeout <= 0 {
		timeout = defaultRequestTimeout
	}

	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL(%s): %w", baseURL, err)
	}

	return &Client{
		httpClient: http.Client{
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
			Timeout:   timeout,
			Transport: NewExportarrTransport(BaseTransport(insecureSkipVerify), auth),
		},
		URL: *u,
	}, nil
}

func (c *Client) unmarshalBody(b io.Reader, target any) (err error) {
	defer func() {
		if r := recover(); r != nil {
			// return recovered panic as error
			err = fmt.Errorf("recovered from panic: %s", r)

			log := slog.Default()
			if log.Enabled(context.Background(), slog.LevelDebug) {
				s := new(strings.Builder)
				if _, copyErr := io.Copy(s, b); copyErr != nil {
					log.Error("Failed to copy body to string in recover",
						"error", copyErr, "recover", r)
				}
				log = log.With("body", s.String())
			}
			log.Error("Recovered while unmarshalling response", "error", r)
		}
	}()
	err = json.NewDecoder(b).Decode(target)
	return
}

// DoRequest - Take a HTTP Request and return Unmarshaled data
func (c *Client) DoRequest(endpoint string, target any, queryParams ...QueryParams) error {
	values := c.URL.Query()

	// merge all query params
	for _, m := range queryParams {
		for key, vals := range m {
			for _, val := range vals {
				values.Add(key, val)
			}
		}
	}

	endpointURL := c.URL.JoinPath(endpoint)
	endpointURL.RawQuery = values.Encode()
	slog.Debug("Sending HTTP request", "url", endpointURL)

	req, err := http.NewRequest(http.MethodGet, endpointURL.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to create HTTP Request(%s): %w", endpointURL, err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute HTTP Request(%s): %w", endpointURL, err)
	}
	defer func() { _ = resp.Body.Close() }()
	return c.unmarshalBody(resp.Body, target)
}

// Get fetches an endpoint and decodes the JSON response into T.
func Get[T any](c *Client, endpoint string, queryParams ...QueryParams) (T, error) {
	var out T
	err := c.DoRequest(endpoint, &out, queryParams...)
	return out, err
}

// BaseTransport returns a clone of the default transport, optionally with TLS
// verification disabled. Cloning keeps the insecure setting scoped to this
// client instead of mutating the process-wide http.DefaultTransport.
func BaseTransport(insecureSkipVerify bool) http.RoundTripper {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	// Every collector in a command scrapes the same host concurrently; the
	// default of 2 idle conns per host forces constant TLS re-handshakes.
	transport.MaxIdleConnsPerHost = 16
	if insecureSkipVerify {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // opt-in via --disable-ssl-verify
	}
	return transport
}
