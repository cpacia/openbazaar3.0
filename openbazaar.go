package main

import (
	"github.com/cpacia/openbazaar3.0/cmd"
	"github.com/jessevdk/go-flags"
	"log"
	"os"
)

func main() {
	parser := flags.NewParser(nil, flags.Default)

	_, err := parser.AddCommand("start",
		"start the OpenBazaar node",
		"The start command starts the OpenBazaar node",
		&cmd.Start{})
	if err != nil {
		log.Fatal(err)
	}
	_, err = parser.AddCommand("init",
		"initialize an OpenBazaar node",
		"The init command creates and initializes a new data directory and database.",
		&cmd.Init{})
	if err != nil {
		log.Fatal(err)
	}
	_, err = parser.AddCommand("devnet",
		"start a local dev net",
		"The devnet command spins up a local network of three nodes (buyer, vendor, moderator)"+
			"that connects all three nodes together and uses a mock wallet and mock coins. Each node is pre-populated"+
			"with data for ease of use.",
		&cmd.DevNet{})
	if err != nil {
		log.Fatal(err)
	}

	if _, err := parser.Parse(); err != nil {
		os.Exit(1)
	}
}
