package main

import (
	"github.com/cpacia/openbazaar3.0/cmd"
	"github.com/jessevdk/go-flags"
	"github.com/whyrusleeping/go-logging"
	"os"
)

var log = logging.MustGetLogger("main")

var parser = flags.NewParser(nil, flags.Default)

func main() {
	parser.AddCommand("start",
		"start the OpenBazaar server",
		"The start command starts the OpenBazaar server",
		&cmd.Start{})

	if _, err := parser.Parse(); err != nil {
		os.Exit(1)
	}
}
