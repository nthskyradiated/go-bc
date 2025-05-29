package blockchain

import "github.com/dgraph-io/badger"

type BlockChainIterator struct {
	CurrentHash []byte
	Database    *badger.DB
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
