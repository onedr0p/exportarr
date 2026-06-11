package client

import (
	"encoding/json"
	"fmt"
	"github.com/onedr0p/exportarr/internal/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClient(t *testing.T) {
	u := "http://localhost"
	c, err := NewClient(u, true, 0, nil)
	assert.NoError(t, err, "NewClient should not return an error")
	assert.NotNil(t, c, "NewClient should return a client")
	assert.Equal(t, c.URL.String(), u, "NewClient should set the correct URL")
	assert.True(t, c.httpClient.Transport.(*ExportarrTransport).inner.(*http.Transport).TLSClientConfig.InsecureSkipVerify)
}

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
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, r.URL.String(), param.expectedURL, "DoRequest should use the correct URL")
				fmt.Fprintln(w, "{\"test\": \"asdf2\"}")
			}))
			defer ts.Close()

			target := struct {
				Test string `json:"test"`
			}{}
			expected := target
			expected.Test = "asdf2"
			client, err := NewClient(ts.URL, false, 0, nil)
			if err != nil {
				panic(err)
			}
			assert.Nil(t, err, "NewClient should not return an error")
			assert.NotNil(t, client, "NewClient should return a client")
			err = client.DoRequest(param.endpoint, &target, param.queryParams)
			assert.Nil(t, err, "DoRequest should not return an error: %s", err)
			assert.Equal(t, target, expected, "DoRequest should return the correct data")
		})
	}
}

func TestDoRequest_PanicRecovery(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		ret := struct {
			TestField  string
			TestField2 string
		}{
			TestField:  "asdf",
			TestField2: "asdf2",
		}
		s, err := json.Marshal(ret)
		assert.NoError(t, err)
		_, _ = w.Write(s)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	client, err := NewClient(ts.URL, false, 0, nil)
	assert.Nil(t, err, "NewClient should not return an error")
	assert.NotNil(t, client, "NewClient should return a client")

	err = client.DoRequest("test", nil)
	assert.NotPanics(t, func() {
		assert.Error(t, err, "DoRequest should return an error: %s", err)
	}, "DoRequest should recover from a panic")
}
