package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"runtime"

	"github.com/dgraph-io/badger"
)

const (
	dbPath = "./tmp/blocks"
	dbFile = "./tmp/blocks/MANIFEST"
	genesisData = "First Transaction from Genesis"
)
type BlockChain struct {
	LastHash []byte
	Database *badger.DB
}

type BlockChainIterator struct {
	CurrentHash []byte
	Database    *badger.DB
}

func DBExists() bool {
	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		return false
	}
	return true
}

func (bc *BlockChain) AddBlock(transactions []*Transaction) *Block{
	var lastHash []byte
	err := bc.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("lh"))
		HandleError(err)
		err = item.Value(func (val []byte) error {
			lastHash = append([]byte{}, val...)
			return nil
		})
		return err
	})
	HandleError(err)
	newBlock := NewBlock(transactions, lastHash)
	err = bc.Database.Update(func(txn *badger.Txn) error {
		err := txn.Set(newBlock.Hash, newBlock.Serialize())
		HandleError(err)
		err = txn.Set([]byte("lh"), newBlock.Hash)
		bc.LastHash = newBlock.Hash
		return err
	})
	HandleError(err)
	return newBlock
}

func NewBlockChain(address string) *BlockChain {
	var lastHash []byte

	if DBExists() {
		fmt.Println("Blockchain already exists, loading from disk")
		runtime.Goexit()
	}
	opts := badger.DefaultOptions(dbPath)
	// opts.Dir = dbPath
	// opts.ValueDir = dbPath
	db, err := badger.Open(opts)
	HandleError(err)
	err = db.Update(func(txn *badger.Txn) error {
	cbtx := CoinbaseTx(address, genesisData)
		genesis := GenesisBlock(cbtx)
		fmt.Println("Genesis Block Created")
		err := txn.Set(genesis.Hash, genesis.Serialize())
		HandleError(err)
		err = txn.Set([]byte("lh"), genesis.Hash)
		lastHash = genesis.Hash
		return err

	})
	HandleError(err)
	bc := BlockChain{lastHash, db}
	return &bc
}

func ContinueBlockChain(address string) *BlockChain {
	if !DBExists() {
		fmt.Println("No existing blockchain found, create a new one")
		runtime.Goexit()
	}
	var lastHash []byte
	opts := badger.DefaultOptions(dbPath)
	db, err := badger.Open(opts)
	HandleError(err)
	err = db.Update(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("lh"))
		HandleError(err)
			err = item.Value(func(val []byte) error {
			lastHash = append([]byte{}, val...)
			return nil
		})
		return err
	})
	HandleError(err)
	bc := BlockChain{lastHash, db}
	return &bc
}

func (bc *BlockChain) Iterator() *BlockChainIterator {
	iter := &BlockChainIterator{bc.LastHash, bc.Database}
	return iter
}

func (iter *BlockChainIterator) Next() *Block {
	var block *Block
	err := iter.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get(iter.CurrentHash)
		HandleError(err)
		var encodedBlock []byte
		err = item.Value(func(val []byte) error {
			encodedBlock = append([]byte{}, val...)
			return nil
		})
		block = Deserialize(encodedBlock)
		return err
	})
	HandleError(err)
	iter.CurrentHash = block.PrevHash
	return block
}

func (chain *BlockChain) FindUnspentTransactions(pubKeyHash []byte) []Transaction {
	var unspentTxs []Transaction
	spentTXOs := make(map[string][]int)

	iter := chain.Iterator()
	for {
		block := iter.Next()
		for _, tx := range block.Transactions {
			txID := hex.EncodeToString(tx.ID)

			Outputs:
			for outIdx, out := range tx.Outputs {
				if spentTXOs[txID] != nil {
					for _, spentOutIdx := range spentTXOs[txID] {
						if spentOutIdx == outIdx {
							continue Outputs
						}
					}
				}
				if out.IsLockedWithKey(pubKeyHash) {
					unspentTxs = append(unspentTxs, *tx)
					fmt.Printf("Found unspent transaction: %s\n", txID)
				}
			}

			if !tx.IsCoinbase() {
				for _, in := range tx.Inputs {
					if in.UsesKey(pubKeyHash) {
						inID := hex.EncodeToString(in.ID)
						spentTXOs[inID] = append(spentTXOs[inID], in.OutIndex)
					}
				}
			}

		}
		if len(block.PrevHash) == 0 {
			break
		}
	}
	return unspentTxs
}

func (chain *BlockChain) FindUTXO() map[string]TxOutputs {
	UTXOs := make(map[string]TxOutputs) 
	spentTxos := make(map[string][]int)
	iter := chain.Iterator()
	for {
		block := iter.Next()
		for _, tx := range block.Transactions {
			txID := hex.EncodeToString(tx.ID)

			Outputs:
			for outIdx, out := range tx.Outputs {
				if spentTxos[txID] != nil {
					for _, spentOutIdx := range spentTxos[txID] {
						if spentOutIdx == outIdx {
							continue Outputs
						}
					}
				}
				outs := UTXOs[txID]
				outs.Outputs = append(outs.Outputs, out)
				UTXOs[txID] = outs
		}
			if !tx.IsCoinbase() {
				for _, in := range tx.Inputs {
						inID := hex.EncodeToString(in.ID)
						spentTxos[inID] = append(spentTxos[inID], in.OutIndex)
				}
			}
	}
		if len(block.PrevHash) == 0 {
			break
		}
	}	
	return UTXOs
}

func (chain *BlockChain) FindSpendableOutputs(pubKeyHash []byte, amount int) (int, map[string][]int) {
	unspentOutputs := make(map[string][]int)
	unspentTxs := chain.FindUnspentTransactions(pubKeyHash)
	accumulated := 0

	Work:
	for _, tx := range unspentTxs {
		txID := hex.EncodeToString(tx.ID)
		for outIdx, out := range tx.Outputs {
			if out.IsLockedWithKey(pubKeyHash) && accumulated < amount {
				accumulated += out.Value
				unspentOutputs[txID] = append(unspentOutputs[txID], outIdx)
			}
			if accumulated >= amount {
				break Work
			}
		}
	}
	return accumulated, unspentOutputs
}

func (chain *BlockChain) FindTransaction(ID []byte) (Transaction, error) {
	iter := chain.Iterator()
	for {
		block := iter.Next()
		for _, tx := range block.Transactions {
			if bytes.Equal(tx.ID, ID) {
				return *tx, nil
			}
		}
		if len(block.PrevHash) == 0 {
			break
		}
	}
	return Transaction{}, errors.New("transaction not found")

}

func (chain *BlockChain) SignTransaction(tx *Transaction, privateKey ecdsa.PrivateKey) {
	prevTXs := make(map[string]Transaction)
	for _, in := range tx.Inputs {
		prevTX, err := chain.FindTransaction(in.ID)
		HandleError(err)
		prevTXs[hex.EncodeToString(prevTX.ID)] = prevTX
	}
	tx.Sign(privateKey, prevTXs)
}

func (chain *BlockChain) VerifyTransaction(tx *Transaction) bool {

	prevTXs := make(map[string]Transaction)
	for _, in := range tx.Inputs {
		prevTX, err := chain.FindTransaction(in.ID)
		HandleError(err)
		prevTXs[hex.EncodeToString(prevTX.ID)] = prevTX
	}

	return tx.Verify(prevTXs)
}