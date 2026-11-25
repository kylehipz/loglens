package cli

import (
	"fmt"
	"slices"
)

type LogLensConfig struct {
	Format, CPUProfile, MemProfile, Trace string
	Top, Errors, Workers, MaxErrors       int
}

func newLogLensConfig(
	format, cpuProfile, memProfile, trace string,
	top, errs, workers, maxErrors int,
) *LogLensConfig {
	return &LogLensConfig{
		Format:     format,
		CPUProfile: cpuProfile,
		MemProfile: memProfile,
		Trace:      trace,
		Top:        top,
		Errors:     errs,
		Workers:    workers,
		MaxErrors:  maxErrors,
	}
}

func (c LogLensConfig) validate() error {
	// format
	validFormats := []string{"text", "json"}

	if !slices.Contains(validFormats, c.Format) {
		return fmt.Errorf("invalid value %d for --format: must be in: text, json", c.Format)
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
		return fmt.Errorf("invalid value %d for --maxErrors: must be >= 0", c.MaxErrors)
	}

	return nil
}
