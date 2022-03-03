package main

import (
	"os"

	"github.com/slowteetoe/blockchain/cli"
)

func main() {
	defer os.Exit(0)
	cli := cli.CommandLine{}
	cli.Run()

}
