package client

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	TEST_USER = "testuser1"
	TEST_PASS = "hunter2"
	TEST_KEY  = "abcdef1234567890abcdef1234567890"
)

type testRoundTripFunc func(req *http.Request) (*http.Response, error)

func (t testRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return t(req)
}

func TestRoundTrip_Auth(t *testing.T) {
	parameters := []struct {
		name     string
		auth     AuthConfig
		testFunc func(req *http.Request) (*http.Response, error)
	}{
		{
			name: "BasicAuth",
			auth: AuthConfig{
				Username: TEST_USER,
				Password: TEST_PASS,
				ApiKey:   TEST_KEY,
			},
			testFunc: func(req *http.Request) (*http.Response, error) {
				require.NotNil(t, req, "Request should not be nil")
				require.NotNil(t, req.Header, "Request header should not be nil")
				require.NotEmpty(t, req.Header.Get("Authorization"), "Authorization header should be set")
				require.Equal(
					t,
					"Basic "+base64.StdEncoding.EncodeToString([]byte(TEST_USER+":"+TEST_PASS)),
					req.Header.Get("Authorization"),
					"Authorization Header set to wrong value",
				)
				require.NotEmpty(t, req.Header.Get("X-Api-Key"), "X-Api-Key header should be set")
				require.Equal(t, TEST_KEY, req.Header.Get("X-Api-Key"), "X-Api-Key Header set to wrong value")
				return &http.Response{
					StatusCode: 200,
					Body:       nil,
					Header:     make(http.Header),
				}, nil
			},
		},
		{
			name: "ApiKey",
			auth: AuthConfig{
				Username: "",
				Password: "",
				ApiKey:   TEST_KEY,
			},
			testFunc: func(req *http.Request) (*http.Response, error) {
				require.NotNil(t, req, "Request should not be nil")
				require.NotNil(t, req.Header, "Request header should not be nil")
				require.Empty(t, req.Header.Get("Authorization"), "Authorization header should be empty")
				require.NotEmpty(t, req.Header.Get("X-Api-Key"), "X-Api-Key header should be set")
				require.Equal(t, TEST_KEY, req.Header.Get("X-Api-Key"), "X-Api-Key Header set to wrong value")
				return &http.Response{
					StatusCode: 200,
					Body:       nil,
					Header:     make(http.Header),
				}, nil
			},
		},
	}
	for _, param := range parameters {
		t.Run(param.name, func(t *testing.T) {
			transport := NewArrTransport(param.auth, testRoundTripFunc(param.testFunc))
			client := &http.Client{Transport: transport}
			req, err := http.NewRequest("GET", "http://example.com", nil)
			require.Nil(t, err, "Error creating request: %s", err)
			_, err = client.Do(req)
			require.Nil(t, err, "Error sending request: %s", err)
		})
	}
}

func TestRoundTrip_Retries(t *testing.T) {
	parameters := []struct {
		name     string
		testFunc func(req *http.Request) (*http.Response, error)
	}{
		{
			name: "500",
			testFunc: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: 500,
					Body:       nil,
					Header:     make(http.Header),
				}, nil
			},
		},
		{
			name: "Err",
			testFunc: func(req *http.Request) (*http.Response, error) {
				return nil, &http.ProtocolError{}
			},
		},
	}
	for _, param := range parameters {
		t.Run(param.name, func(t *testing.T) {
			require := require.New(t)
			auth := AuthConfig{
				ApiKey: TEST_KEY,
			}
			attempts := 0
			transport := NewArrTransport(auth, testRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				attempts++
				return param.testFunc(req)
			}))
			client := &http.Client{Transport: transport}
			req, err := http.NewRequest("GET", "http://example.com", nil)
			require.Nil(err, "Error creating request: %s", err)
			_, err = client.Do(req)
			require.NotNil(err, "Error should be returned from Do()")
			require.Equal(3, attempts, "Should retry 3 times")
		})
	}
}

func TestRoundTrip_StatusCodes(t *testing.T) {
	parameters := []int{200, 201, 202, 204, 301, 302, 400, 401, 403, 404, 500, 503}
	for _, param := range parameters {
		t.Run(fmt.Sprintf("%d", param), func(t *testing.T) {
			require := require.New(t)
			auth := AuthConfig{
				ApiKey: TEST_KEY,
			}
			transport := NewArrTransport(auth, testRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: param,
					Body:       nil,
					Header:     make(http.Header),
				}, nil
			}))
			client := &http.Client{Transport: transport}
			req, err := http.NewRequest("GET", "http://example.com", nil)
			require.Nil(err, "Error creating request: %s", err)
			_, err = client.Do(req)
			if param >= 200 && param < 300 {
				require.Nil(err, "Should Not error on 2XX: %s", err)
			} else {
				require.NotNil(err, "Should error on non-2XX")
			}
		})
	}
}
