package handlers

import (
	"fmt"
	"net/http"
)

// IndexHandler serves the landing page linking to /metrics.
func IndexHandler(w http.ResponseWriter, _ *http.Request) {
	response := `<h1>Exportarr</h1><p><a href='/metrics'>metrics</a></p>`
	_, _ = fmt.Fprintln(w, response)
}
