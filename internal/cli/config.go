package cli

import (
	"fmt"
	"slices"
)

type logLensConfig struct {
	Format, CPUProfile, MemProfile, Trace string
	Top, Errors, Workers, MaxErrors       int
}

func (c logLensConfig) validate() error {
	// format
	validFormats := []string{"text", "json"}

	if !slices.Contains(validFormats, c.Format) {
		return fmt.Errorf("invalid value %q for --format: must be in: text, json", c.Format)
	}

	// top
	if c.Top < 0 {
		return fmt.Errorf("invalid value %d for --top: must be >= 0", c.Top)
	}

	// errors
	if c.Errors < 0 {
		return fmt.Errorf("invalid value %d for --errors: must be >= 0", c.Errors)
	}

	// workers
	if c.Workers < 1 {
		return fmt.Errorf("invalid value %d for --jobs: must be >= 1", c.Workers)
	}

	// max errors
	if c.MaxErrors < 0 {
		return fmt.Errorf("invalid value %d for --max-errors: must be >= 0", c.MaxErrors)
	}

	return nil
}
