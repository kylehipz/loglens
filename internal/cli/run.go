// Package cli Flag and Args Validation.
package cli

import (
	"errors"
	"flag"
	"io"
	"runtime"
)

func Run(args []string, stdout, stderr io.Writer) int {
	var logLensConfig LogLensConfig

	fs := flag.NewFlagSet("loglens", flag.ContinueOnError)
	fs.SetOutput(stderr)

	// format
	fs.StringVar(&logLensConfig.Format, "format", "text", "Output report format")

	// top
	fs.IntVar(&logLensConfig.Top, "n", 10, "Rows in each top-N list; 0 omits the list")
	fs.IntVar(&logLensConfig.Top, "top", 10, "Rows in each top-N list; 0 omits the list")

	// errors
	fs.IntVar(&logLensConfig.Errors, "errors", 5, "Malformed lines listed; 0 lists none (the total is still shown)")

	// jobs
	fs.IntVar(&logLensConfig.Workers, "jobs", runtime.GOMAXPROCS(0), "Maximum workers; 1 means no concurrent processing")

	// max errors
	fs.IntVar(&logLensConfig.MaxErrors, "max-errors", 0, "Stop after N malformed lines")

	// profile
	fs.StringVar(&logLensConfig.CPUProfile, "cpuprofile", "", "Write a CPU profile")
	fs.StringVar(&logLensConfig.MemProfile, "memprofile", "", "Write a heap profile")
	fs.StringVar(&logLensConfig.Trace, "trace", "", "Write an execution trace")

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

	err = logLensConfig.validate()
	if err != nil {
		fs.Usage()
		return 2
	}

	return 0
}
