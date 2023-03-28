package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onedr0p/exportarr/internal/config"
)

func TestNewClient(t *testing.T) {
	require := require.New(t)
	c := &config.Config{
		URL:        "http://localhost:7878",
		ApiKey:     "abcdef0123456789abcdef0123456789",
		ApiVersion: "v3",
	}

	client, err := NewClient(c)
	_, ok := client.httpClient.Transport.(*ArrTransport).auth.(*ApiKeyAuth)
	require.True(ok, "NewClient should return a client with an ApiKeyAuth authenticator")
	require.Nil(err, "NewClient should not return an error")
	require.NotNil(client, "NewClient should return a client")
	require.Equal(client.URL.String(), "http://localhost:7878/api/v3", "NewClient should return a client with the correct URL")
}

// Need tests for FormAuth & BasicAuth

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
			expectedURL: "/api/v3/queue",
		},
		{
			name:     "params",
			endpoint: "test",
			queryParams: map[string]string{
				"page":      "1",
				"testParam": "asdf",
			},
			expectedURL: "/api/v3/test?page=1&testParam=asdf",
		},
	}
	for _, param := range parameters {
		t.Run(param.name, func(t *testing.T) {
			require := require.New(t)
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(param.expectedURL, r.URL.String(), "DoRequest should use the correct URL")
				fmt.Fprintln(w, "{\"test\": \"asdf2\"}")
			}))
			defer ts.Close()

			c := &config.Config{
				URL:        ts.URL,
				ApiVersion: "v3",
			}

			target := struct {
				Test string `json:"test"`
			}{}
			expected := target
			expected.Test = "asdf2"
			client, err := NewClient(c)
			require.Nil(err, "NewClient should not return an error")
			require.NotNil(client, "NewClient should return a client")
			err = client.DoRequest(param.endpoint, &target, param.queryParams)
			require.Nil(err, "DoRequest should not return an error: %s", err)
			require.Equal(expected, target, "DoRequest should return the correct data")
		})
	}
}

func TestDoRequest_PanicRecovery(t *testing.T) {
	require := require.New(t)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ret := struct {
			TestField  string
			TestField2 string
		}{
			TestField:  "asdf",
			TestField2: "asdf2",
		}
		s, err := json.Marshal(ret)
		require.NoError(err)
		w.Write(s)
		w.WriteHeader(http.StatusOK)
		return
	}))
	defer ts.Close()

	c := &config.Config{
		URL:        ts.URL,
		ApiVersion: "v3",
	}

	client, err := NewClient(c)
	require.Nil(err, "NewClient should not return an error")
	require.NotNil(client, "NewClient should return a client")

	err = client.DoRequest("test", nil)
	require.NotPanics(func() {
		require.Error(err, "DoRequest should return an error: %s", err)
	}, "DoRequest should recover from a panic")
}
