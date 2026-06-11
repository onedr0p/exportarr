package config

import (
	"errors"
	"time"

	flag "github.com/spf13/pflag"

	base_config "github.com/onedr0p/exportarr/internal/config"
)

// backfillDateFormat is the expected layout of backfill-since-date.
const backfillDateFormat = "2006-01-02"

// ProwlarrConfig holds prowlarr-specific exporter options.
type ProwlarrConfig struct {
	Backfill          bool   `env:"BACKFILL"`
	BackfillSinceDate string `env:"BACKFILL_SINCE_DATE"`
	BackfillSinceTime time.Time
}

// RegisterProwlarrFlags registers prowlarr-specific flags on the given
// FlagSet.
func RegisterProwlarrFlags(flags *flag.FlagSet) {
	flags.Bool("backfill", false, "Backfill Prowlarr")
	flags.String("backfill-since-date", "", "Date from which to start Prowlarr Backfill")
}

// Validate checks the prowlarr configuration.
func (p ProwlarrConfig) Validate() error {
	if p.BackfillSinceDate != "" {
		if _, err := time.Parse(backfillDateFormat, p.BackfillSinceDate); err != nil {
			return errors.New("backfill-since-date must be in the format YYYY-MM-DD")
		}
	}
	return nil
}

// LoadProwlarrConfig overlays prowlarr-specific flags onto the ArrConfig
// (environment variables were already parsed by LoadArrConfig) and resolves
// the backfill date.
func (c *ArrConfig) LoadProwlarrConfig(flags *flag.FlagSet) error {
	base_config.OverlayFlag(flags, "backfill", flags.GetBool, &c.Prowlarr.Backfill)
	base_config.OverlayFlag(flags, "backfill-since-date", flags.GetString, &c.Prowlarr.BackfillSinceDate)
	if c.Prowlarr.BackfillSinceDate != "" {
		t, err := time.Parse(backfillDateFormat, c.Prowlarr.BackfillSinceDate)
		if err != nil {
			return errors.New("backfill-since-date must be in the format YYYY-MM-DD")
		}
		c.Prowlarr.BackfillSinceTime = t
	}
	return nil
}
