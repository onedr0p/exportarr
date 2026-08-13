package handlers

import (
	"fmt"
	"net/http"
	"strings"
)

// IndexHandler serves the landing page linking to each metrics endpoint in
// paths. An empty path renders a bare "/metrics" link (single-app mode); a
// non-empty path renders "/metrics/<path>" (multi-app "serve" mode).
func IndexHandler(paths []string) http.HandlerFunc {
	var b strings.Builder
	b.WriteString("<h1>Exportarr</h1>")
	for _, p := range paths {
		href := "/metrics"
		if p != "" {
			href += "/" + p
		}
		fmt.Fprintf(&b, "<p><a href='%s'>%s</a></p>", href, href)
	}
	response := b.String()
	return func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, response)
	}
}
