package client

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/onedr0p/exportarr/internal/config"
	"go.uber.org/zap"
)

// Client struct is an *Arr client.
type Client struct {
	httpClient http.Client
	URL        url.URL
}

// NewClient method initializes a new *Arr client.
func NewClient(config *config.Config) (*Client, error) {

	baseURL, err := url.Parse(config.URL)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse URL(%s): %w", config.URL, err)
	}

	baseTransport := http.DefaultTransport
	if config.DisableSSLVerify {
		baseTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	var auth Authenticator
	if config.UseFormAuth() {
		auth = &FormAuth{
			Username:    config.AuthUsername,
			Password:    config.AuthPassword,
			ApiKey:      config.ApiKey,
			AuthBaseURL: baseURL,
			Transport:   baseTransport,
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

	return &Client{
		httpClient: http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
			Transport: NewArrTransport(auth, baseTransport),
		},
		URL: *baseURL.JoinPath("api", config.ApiVersion),
	}, nil
}

func (c *Client) unmarshalBody(b io.Reader, target interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			// return recovered panic as error
			err = fmt.Errorf("Recovered from panic: %s", r)

			log := zap.S()
			if zap.S().Level() == zap.DebugLevel {
				s := new(strings.Builder)
				_, copyErr := io.Copy(s, b)
				if copyErr != nil {
					zap.S().Errorw("Failed to copy body to string in recover",
						"error", err, "recover", r)
				}
				log = log.With("body", s.String())
			}
			log.Errorw("Recovered while unmarshalling response", "error", r)

		}
	}()
	err = json.NewDecoder(b).Decode(target)
	return
}

// DoRequest - Take a HTTP Request and return Unmarshaled data
func (c *Client) DoRequest(endpoint string, target interface{}, queryParams ...map[string]string) error {
	values := c.URL.Query()
	for _, m := range queryParams {
		for k, v := range m {
			values.Add(k, v)
		}
	}
	url := c.URL.JoinPath(endpoint)
	url.RawQuery = values.Encode()
	zap.S().Infow("Sending HTTP request",
		"url", url)

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return fmt.Errorf("Failed to create HTTP Request(%s): %w", url, err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("Failed to execute HTTP Request(%s): %w", url, err)
	}
	defer resp.Body.Close()
	return c.unmarshalBody(resp.Body, target)
}
