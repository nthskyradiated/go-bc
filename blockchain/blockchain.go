package blockchain

import (
	"fmt"

	"github.com/dgraph-io/badger"
)

const (
	dbPath = "./tmp/blocks"
)
type BlockChain struct {
	LastHash []byte
	Database *badger.DB
}

type BlockChainIterator struct {
	CurrentHash []byte
	Database    *badger.DB
}

func (bc *BlockChain) AddBlock(data string) {
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
	newBlock := NewBlock(data, lastHash)
	err = bc.Database.Update(func(txn *badger.Txn) error {
		err := txn.Set(newBlock.Hash, newBlock.Serialize())
		HandleError(err)
		err = txn.Set([]byte("lh"), newBlock.Hash)
		bc.LastHash = newBlock.Hash
		return err
	})
	HandleError(err)
}

func NewBlockChain() *BlockChain {
	var lastHash []byte
	opts := badger.DefaultOptions(dbPath)
	// opts.Dir = dbPath
	// opts.ValueDir = dbPath
	db, err := badger.Open(opts)
	HandleError(err)
	err = db.Update(func(txn *badger.Txn) error {
		if _, err := txn.Get([]byte("lh")); err == badger.ErrKeyNotFound {
			fmt.Println("No last hash found, creating genesis block")
			genesis := GenesisBlock()
			fmt.Println("Genesis block created")
			err = txn.Set(genesis.Hash, genesis.Serialize())
			HandleError(err)
			err = txn.Set([]byte("lh"), genesis.Hash)
			lastHash = genesis.Hash
			return err
		} else {
			item, err := txn.Get([]byte("lh"))
			HandleError(err)
			err = item.Value(func (val []byte) error {
			lastHash = append([]byte{}, val...)
			return nil
		})
			return err
		}
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