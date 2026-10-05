// Command otelstarter sets up OpenTelemetry observability for an application.
//
// main stays tiny on purpose: all behaviour lives in internal/cli so it can be
// tested without building a binary.
package main

import (
	"os"

	"github.com/Murage-Ivy/otelstarter/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
