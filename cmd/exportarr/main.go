package main

import (
	"github.com/shamelin/exportarr/internal/commands"
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
		panic(err)
	}
}
