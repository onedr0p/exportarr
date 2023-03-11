package client

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"

	"github.com/onedr0p/exportarr/internal/model"
)

func testContext(args map[string]string) *cli.Context {
	var ret cli.Context

	osArgs := os.Args[0:1]
	for k, v := range args {
		osArgs = append(osArgs, fmt.Sprintf("--%s=%s", k, v))
	}
	app := cli.NewApp()
	app.Flags = []cli.Flag{
		&cli.StringFlag{Name: "url"},
		&cli.StringFlag{Name: "api-key"},
		&cli.StringFlag{Name: "api-key-file"},
		&cli.StringFlag{Name: "basic-auth-username"},
		&cli.StringFlag{Name: "basic-auth-password"},
		&cli.StringFlag{Name: "config"},
		&cli.StringFlag{Name: "disable-ssl-verify"},
	}

	app.Action = func(c *cli.Context) error {
		// copy out context
		ret = *c
		return nil
	}
	err := app.Run(osArgs)
	if err != nil {
		panic(err)
	}
	return &ret
}
func TestNewClient_Flags(t *testing.T) {
	require := require.New(t)
	c := testContext(map[string]string{
		"url":     "http://localhost:7878",
		"api-key": "abcdef0123456789abcdef0123456789",
	})
	cf := model.NewConfig()
	client, err := NewClient(c, cf)
	require.Nil(err, "NewClient should not return an error")
	require.NotNil(client, "NewClient should return a client")
	require.Equal(client.URL.String(), "http://localhost:7878/api/v3", "NewClient should return a client with the correct URL")
}

func TestNewClient_File(t *testing.T) {
	require := require.New(t)
	c := testContext(map[string]string{
		"config":  "testdata/config.json",
		"url":     "http://localhost",
		"api-key": "abcdef0123456789abcdef0123456789",
	})
	cf := model.NewConfig()
	cf.Port = "7878"
	cf.UrlBase = "/radarr"

	client, err := NewClient(c, cf)
	require.Nil(err, "NewClient should not return an error")
	require.NotNil(client, "NewClient should return a client")
	require.Equal(client.URL.String(), "http://localhost:7878/radarr/api/v3", "NewClient should return a client with the correct URL")
}

func TestDoRequest(t *testing.T) {
	parameters := []struct {
		name        string
		endpoint    string
		queryParams map[string]string
		expectedURL string
	}{
		{
			name:        "noParams",
			endpoint:    "queue",
			expectedURL: "http://localhost:7878/api/v3/queue",
		},
		{
			name:     "params",
			endpoint: "test",
			queryParams: map[string]string{
				"page":      "1",
				"testParam": "asdf",
			},
			expectedURL: "http://localhost:7878/api/v3/test?page=1&testParam=asdf",
		},
	}
	for _, param := range parameters {
		t.Run(param.name, func(t *testing.T) {
			require := require.New(t)
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintln(w, "{\"test\": \"asdf2\"}")
			}))
			defer ts.Close()

			c := testContext(map[string]string{
				"url": ts.URL,
			})
			cf := model.NewConfig()

			target := struct {
				Test string `json:"test"`
			}{}
			expected := target
			expected.Test = "asdf2"
			client, err := NewClient(c, cf)
			require.Nil(err, "NewClient should not return an error")
			require.NotNil(client, "NewClient should return a client")
			err = client.DoRequest(param.endpoint, &target, param.queryParams)
			require.Nil(err, "DoRequest should not return an error: %s", err)
			require.Equal(expected, target, "DoRequest should return the correct data")
		})
	}
}
