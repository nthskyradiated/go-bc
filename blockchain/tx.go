package blockchain

import (
	"bytes"

	"github.com/nthskyradiated/go-bc/wallet"
)

type TxInput struct {
	ID       []byte
	OutIndex int
	Sig      []byte
	PubKey   []byte
}

// * TODO: implement the script
type TxOutput struct {
	Value        int
	ScriptPubKey []byte
}

func (in *TxInput) UsesKey(pubKeyHash []byte) bool {
	lockingHash := wallet.PublicKeyHash(in.PubKey)
	return bytes.Equal(lockingHash, pubKeyHash)
}

func (out *TxOutput) Lock(address []byte) {
	pubKeyHash := wallet.Base58Decode([]byte(address))
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-4]
	out.ScriptPubKey = pubKeyHash
}

func (out *TxOutput) IsLockedWithKey(pubKeyHash []byte) bool {
	// lockingHash := wallet.PublicKeyHash(out.ScriptPubKey)
	return bytes.Equal(out.ScriptPubKey, pubKeyHash)
}

func NewTXOutput(value int, address string) *TxOutput {
	out := &TxOutput{value, nil}
	out.Lock([]byte(address))
	return out
}