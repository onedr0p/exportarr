package commands

import (
	"errors"
	"net"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestServeHTTPListenAddress(t *testing.T) {
	// Isolate serveHTTP's signal handler and global configuration in a subprocess.
	if address := os.Getenv("EXPORTARR_TEST_LISTEN_ADDRESS"); address != "" {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			t.Fatal(err)
		}
		conf.Interface = host
		conf.Port, err = strconv.Atoi(port)
		if err != nil {
			t.Fatal(err)
		}
		err = serveHTTP(func(prometheus.Registerer) {})
		if !errors.Is(err, syscall.EADDRINUSE) {
			t.Fatalf("expected address already in use for %s, got %v", address, err)
		}
		return
	}

	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name    string
		network string
		host    string
	}{
		{name: "IPv4 wildcard", network: "tcp4", host: "0.0.0.0"},
		{name: "IPv4 loopback", network: "tcp4", host: "127.0.0.1"},
		{name: "IPv6 loopback", network: "tcp6", host: "::1"},
		{name: "IPv6 wildcard", network: "tcp6", host: "::"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			listener, err := net.Listen(tc.network, net.JoinHostPort(tc.host, "0"))
			if err != nil {
				if tc.network == "tcp6" {
					t.Skipf("IPv6 unavailable: %v", err)
				}
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := listener.Close(); err != nil {
					t.Error(err)
				}
			})

			cmd := exec.Command(executable, "-test.run=^TestServeHTTPListenAddress$")
			cmd.Env = append(os.Environ(), "EXPORTARR_TEST_LISTEN_ADDRESS="+listener.Addr().String())
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("serveHTTP failed: %v\n%s", err, output)
			}
		})
	}
}
