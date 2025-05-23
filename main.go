package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"strconv"

	"github.com/nthskyradiated/go-bc/blockchain"
)

type CommandLine struct {
		BlockChain *blockchain.BlockChain
}

func (cli *CommandLine) printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  addblock -data DATA - Add a block to the blockchain")
	fmt.Println("  printchain - Print the blockchain")
}

func (cli *CommandLine) validateArgs() {
	if len(os.Args) < 2 {
		cli.printUsage()
		runtime.Goexit()
	}
}

func (cli *CommandLine) addBlock(data string) {
	cli.BlockChain.AddBlock(data)
	fmt.Println("Block added!")
}

func (cli *CommandLine) printChain() {
	iter := cli.BlockChain.Iterator()
	for  {
		block := iter.Next()
		fmt.Printf("Prev. hash: %x\n", block.PrevHash)
		fmt.Printf("Data: %s\n", block.Data)
		fmt.Printf("Hash: %x\n", block.Hash)
		pow := blockchain.NewProofOfWork(block)
		fmt.Printf("PoW: %s\n", strconv.FormatBool(pow.Validate()))
		if len(block.PrevHash) == 0 {
			break
		}
	}
}

func (cli *CommandLine) run() {
	cli.validateArgs()
	addBlockCmd := flag.NewFlagSet("addblock", flag.ExitOnError)
	printChainCmd := flag.NewFlagSet("print", flag.ExitOnError)
	addBlockData := addBlockCmd.String("block", "", "Block data")

	switch os.Args[1] {
	case "addblock":
		err := addBlockCmd.Parse(os.Args[2:])
		blockchain.HandleError(err)
	case "print":
		err := printChainCmd.Parse(os.Args[2:])
		blockchain.HandleError(err)
	default:
		cli.printUsage()
		runtime.Goexit()
	}

	if addBlockCmd.Parsed() {
		if *addBlockData == "" {
			addBlockCmd.Usage()
			runtime.Goexit()
		}
		cli.addBlock(*addBlockData)
	}

	if printChainCmd.Parsed() {
		cli.printChain()
	}
}

func main() {
	defer os.Exit(0)
	chain := blockchain.NewBlockChain()
	defer chain.Database.Close()
	cli := CommandLine{chain}
	cli.run()

}
