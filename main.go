package main

import (
	"fmt"
	"strconv"
	"github.com/nthskyradiated/go-bc/blockchain"
)



func main() {
	chain := blockchain.NewBlockChain()
	chain.AddBlock("First Block")
	chain.AddBlock("Second Block")

	for _, block := range chain.Blocks {
		fmt.Printf("Prev. hash: %x\n", block.PrevHash)
		fmt.Printf("Data: %s\n", block.Data)
		fmt.Printf("Hash: %x\n", block.Hash)
		
		pow := blockchain.NewProofOfWork(block)
		fmt.Printf("PoW: %s\n", strconv.FormatBool(pow.Validate()))
		// fmt.Printf("Nonce: %d\n", block.Nonce)
		fmt.Println()
	}
}
