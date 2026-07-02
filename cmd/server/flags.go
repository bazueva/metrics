package main

import (
	"flag"

	config2 "github.com/bazueva/metrics/cmd/config"
)

func parseFlags(config *config2.ServerAddr) {
	flag.Var(config, "a", "address http server")
	flag.Parse()
}
