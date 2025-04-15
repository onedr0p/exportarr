package config

import (
	"fmt"

	"github.com/gookit/validate"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/v2"
	flag "github.com/spf13/pflag"
)

type BazarrConfig struct {
	SeriesBatchSize        int `koanf:"series-batch-size"`
	SeriesBatchConcurrency int `koanf:"series-batch-concurrency"`
}

func RegisterBazarrFlags(flags *flag.FlagSet) {
	flags.Int("series-batch-size", 300, "Number of Series to retrieve from Bazarr in each API Call")
	flags.Int("series-batch-concurrency", 10, "Calls to make to Bazarrr Concurrently")
}

func (b BazarrConfig) Validate() error {
	v := validate.Struct(b)
	if !v.Validate() {
		return v.Errors
	}
	if b.SeriesBatchSize < 1 {
		return fmt.Errorf("series-batch-size must be greater than zero")
	}
	if b.SeriesBatchConcurrency < 1 {
		return fmt.Errorf("series-batch-concurrency must be greater than zero")
	}
	return nil
}

func (b BazarrConfig) Translate() map[string]string {
	return validate.MS{
		"SeriesBatchSize":        "series-batch-size",
		"SeriesBatchConcurrency": "series-batch-concurrency",
	}
}

func (c *ArrConfig) LoadBazarrConfig(flags *flag.FlagSet) error {
	err := c.k.Load(posflag.Provider(flags, ".", c.k), nil, koanf.WithMergeFunc(func(src, dest map[string]interface{}) error {
		dest["bazarr"] = src
		return nil
	}))
	if err != nil {
		return err
	}

	c.Bazarr = BazarrConfig{
		SeriesBatchSize:        300,
		SeriesBatchConcurrency: 10,
	}
	err = c.k.Unmarshal("bazarr", &c.Bazarr)
	if err != nil {
		return err
	}
	return nil
}
