// Loglens analyzes access logs.
package main

import (
	"os"

	"github.com/kylehipz/loglens/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
