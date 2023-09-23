package client

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"go.uber.org/zap"
)

// Client struct is an *Arr client.
type Client struct {
	httpClient http.Client
	URL        url.URL
}

// NewClient method initializes a new *Arr client.
func NewClient(baseURL string, insecureSkipVerify bool, auth Authenticator) (*Client, error) {

	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse URL(%s): %w", baseURL, err)
	}

	return &Client{
		httpClient: http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
			Transport: NewExportarrTransport(BaseTransport(insecureSkipVerify), auth),
		},
		URL: *u,
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
			// TODO: is there a better way to do this?
			for _, j := range strings.Split(v, ",") {
				values.Add(k, j)
			}
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

func BaseTransport(insecureSkipVerify bool) http.RoundTripper {
	baseTransport := http.DefaultTransport
	if insecureSkipVerify {
		baseTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	return baseTransport
}
