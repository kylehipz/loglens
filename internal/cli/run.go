// Package cli Flag and Args Validation.
package cli

import (
	"errors"
	"flag"
	"io"
	"runtime"
)

func Run(args []string, stdout, stderr io.Writer) int {
	var format, cpuProfile, memProfile, trace string
	var top, errs, workers, maxErrors int

	fs := flag.NewFlagSet("loglens", flag.ContinueOnError)
	fs.SetOutput(stderr)

	// format
	fs.StringVar(&format, "format", "text", "Output report format")

	// top
	fs.IntVar(&top, "n", 10, "Rows in each top-N list; 0 omits the list")
	fs.IntVar(&top, "top", 10, "Rows in each top-N list; 0 omits the list")

	// errors
	fs.IntVar(&errs, "errors", 5, "Malformed lines listed; 0 lists none (the total is still shown)")

	// jobs
	fs.IntVar(&workers, "jobs", runtime.GOMAXPROCS(0), "Maximum workers; 1 means no concurrent processing")

	// max errors
	fs.IntVar(&maxErrors, "max-errors", 0, "Stop after N malformed lines")

	// profile
	fs.StringVar(&cpuProfile, "cpuprofile", "", "Write a CPU profile")
	fs.StringVar(&memProfile, "memprofile", "", "Write a heap profile")
	fs.StringVar(&trace, "trace", "", "Write an execution trace")

	err := fs.Parse(args)
	if errors.Is(err, flag.ErrHelp) {
		return 0
	}
	if err != nil {
		return 2
	}

	if len(fs.Args()) == 0 {
		fs.Usage()
		return 2
	}

	logLensConfig := newLogLensConfig(
		format,
		cpuProfile,
		memProfile,
		trace,
		top,
		errs,
		workers,
		maxErrors,
	)

	err = logLensConfig.validate()
	if err != nil {
		fs.Usage()
		return 2
	}

	return 0
}
