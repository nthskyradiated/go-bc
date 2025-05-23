package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"math/big"
)

// Take data from the block

// Create a counter (nonce) which starts at 0

// Create a hash of the block data + nonce

// Check the hash if it meets a set of requirements (difficulty)

// Requirements:
// 1. The hash must start with a certain number of zeros
// 2. The hash must be less than a target value

const Difficulty = 18
type ProofOfWork struct {
	Block  *Block
	Target *big.Int
}

func NewProofOfWork(b *Block) *ProofOfWork {
	target := big.NewInt(1)
	target.Lsh(target, uint(256-Difficulty))
	pow := &ProofOfWork{b, target}
	return pow
}

func (pow *ProofOfWork) PrepareData(nonce int) []byte {
	data := bytes.Join([][]byte{
		pow.Block.PrevHash,
		pow.Block.Data,
		ToHex(int64(nonce)),
		ToHex(int64(Difficulty)),

	}, []byte{})
	return data
}

func (pow *ProofOfWork) Run() (int, []byte) {
	var initHash = new(big.Int)
	nonce := 0
	var hash [32]byte
	for nonce < math.MaxInt64 {
		data := pow.PrepareData(nonce)
		hash = sha256.Sum256(data)
		fmt.Printf("Nonce: %d, Hash: %x\n", nonce, hash)
		initHash.SetBytes(hash[:])
		if initHash.Cmp(pow.Target) == -1 {
			break
		} else {
			nonce++
		}
	}
	fmt.Println("Mining Success!")
	return nonce, hash[:]
}

func (pow *ProofOfWork) Validate() bool {
	var initHash = new(big.Int)
	data := pow.PrepareData(pow.Block.Nonce)
	hash := sha256.Sum256(data)
	initHash.SetBytes(hash[:])
	return initHash.Cmp(pow.Target) == -1
}

func ToHex(num int64)[]byte {
	buff := new(bytes.Buffer)
	err := binary.Write(buff, binary.BigEndian, num)
	if err != nil {
		log.Panic(err)
	}
	return buff.Bytes()
}