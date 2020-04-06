// +build !citest

package main

import (
	"os"

	"github.com/tada/mqtt-nats/cli"
)

func main() {
	os.Exit(cli.Bridge(os.Args, os.Stdout, os.Stderr))
}
