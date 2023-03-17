package client

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/onedr0p/exportarr/internal/model"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

// Client struct is a Radarr client to request an instance of a Radarr
type Client struct {
	httpClient http.Client
	URL        url.URL
}

// NewClient method initializes a new Radarr client.
func NewClient(c *cli.Context, cf *model.Config) (*Client, error) {
	var apiKey string
	var baseURL *url.URL

	apiVersion := cf.ApiVersion

	if c.String("config") != "" {
		var err error
		baseURL, err = baseURL.Parse(c.String("url") + ":" + cf.Port)
		if err != nil {
			return nil, fmt.Errorf("Couldn't parse URL: %w", err)
		}
		baseURL = baseURL.JoinPath(cf.UrlBase)
		apiKey = cf.ApiKey

	} else {
		// Otherwise use the value provided in the api-key flag
		var err error
		baseURL, err = baseURL.Parse(c.String("url"))
		if err != nil {
			return nil, fmt.Errorf("Couldn't parse URL: %w", err)
		}

		if c.String("api-key") != "" {
			apiKey = c.String("api-key")
		} else if c.String("api-key-file") != "" {
			data, err := os.ReadFile(c.String("api-key-file"))
			if err != nil {
				return nil, fmt.Errorf("Couldn't Read API Key file %w", err)
			}

			apiKey = string(data)
		}
	}

	baseTransport := http.DefaultTransport
	if c.Bool("disable-ssl-verify") {
		baseTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	var auth Authenticator
	if c.Bool("form-auth") {
		auth = &FormAuth{
			Username:    c.String("auth-username"),
			Password:    c.String("auth-password"),
			ApiKey:      apiKey,
			AuthBaseURL: baseURL,
			Transport:   baseTransport,
		}
	} else if c.String("username") != "" && c.String("password") != "" {
		auth = &BasicAuth{
			Username: c.String("auth-username"),
			Password: c.String("auth-password"),
			ApiKey:   apiKey,
		}
	} else {
		auth = &ApiKeyAuth{
			ApiKey: apiKey,
		}
	}

	return &Client{
		httpClient: http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
			Transport: NewArrTransport(auth, baseTransport),
		},
		URL: *baseURL.JoinPath("api", apiVersion),
	}, nil
}

// DoRequest - Take a HTTP Request and return Unmarshaled data
func (c *Client) DoRequest(endpoint string, target interface{}, queryParams ...map[string]string) error {
	for _, m := range queryParams {
		for k, v := range m {
			c.URL.Query().Add(k, v)
		}
	}
	url := c.URL.JoinPath(endpoint).String()
	log.Infof("Sending HTTP request to %s", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("Failed to create HTTP Request(%s): %w", url, err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("Failed to execute HTTP Request(%s): %w", url, err)
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(target)
}
