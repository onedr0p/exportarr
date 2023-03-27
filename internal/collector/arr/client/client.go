package client

import (
	base_client "github.com/onedr0p/exportarr/internal/client"
	"github.com/onedr0p/exportarr/internal/config"
)

func NewClient(config *config.Config) (*base_client.Client, error) {
	auth, err := NewAuth(config)
	if err != nil {
		return nil, err
	}
	return base_client.NewClient(config, auth)
}
