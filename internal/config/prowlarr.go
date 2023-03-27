package config

import (
	"fmt"
	"time"

	"github.com/gookit/validate"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/v2"
	flag "github.com/spf13/pflag"
)

type ProwlarrConfig struct {
	Backfill          bool   `koanf:"backfill"`
	BackfillSinceDate string `koanf:"backfill-since-date" validate:"date"`
	BackfillSinceTime time.Time
}

func (p ProwlarrConfig) Validate() error {
	v := validate.Struct(p)
	if !v.Validate() {
		return v.Errors
	}
	if p.BackfillSinceDate != "" && p.BackfillSinceTime.IsZero() {
		// Should be unreachable as long as we validate that the date is valid in LoadProwlarrFlags/Validate
		return fmt.Errorf("backfill-since-date is not a valid date")
	}
	return nil
}

func (p ProwlarrConfig) Messages() map[string]string {
	return validate.MS{
		"BackfillSinceDate.date": "backfill-since-date must be in the format YYYY-MM-DD",
	}
}

func (p ProwlarrConfig) Translates() map[string]string {
	return validate.MS{
		"BackfillSinceDate": "backfill-since-date",
	}
}

func (c *Config) LoadProwlarrFlags(flags *flag.FlagSet) error {
	err := c.k.Load(posflag.Provider(flags, ".", c.k), nil, koanf.WithMergeFunc(func(src, dest map[string]interface{}) error {
		dest["arr.prowlarr"] = src
		return nil
	}))
	if err != nil {
		return err
	}

	err = c.k.Unmarshal("arr.prowlarr", &c.Prowlarr)
	if err != nil {
		return err
	}
	c.Prowlarr.BackfillSinceTime = c.k.Time("prowlarr.backfill-since-date", "2006-01-02")
	return nil
}
