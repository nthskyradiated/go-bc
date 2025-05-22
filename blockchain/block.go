package blockchain
import (

)

type Block struct {
	Hash []byte
	Data []byte
	PrevHash []byte
	Nonce int
}

// func (b *Block) SetHash() {
// info := bytes.Join([][]byte{b.PrevHash, b.Data}, []byte{})
// 	hash := sha256.Sum256(info)
// 	b.Hash = hash[:]
// }


type BlockChain struct {
	Blocks []*Block
}
func (bc *BlockChain) AddBlock(data string) {
	prevBlock := bc.Blocks[len(bc.Blocks)-1]
	newBlock := NewBlock(data, prevBlock.Hash)
	bc.Blocks = append(bc.Blocks, newBlock)
}

func GenesisBlock() *Block {
	return NewBlock("Genesis Block", []byte{})
}


func NewBlock(data string, prevHash []byte) *Block {
	block := &Block{[]byte{}, []byte(data), prevHash, 0}
	// block.SetHash()
	pow := NewProofOfWork(block)
	nonce, hash := pow.Run()

	block.Hash = hash[:]
	block.Nonce = nonce
	return block
}

func NewBlockChain() *BlockChain {
	genesisBlock := GenesisBlock()
	return &BlockChain{[]*Block{genesisBlock}}
}