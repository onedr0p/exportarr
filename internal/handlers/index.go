package handlers

import (
	"fmt"
	"net/http"
)

func IndexHandler(w http.ResponseWriter, _ *http.Request) {
	response := `<h1>Exportarr</h1><p><a href='/metrics'>metrics</a></p>`
	fmt.Fprintln(w, response)
}
