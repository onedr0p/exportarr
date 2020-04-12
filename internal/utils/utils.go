package utils

import (
	"net/url"
	"regexp"
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
