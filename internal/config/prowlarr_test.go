package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestValidateProwlarr(t *testing.T) {
	tm, _ := time.Parse("2006-01-02", "2021-01-01")
	parameters := []struct {
		name        string
		config      *ProwlarrConfig
		shouldError bool
	}{
		{
			name: "good",
			config: &ProwlarrConfig{
				Backfill:          true,
				BackfillSinceTime: tm,
				BackfillSinceDate: "2021-01-01",
			},
		},
		{
			name: "bad-date",
			config: &ProwlarrConfig{
				Backfill:          true,
				BackfillSinceTime: tm,
				BackfillSinceDate: "2021-31-31",
			},
			shouldError: true,
		},
	}

	for _, parameter := range parameters {
		t.Run(parameter.name, func(t *testing.T) {
			err := parameter.config.Validate()
			if parameter.shouldError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
