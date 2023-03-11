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
	var baseURL *url.URL
	auth := AuthConfig{
		Username: c.String("basic-auth-username"),
		Password: c.String("basic-auth-password"),
	}

	apiVersion := cf.ApiVersion

	if c.String("config") != "" {
		baseURL.Parse(c.String("url") + ":" + cf.Port)
		baseURL = baseURL.JoinPath(cf.UrlBase, "api", apiVersion)
		auth.ApiKey = cf.ApiKey

	} else {
		// Otherwise use the value provided in the api-key flag
		baseURL.Parse(c.String("url"))
		baseURL = baseURL.JoinPath("api", apiVersion)

		if c.String("api-key") != "" {
			auth.ApiKey = c.String("api-key")
		} else if c.String("api-key-file") != "" {
			data, err := os.ReadFile(c.String("api-key-file"))
			if err != nil {
				return nil, fmt.Errorf("Couldn't Read API Key file %w", err)
			}
			auth.ApiKey = string(data)
		}
	}
	baseTransport := http.DefaultTransport
	if c.Bool("disable-ssl-verify") {
		baseTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	return &Client{
		httpClient: http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
			Transport: NewArrTransport(auth, baseTransport),
		},
	}, nil
}

// DoRequest - Take a HTTP Request and return Unmarshaled data
func (c *Client) DoRequest(endpoint string, target interface{}) error {
	url := c.URL.JoinPath(endpoint).String()
	log.Infof("Sending HTTP request to %s", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("Failed to create HTTP Request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		if location := resp.Header.Get("Location"); location != "" {
			return fmt.Errorf("Failed to execute HTTP Request(%s - %s): %w", url, location, err)
		} else {
			return fmt.Errorf("Failed to execute HTTP Request(%s): %w", url, err)
		}
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(target)
}
