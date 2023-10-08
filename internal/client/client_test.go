package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	u := "http://localhost"

	require := require.New(t)
	c, err := NewClient(u, true, nil)
	require.NoError(err, "NewClient should not return an error")
	require.NotNil(c, "NewClient should return a client")
	require.Equal(u, c.URL.String(), "NewClient should set the correct URL")
	require.True(c.httpClient.Transport.(*ExportarrTransport).inner.(*http.Transport).TLSClientConfig.InsecureSkipVerify)
}

// Need tests for FormAuth & BasicAuth

func TestDoRequest(t *testing.T) {
	parameters := []struct {
		name        string
		endpoint    string
		queryParams QueryParams
		expectedURL string
	}{
		{
			name:        "noParams",
			endpoint:    "queue",
			expectedURL: "/queue",
		},
		{
			name:     "params",
			endpoint: "test",
			queryParams: QueryParams{
				"page":      {"1"},
				"testParam": {"asdf"},
			},
			expectedURL: "/test?page=1&testParam=asdf",
		},
		{
			name:     "csv params",
			endpoint: "test",
			queryParams: QueryParams{
				"ids[]":     {"1", "2", "1234"},
				"testParam": {"asdf"},
			},
			expectedURL: "/test?ids%5B%5D=1&ids%5B%5D=2&ids%5B%5D=1234&testParam=asdf",
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

			target := struct {
				Test string `json:"test"`
			}{}
			expected := target
			expected.Test = "asdf2"
			client, err := NewClient(ts.URL, false, nil)
			if err != nil {
				panic(err)
			}
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
	}))
	defer ts.Close()

	client, err := NewClient(ts.URL, false, nil)
	require.Nil(err, "NewClient should not return an error")
	require.NotNil(client, "NewClient should return a client")

	err = client.DoRequest("test", nil)
	require.NotPanics(func() {
		require.Error(err, "DoRequest should return an error: %s", err)
	}, "DoRequest should recover from a panic")
}
