package commands

import (
	"testing"

	"github.com/onedr0p/exportarr/internal/config"
	"github.com/stretchr/testify/require"
)

func TestBackwardsCompatibility(t *testing.T) {
	flags := rootCmd.PersistentFlags()
	flags.Set("url", "http://localhost")
	flags.Set("api-key", "abcdef0123456789abcdef0123456789")
	flags.Set("basic-auth-username", "user")
	flags.Set("basic-auth-password", "pass")

	require := require.New(t)
	config, err := config.LoadConfig(flags)
	require.NoError(err)
	require.Equal("user", config.AuthUsername)
	require.Equal("pass", config.AuthPassword)
}
