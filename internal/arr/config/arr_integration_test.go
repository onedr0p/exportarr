package config

import (
	"testing"

	base_config "github.com/shamelin/exportarr/internal/config"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
)

func TestArrConfigIntegration(t *testing.T) {
	require := require.New(t)

	configFlags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	base_config.RegisterConfigFlags(configFlags)

	arrConfigFlags := testFlagSet()

	t.Setenv("URL", "http://localhost")
	t.Setenv("CONFIG", "test_fixtures/config.test_xml")
	t.Setenv("PORT", "9792")
	t.Setenv("ENABLE_ADDITIONAL_METRICS", "true")
	t.Setenv("ENABLE_UNKNOWN_QUEUE_ITEMS", "false")

	config, err := base_config.LoadConfig(configFlags)
	require.NoError(err)
	arrConfig, err := LoadArrConfig(*config, arrConfigFlags)
	require.NoError(err)

	require.NoError(config.Validate())
	require.NoError(arrConfig.Validate())

}
