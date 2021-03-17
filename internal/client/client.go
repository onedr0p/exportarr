package client

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

// Client struct is a Radarr client to request an instance of a Radarr
type Client struct {
	config     *cli.Context
	httpClient http.Client
}

// NewClient method initializes a new Radarr client.
func NewClient(c *cli.Context) *Client {
	return &Client{
		config: c,
		httpClient: http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

// DoRequest - Take a HTTP Request and return Unmarshaled data
func (c *Client) DoRequest(endpoint string, target interface{}) error {
	apiVersion := "v3"

	if c.config.Command.Name == "lidarr" {
		apiVersion = "v1"
	}

	url := fmt.Sprintf("%s/api/%s/%s", c.config.String("url"), apiVersion, endpoint)

	log.Infof("Sending HTTP request to %s", url)

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: c.config.Bool("disable-ssl-verify")}
	req, err := http.NewRequest("GET", url, nil)
	if c.config.String("basic-auth-username") != "" && c.config.String("basic-auth-password") != "" {
		req.Header.Add("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(c.config.String("basic-auth-username")+":"+c.config.String("basic-auth-password"))))
	}
	req.Header.Add("X-Api-Key", c.config.String("api-key"))

	if err != nil {
		log.Fatalf("An error has occurred when creating HTTP request %v", err)
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Fatalf("An error has occurred during retrieving statistics %v", err)
		return err
	}
	if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
		errMsg := fmt.Sprintf("An error has occurred during retrieving statistics HTTP statuscode %d", resp.StatusCode)
		log.Fatal(errMsg)
		return errors.New(errMsg)
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(target)
}
