// Command exportarr is an AIO Prometheus exporter for *arr applications.
package main

import (
	"os"

	"github.com/onedr0p/exportarr/internal/commands"
)

var (
	appName   = "exportarr"
	version   = "development"
	buildTime = ""
	revision  = ""
)

func main() {
	err := commands.Execute(commands.AppInfo{
		Name:      appName,
		Version:   version,
		BuildTime: buildTime,
		Revision:  revision,
	})
	if err != nil {
		// cobra has already printed the error and usage.
		os.Exit(1)
	}
}
