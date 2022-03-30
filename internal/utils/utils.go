package utils

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/onedr0p/exportarr/internal/model"
	log "github.com/sirupsen/logrus"
)

// IsValidApikey - Check if the API Key is 32 characters and only a-z0-9
func IsValidApikey(str string) bool {
	found, err := regexp.MatchString("([a-z0-9]{32})", str)
	if err != nil {
		return false
	}
	return found
}

// IsValidUrl - Checks if the URL is valid
func IsValidUrl(str string) bool {
	u, err := url.Parse(str)
	return err == nil && u.Scheme != "" && u.Host != ""
}

// IsFileThere - Checks if the file is there
func IsFileThere(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// FormatURLBase - Formats a base URL
func FormatURLBase(urlBase string) string {
	u := strings.Trim(urlBase, "/")
	if urlBase == "" {
		return u
	}
	return fmt.Sprintf("/%s", strings.Trim(u, "/"))
}

// GetArrConfigFromFile - Get the config from config.xml
func GetArrConfigFromFile(file string) (*model.Config, error) {
	xmlFile, err := os.Open(file)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	defer xmlFile.Close()
	byteValue, _ := ioutil.ReadAll(xmlFile)

	var config model.Config
	xml.Unmarshal(byteValue, &config)

	log.Infof("Getting Config from %s", file)

	return &config, nil
}
