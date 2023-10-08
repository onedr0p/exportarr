package handlers

import (
	"fmt"
	"net/http"
)

func HealthzHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK")) //nolint:errcheck
	fmt.Fprint(w)
}
