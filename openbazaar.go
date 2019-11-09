package main

import (
	"github.com/cpacia/openbazaar3.0/cmd"
	"github.com/jessevdk/go-flags"
	"os"
)

func main() {
	parser := flags.NewParser(nil, flags.Default)

	parser.AddCommand("start",
		"start the OpenBazaar node",
		"The start command starts the OpenBazaar node",
		&cmd.Start{})
	parser.AddCommand("init",
		"initialize an OpenBazaar node",
		"The init command creates and initializes a new data directory and database.",
		&cmd.Init{})

	if _, err := parser.Parse(); err != nil {
		os.Exit(1)
	}
}
