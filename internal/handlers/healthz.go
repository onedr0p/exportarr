// Package handlers provides exportarr's HTTP handlers and middleware.
package handlers

import (
	"net/http"
)

// HealthzHandler always reports OK; it signals process liveness, not backend
// reachability.
func HealthzHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}
