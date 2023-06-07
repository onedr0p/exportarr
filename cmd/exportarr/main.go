package main

import "github.com/onedr0p/exportarr/internal/commands"

var (
	appName   = "exportarr"
	version   = "development"
	buildTime = ""
	revision  = ""
)

func main() {
	commands.Execute(commands.AppInfo{
		Name:      appName,
		Version:   version,
		BuildTime: buildTime,
		Revision:  revision,
	})
}
