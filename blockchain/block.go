package blockchain

import (
	"bytes"
	"encoding/gob"
	"log"
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

func (b *Block) Serialize() []byte {
	var res bytes.Buffer
	encoder := gob.NewEncoder(&res)

	err := encoder.Encode(b)
	HandleError(err)
	return res.Bytes()
}

func Deserialize(data []byte) *Block {
	var block Block
	decoder := gob.NewDecoder(bytes.NewReader(data))
	err := decoder.Decode(&block)
	HandleError(err)
	return &block
}

func HandleError(err error) {
	if err != nil {
		log.Panic(err)
	}
}