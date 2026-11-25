// Package cli implements loglens's command line: it parses and validates flags
// and arguments, runs the analysis, and maps the outcome to an exit code.
package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"runtime"
)

// Run executes loglens with the given command-line arguments, which exclude the
// program name, and returns the process exit code. The report is written to
// stdout; usage messages and errors are written to stderr.
func Run(args []string, stdout, stderr io.Writer) int {
	var c logLensConfig

	fs := flag.NewFlagSet("loglens", flag.ContinueOnError)
	fs.SetOutput(stderr)

	// format
	fs.StringVar(&c.Format, "format", "text", "Output report format")

	// top
	fs.IntVar(&c.Top, "n", 10, "Rows in each top-N list; 0 omits the list")
	fs.IntVar(&c.Top, "top", 10, "Rows in each top-N list; 0 omits the list")

	// errors
	fs.IntVar(&c.Errors, "errors", 5, "Malformed lines listed; 0 lists none (the total is still shown)")

	// jobs
	fs.IntVar(&c.Workers, "jobs", runtime.GOMAXPROCS(0), "Maximum workers; 1 means no concurrent processing")

	// max errors
	fs.IntVar(&c.MaxErrors, "max-errors", 0, "Stop after N malformed lines")

	// profile
	fs.StringVar(&c.CPUProfile, "cpuprofile", "", "Write a CPU profile")
	fs.StringVar(&c.MemProfile, "memprofile", "", "Write a heap profile")
	fs.StringVar(&c.Trace, "trace", "", "Write an execution trace")

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

	err = c.validate()
	if err != nil {
		fmt.Fprintln(stderr, err)
		fs.Usage()
		return 2
	}

	return 0
}
