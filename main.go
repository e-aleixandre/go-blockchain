package main

import (
	"github.com/e-aleixandre/go-blockchain/cli"
	"os"
)

func main() {
	defer os.Exit(0)

	cmd := cli.CommandLine{}
	cmd.Run()
}
