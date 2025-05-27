package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"strings"

	"github.com/nthskyradiated/go-bc/wallet"
)

type Transaction struct {
	ID      []byte
	Inputs  []TxInput
	Outputs []TxOutput
}

func (tx Transaction) Serialize() []byte{
	var encoded bytes.Buffer
	encode := gob.NewEncoder(&encoded)
	err := encode.Encode(tx)
	if err != nil {
		log.Panicf("Failed to serialize transaction: %v", err)
	}
	return encoded.Bytes()
}

func (tx *Transaction) Hash() []byte {
	var hash [32]byte
	txCopy := *tx
	txCopy.ID = nil // Clear the ID to avoid using it in the hash
	hash = sha256.Sum256(txCopy.Serialize())
	return hash[:]
}

func (tx *Transaction) IsCoinbase() bool {
	return len(tx.Inputs) == 1 && len(tx.Inputs[0].ID) == 0 && tx.Inputs[0].OutIndex == -1
}

func (tx *Transaction) Sign(privKey ecdsa.PrivateKey, prevTXs map[string]Transaction) {
	if tx.IsCoinbase() {
		return
	}

	for _, input := range tx.Inputs {
		if prevTXs[hex.EncodeToString(input.ID)].ID == nil {
			log.Panicf("Previous transaction not found: %x", input.ID)
		}
	}
	txCopy := tx.TrimmedCopy()

	for i, input := range txCopy.Inputs {
		prevTX := prevTXs[hex.EncodeToString(input.ID)]
		txCopy.Inputs[i].Sig = nil // Clear the signature for signing
		txCopy.Inputs[i].PubKey = prevTX.Outputs[input.OutIndex].ScriptPubKey
		txCopy.ID = txCopy.Hash()
		txCopy.Inputs[i].PubKey = nil

		r, s, err := ecdsa.Sign(rand.Reader, &privKey, txCopy.ID)
		HandleError(err)
		signature := append(r.Bytes(), s.Bytes()...)
		tx.Inputs[i].Sig = signature
	}
}

func (tx *Transaction) TrimmedCopy() Transaction {
	var outputs []TxOutput
	var inputs []TxInput
	for _, in := range tx.Inputs {
		inputs = append(inputs, TxInput{in.ID, in.OutIndex, nil, nil})
	}
	for _, out := range tx.Outputs {
		outputs = append(outputs, TxOutput{out.Value, out.ScriptPubKey})
	}
	txCopy := Transaction{ tx.ID, inputs, outputs }
	return txCopy
}

func (tx *Transaction) Verify(prevTXs map[string]Transaction) bool {
	if tx.IsCoinbase() {
		return true
	}

	for _, input := range tx.Inputs {
		if prevTXs[hex.EncodeToString(input.ID)].ID == nil {
			log.Panicf("Previous transaction not found: %x", input.ID)
		}
	}

	txCopy := tx.TrimmedCopy()
	curve := elliptic.P256()
	for i, input := range tx.Inputs {
		prevTX := prevTXs[hex.EncodeToString(input.ID)]
		txCopy.Inputs[i].Sig = nil // Clear the signature for verification
		txCopy.Inputs[i].PubKey = prevTX.Outputs[input.OutIndex].ScriptPubKey
		txCopy.ID = txCopy.Hash()
		txCopy.Inputs[i].PubKey = nil

		r := new(big.Int).SetBytes(input.Sig[:len(input.Sig)/2])
		s := new(big.Int).SetBytes(input.Sig[len(input.Sig)/2:])


		x := new(big.Int).SetBytes(input.PubKey[:len(input.PubKey)/2])
		y := new(big.Int).SetBytes(input.PubKey[len(input.PubKey)/2:])

		rawPubKey := ecdsa.PublicKey{Curve: curve, X: x, Y: y}

		if !ecdsa.Verify(&rawPubKey, txCopy.ID, r, s) {
			return false
		}
	}
	return true
}


func (tx *Transaction) SetID() {
	var encoded bytes.Buffer
	var hash [32]byte
	encode := gob.NewEncoder(&encoded)
	err := encode.Encode(tx)
	HandleError(err)
	hash = sha256.Sum256(encoded.Bytes())
	tx.ID = hash[:]
	
}

func NewTransaction(from, to string, amount int, UTXO *UTXOSet) *Transaction {
	var inputs []TxInput
	var outputs []TxOutput

	wallets, err := wallet.NewWallets()
	HandleError(err)
	w := wallets.GetWallet(from)
	pubKeyHash := wallet.PublicKeyHash(w.PublicKey)

	acc, validOutputs := UTXO.FindSpendableOutputs(pubKeyHash, amount)

	if acc < amount {
		log.Panicf("Not enough funds: %d < %d", acc, amount)
	}

	for txid, outs := range validOutputs {
		txID, err := hex.DecodeString(txid)
		HandleError(err)
		for _, out := range outs {
			input := TxInput{txID, out, nil, w.PublicKey}
			inputs = append(inputs, input)
		}
	}
	outputs = append(outputs, *NewTXOutput(amount, to))
	if acc > amount {
		outputs = append(outputs, *NewTXOutput(acc - amount, from))
	}
	tx := Transaction{nil, inputs, outputs}
	tx.ID = tx.Hash()
	UTXO.Blockchain.SignTransaction(&tx, w.PrivateKey)
	return &tx
}

func CoinbaseTx(to, data string) *Transaction {
	if data == "" {
		data = fmt.Sprintf("Coins to %s", to)
	}

	txin := TxInput{[]byte{}, -1, nil, []byte(data)}
	txout := NewTXOutput(100, to)

	tx := Transaction{nil, []TxInput{txin}, []TxOutput{*txout}}
	tx.SetID()

	return &tx
}

func (tx Transaction) String() string {
	var lines []string

	lines = append(lines, fmt.Sprintf("--- Transaction %x:", tx.ID))
	for i, input := range tx.Inputs {
		lines = append(lines, fmt.Sprintf("     Input %d:", i))
		lines = append(lines, fmt.Sprintf("       TXID:     %x", input.ID))
		lines = append(lines, fmt.Sprintf("       Out:       %d", input.OutIndex))
		lines = append(lines, fmt.Sprintf("       Signature: %x", input.Sig))
		lines = append(lines, fmt.Sprintf("       PubKey:    %x", input.PubKey))
	}

	for i, output := range tx.Outputs {
		lines = append(lines, fmt.Sprintf("     Output %d:", i))
		lines = append(lines, fmt.Sprintf("       Value:  %d", output.Value))
		lines = append(lines, fmt.Sprintf("       Script: %x", output.ScriptPubKey))
	}

	return strings.Join(lines, "\n")
}