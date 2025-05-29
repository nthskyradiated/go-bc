package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/dgraph-io/badger"
)

const (
	dbPath = "./tmp/blocks_%s"
	genesisData = "Genesis Block"
)
type BlockChain struct {
	LastHash []byte
	Database *badger.DB
}

func DBExists(path string) bool {
	if _, err := os.Stat(path + "/MANIFEST"); os.IsNotExist(err) {
		return false
	}
	return true
}

func (bc *BlockChain) AddBlock(block *Block) {
	err := bc.Database.Update(func(txn *badger.Txn) error {
		if _, err := txn.Get(block.Hash); err == nil {
			return nil
		}

		blockData := block.Serialize()
		err := txn.Set(block.Hash, blockData)
		HandleError(err)

		item, err := txn.Get([]byte("lh"))
		HandleError(err)
		var lastHash []byte
		err = item.Value(func(val []byte) error {
			lastHash = append([]byte{}, val...)
			return nil
		})
		HandleError(err)

		item, err = txn.Get(lastHash)
		HandleError(err)
		var lastBlockData []byte
		err = item.Value(func(val []byte) error {
			lastBlockData = append([]byte{}, val...)
			return nil
		})
		HandleError(err)

		lastBlock := Deserialize(lastBlockData)

		if block.Height > lastBlock.Height {
			err = txn.Set([]byte("lh"), block.Hash)
			HandleError(err)
			bc.LastHash = block.Hash
		}

		return nil
	})
	HandleError(err)
}

func (bc *BlockChain) GetBlock(blockHash []byte) (Block, error) {
	var block Block

	err := bc.Database.View(func(txn *badger.Txn) error {
		if item, err := txn.Get(blockHash); err != nil {
			return errors.New("Block is not found")
		} else {
			var blockData []byte
			err = item.Value(func(val []byte) error {
				blockData = append([]byte{}, val...)
				return nil
			})
			HandleError(err)
			block = *Deserialize(blockData)
		}
		return nil
	})
	if err != nil {
		return block, err
	}

	return block, nil
}


func (bc *BlockChain) GetBestHeight() int {
	var lastBlock Block

	err := bc.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("lh"))
		HandleError(err)
		var lastHash []byte
		err = item.Value(func(val []byte) error {
			lastHash = append([]byte{}, val...)
			return nil
		})
		if err != nil {
			return err
		}

		item, err = txn.Get(lastHash)
		HandleError(err)
		var lastBlockData []byte
		err = item.Value(func(val []byte) error {
			lastBlockData = append([]byte{}, val...)
			return nil
		})

		lastBlock = *Deserialize(lastBlockData)

		return nil
	})
	HandleError(err)

	return lastBlock.Height
}

func (bc *BlockChain) GetBlockHashes() [][]byte {
	var blocks [][]byte

	iter := bc.Iterator()

	for {
		block := iter.Next()

		blocks = append(blocks, block.Hash)

		if len(block.PrevHash) == 0 {
			break
		}
	}

	return blocks
}

func (bc *BlockChain) MineBlock(transactions []*Transaction) *Block{
	var lastHash []byte
	var lastHeight int
	for _, tx := range transactions {
		if !bc.VerifyTransaction(tx) {
			log.Panic("Invalid Transaction")
		}
	}
	err := bc.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("lh"))
		HandleError(err)
		if err = item.Value(func (val []byte) error {
			lastHash = append([]byte{}, val...)
			return nil
		}); err != nil {
			return err
		}
		item, err = txn.Get(lastHash)
		HandleError(err)
		err = item.Value(func(val []byte) error {
			block := Deserialize(val)
			lastHeight = block.Height
			return err
		})
		HandleError(err)
		return nil
	})
	HandleError(err)
	newBlock := NewBlock(transactions, lastHash, lastHeight+1)
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

func NewBlockChain(address, nodeId string) *BlockChain {
	var lastHash []byte
	path := fmt.Sprintf(dbPath, nodeId)
	if DBExists(path) {
		fmt.Println("Blockchain already exists, loading from disk")
		runtime.Goexit()
	}
	opts := badger.DefaultOptions(path)
	// opts.Dir = dbPath
	// opts.ValueDir = dbPath
	db, err := openDB(path, opts)
	HandleError(err)
	err = db.Update(func(txn *badger.Txn) error {
		cbtx := CoinbaseTx(address, genesisData)
		genesis := GenesisBlock(cbtx)
		fmt.Println("Genesis Block Created")
		fmt.Printf("Continuing blockchain at %s\n", path)
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

func ContinueBlockChain(nodeId string) *BlockChain {
	path := fmt.Sprintf(dbPath, nodeId)
	
	if !DBExists(path) {
		fmt.Println("No existing blockchain found, create a new one")
		runtime.Goexit()
	}
	var lastHash []byte
	opts := badger.DefaultOptions(path)
	db, err := openDB(path, opts)
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
	
	if tx.IsCoinbase() {
		return true
	}

	prevTXs := make(map[string]Transaction)
	for _, in := range tx.Inputs {
		prevTX, err := chain.FindTransaction(in.ID)
		HandleError(err)
		prevTXs[hex.EncodeToString(prevTX.ID)] = prevTX
	}

	return tx.Verify(prevTXs)
}

func retry(dir string, originalOpts badger.Options) (*badger.DB, error) {
	lockPath := filepath.Join(dir, "LOCK")
	if err := os.Remove(lockPath); err != nil {
		return nil, fmt.Errorf(`removing "LOCK": %s`, err)
	}
	retryOpts := originalOpts
	retryOpts.Truncate = true
	db, err := badger.Open(retryOpts)
	return db, err
}

func openDB(dir string, opts badger.Options) (*badger.DB, error) {
	if db, err := badger.Open(opts); err != nil {
		if strings.Contains(err.Error(), "LOCK") {
			if db, err := retry(dir, opts); err == nil {
				log.Println("database unlocked, value log truncated")
				return db, nil
			}
			log.Println("could not unlock database:", err)
		}
		return nil, err
	} else {
		return db, nil
	}
}