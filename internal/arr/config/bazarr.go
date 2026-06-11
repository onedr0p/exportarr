package config

import (
	"errors"

	flag "github.com/spf13/pflag"

	base_config "github.com/onedr0p/exportarr/internal/config"
)

// BazarrConfig holds bazarr-specific exporter options.
type BazarrConfig struct {
	SeriesBatchSize        int `env:"SERIES_BATCH_SIZE" envDefault:"300"`
	SeriesBatchConcurrency int `env:"SERIES_BATCH_CONCURRENCY" envDefault:"10"`
}

// RegisterBazarrFlags registers bazarr-specific flags on the given FlagSet.
func RegisterBazarrFlags(flags *flag.FlagSet) {
	flags.Int("series-batch-size", 300, "Number of Series to retrieve from Bazarr in each API Call")
	flags.Int("series-batch-concurrency", 10, "Calls to make to Bazarrr Concurrently")
}

// Validate checks the bazarr configuration.
func (b BazarrConfig) Validate() error {
	if b.SeriesBatchSize < 1 {
		return errors.New("series-batch-size must be greater than zero")
	}
	if b.SeriesBatchConcurrency < 1 {
		return errors.New("series-batch-concurrency must be greater than zero")
	}
	return nil
}

// LoadBazarrConfig overlays bazarr-specific flags onto the ArrConfig
// (environment variables were already parsed by LoadArrConfig).
func (c *ArrConfig) LoadBazarrConfig(flags *flag.FlagSet) error {
	base_config.OverlayFlag(flags, "series-batch-size", flags.GetInt, &c.Bazarr.SeriesBatchSize)
	base_config.OverlayFlag(flags, "series-batch-concurrency", flags.GetInt, &c.Bazarr.SeriesBatchConcurrency)
	return nil
}
