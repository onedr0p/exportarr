package client

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	base_client "github.com/onedr0p/exportarr/internal/client"

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
	require := require.New(t)
	parameters := []struct {
		name     string
		auth     base_client.Authenticator
		testFunc func(req *http.Request) (*http.Response, error)
	}{
		{
			name: "BasicAuth",
			auth: &BasicAuth{
				Username: TEST_USER,
				Password: TEST_PASS,
				ApiKey:   TEST_KEY,
			},
			testFunc: func(req *http.Request) (*http.Response, error) {
				require.NotNil(req, "Request should not be nil")
				require.NotNil(req.Header, "Request header should not be nil")
				require.NotEmpty(req.Header.Get("Authorization"), "Authorization header should be set")
				require.Equal(
					"Basic "+base64.StdEncoding.EncodeToString([]byte(TEST_USER+":"+TEST_PASS)),
					req.Header.Get("Authorization"),
					"Authorization Header set to wrong value",
				)
				require.NotEmpty(req.Header.Get("X-Api-Key"), "X-Api-Key header should be set")
				require.Equal(TEST_KEY, req.Header.Get("X-Api-Key"), "X-Api-Key Header set to wrong value")
				return &http.Response{
					StatusCode: 200,
					Body:       nil,
					Header:     make(http.Header),
				}, nil
			},
		},
		{
			name: "ApiKey",
			auth: &ApiKeyAuth{
				ApiKey: TEST_KEY,
			},
			testFunc: func(req *http.Request) (*http.Response, error) {
				require.NotNil(req, "Request should not be nil")
				require.NotNil(req.Header, "Request header should not be nil")
				require.Empty(req.Header.Get("Authorization"), "Authorization header should be empty")
				require.NotEmpty(req.Header.Get("X-Api-Key"), "X-Api-Key header should be set")
				require.Equal(TEST_KEY, req.Header.Get("X-Api-Key"), "X-Api-Key Header set to wrong value")
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
			transport := base_client.NewExportarrTransport(testRoundTripFunc(param.testFunc), param.auth)
			client := &http.Client{Transport: transport}
			req, err := http.NewRequest("GET", "http://example.com", nil)
			require.NoError(err, "Error creating request: %s", err)
			_, err = client.Do(req)
			require.NoError(err, "Error sending request: %s", err)
		})
	}
}

func TestRoundTrip_FormAuth(t *testing.T) {
	require := require.New(t)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NotNil(r, "Request should not be nil")
		require.NotNil(r.Header, "Request header should not be nil")
		require.Empty(r.Header.Get("Authorization"), "Authorization header should be empty")
		require.Equal("POST", r.Method, "Request method should be POST")
		require.Equal("/login", r.URL.Path, "Request URL should be /login")
		require.Equal("application/x-www-form-urlencoded", r.Header.Get("Content-Type"), "Content-Type should be application/x-www-form-urlencoded")
		require.Equal(TEST_USER, r.FormValue("username"), "Username should be %s", TEST_USER)
		require.Equal(TEST_PASS, r.FormValue("password"), "Password should be %s", TEST_PASS)
		http.SetCookie(w, &http.Cookie{
			Name:    "RadarrAuth",
			Value:   "abcdef1234567890abcdef1234567890",
			Expires: time.Now().Add(24 * time.Hour),
		})
		w.WriteHeader(http.StatusFound)
		w.Write([]byte("OK"))
	}))
	defer ts.Close()
	tsUrl, _ := url.Parse(ts.URL)
	auth := &FormAuth{
		Username:    TEST_USER,
		Password:    TEST_PASS,
		ApiKey:      TEST_KEY,
		AuthBaseURL: tsUrl,
		Transport:   http.DefaultTransport,
	}
	transport := base_client.NewExportarrTransport(testRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		require.NotNil(req, "Request should not be nil")
		require.NotNil(req.Header, "Request header should not be nil")
		cookie, err := req.Cookie("RadarrAuth")
		require.NoError(err, "Cookie should be set")
		require.Equal(cookie.Value, "abcdef1234567890abcdef1234567890", "Cookie should be set")
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       nil,
			Header:     make(http.Header),
		}, nil
	}), auth)
	client := &http.Client{Transport: transport}
	req, err := http.NewRequest("GET", "http://example.com", nil)
	require.NoError(err, "Error creating request: %s", err)
	_, err = client.Do(req)
	require.NoError(err, "Error sending request: %s", err)
}

func TestRoundTrip_FormAuthFailure(t *testing.T) {
	require := require.New(t)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/?loginFailed=true", http.StatusFound)
	}))
	u, _ := url.Parse(ts.URL)
	auth := &FormAuth{
		Username:    TEST_USER,
		Password:    TEST_PASS,
		ApiKey:      TEST_KEY,
		AuthBaseURL: u,
		Transport:   http.DefaultTransport,
	}
	transport := base_client.NewExportarrTransport(testRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       nil,
			Header:     make(http.Header),
		}, nil
	}), auth)
	client := &http.Client{Transport: transport}
	req, err := http.NewRequest("GET", "http://example.com", nil)
	require.NoError(err, "Error creating request: %s", err)
	require.NotPanics(func() {
		_, err = client.Do(req)
	}, "Form Auth should not panic on auth failure")
	require.Error(err, "Form Auth Transport should throw an error when auth fails")
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
				return nil, http.ErrNotSupported
			},
		},
	}
	for _, param := range parameters {
		t.Run(param.name, func(t *testing.T) {
			require := require.New(t)
			auth := &ApiKeyAuth{
				ApiKey: TEST_KEY,
			}
			attempts := 0
			transport := base_client.NewExportarrTransport(testRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				attempts++
				return param.testFunc(req)
			}), auth)
			client := &http.Client{Transport: transport}
			req, err := http.NewRequest("GET", "http://example.com", nil)
			require.NoError(err, "Error creating request: %s", err)
			_, err = client.Do(req)
			require.Error(err, "Error should be returned from Do()")
			require.Equal(3, attempts, "Should retry 3 times")
		})
	}
}

func TestRoundTrip_StatusCodes(t *testing.T) {
	parameters := []int{200, 201, 202, 204, 301, 302, 400, 401, 403, 404, 500, 503}
	for _, param := range parameters {
		t.Run(fmt.Sprintf("%d", param), func(t *testing.T) {
			require := require.New(t)
			auth := &ApiKeyAuth{
				ApiKey: TEST_KEY,
			}
			transport := base_client.NewExportarrTransport(testRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: param,
					Body:       nil,
					Header:     make(http.Header),
				}, nil
			}), auth)
			client := &http.Client{Transport: transport}
			req, err := http.NewRequest("GET", "http://example.com", nil)
			require.Nil(err, "Error creating request: %s", err)
			_, err = client.Do(req)
			if param >= 200 && param < 300 {
				require.NoError(err, "Should Not error on 2XX: %s", err)
			} else {
				require.Error(err, "Should error on non-2XX")
			}
		})
	}
}
