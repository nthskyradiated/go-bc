package main

import (
	"os"

	"github.com/nthskyradiated/go-bc/cli"
	"github.com/nthskyradiated/go-bc/wallet"
)



func main() {
	defer os.Exit(0)
	cli := cli.CommandLine{}
	cli.Run()

	w := wallet.MakeWallet()
	w.Address()
}
