package wallet

import (
	"log"
	"github.com/mr-tron/base58"
)

func Base58Encode(input []byte) []byte {
	encoded := base58.Encode(input)
	if len(encoded) == 0 {
		log.Panic("Failed to encode input to Base58")
	}
	return []byte(encoded)
}

func Base58Decode(input string) []byte {
	decoded, err := base58.Decode(string(input[:]))
	if err != nil {
		log.Panicf("Failed to decode Base58 input: %v", err)
	}
	if len(decoded) == 0 {
		log.Panic("Decoded Base58 input is empty")
	}
	return decoded
}