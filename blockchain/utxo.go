package blockchain

import (
	"bytes"
	"encoding/hex"
	"log"

	"github.com/dgraph-io/badger"
)

var (
	utxoPrefix   = []byte("utxo-")
	prefixLength = len(utxoPrefix)
)

type UTXOSet struct {
	Blockchain *BlockChain
}

func (u UTXOSet) Reindex() {
	db := u.Blockchain.Database
	u.DeleteByPrefix(utxoPrefix)
	UTXO := u.Blockchain.FindUTXO()

	err := db.Update(func(txn *badger.Txn) error {
		for txID, outputs := range UTXO {
			key, err := hex.DecodeString(txID)
			if err != nil {
				return err
			}
			key = append(utxoPrefix, key...)
			err = txn.Set(key, outputs.Serialize())
			HandleError(err)
		}
		return nil
	})
	HandleError(err)
}

func (u *UTXOSet) Update(block *Block) {
	db := u.Blockchain.Database
	err := db.Update(func(txn *badger.Txn) error {
		for _, tx := range block.Transactions {
			if !tx.IsCoinbase() {
				for _, input := range tx.Inputs {
				updatedOuts := TxOutputs{}
				inID := append(utxoPrefix, input.ID...)
					item, err := txn.Get(inID)
				HandleError(err)
				v, err := item.ValueCopy(nil)
				HandleError(err)

				outs := DeserializeOutputs(v)
				for outIdx, out := range outs.Outputs {
					if outIdx != input.OutIndex {
						updatedOuts.Outputs = append(updatedOuts.Outputs, out)
					}
				}
				if len(updatedOuts.Outputs) == 0 {
					if err := txn.Delete(inID); err != nil {
						log.Panic(err)
					}
				} else {
					if err := txn.Set(inID, updatedOuts.Serialize()); err != nil {
						log.Panic(err)
					}
				}
			}
		}
		newOutputs := TxOutputs{}
			newOutputs.Outputs = append(newOutputs.Outputs, tx.Outputs...)
			txID := append(utxoPrefix, tx.ID...)
			if err := txn.Set(txID, newOutputs.Serialize()); err != nil {
				log.Panic(err)
			}
		}
		return nil
	})
	HandleError(err)
}

func (u *UTXOSet) CountTransactions() int {
	db := u.Blockchain.Database
	count := 0
	err := db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Seek(utxoPrefix); it.ValidForPrefix(utxoPrefix); it.Next() {
			count++
		}
		return nil
	})
	HandleError(err)
	return count
}

func (u *UTXOSet) DeleteByPrefix(prefix []byte) {
	deleteKeys := func(keysForDelete [][]byte) error {
		if err := u.Blockchain.Database.Update(func(txn *badger.Txn) error {
			for _, key := range keysForDelete {
				if err := txn.Delete(key); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return err
		}
		return nil
	}

	collectSize := 100000
	u.Blockchain.Database.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()

		keysForDelete := make([][]byte, 0, collectSize)
		keysCollected := 0
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			key := it.Item().KeyCopy(nil)
			keysForDelete = append(keysForDelete, key)
			keysCollected++
			if keysCollected == collectSize {
				if err := deleteKeys(keysForDelete); err != nil {
					log.Panic(err)
				}
				keysForDelete = make([][]byte, 0, collectSize)
				keysCollected = 0
			}
		}
		if keysCollected > 0 {
			if err := deleteKeys(keysForDelete); err != nil {
				log.Panic(err)
			}
		}
		return nil

	})

}

func (u UTXOSet) FindUnspentTransactions(pubKeyHash []byte) []TxOutput {
	var unspentTxs []TxOutput

	db := u.Blockchain.Database
	err := db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Seek(utxoPrefix); it.ValidForPrefix(utxoPrefix); it.Next() {
			item := it.Item()
			v, err := item.ValueCopy(nil)
			HandleError(err)
			outs := DeserializeOutputs(v)
			for _, out := range outs.Outputs {
				if out.IsLockedWithKey(pubKeyHash) {
					unspentTxs = append(unspentTxs, out)
				}
			}
		}
		return nil
	})
	HandleError(err)	
	return unspentTxs
}




func (u UTXOSet) FindSpendableOutputs(pubKeyHash []byte, amount int) (int, map[string][]int) {
	unspentOuts := make(map[string][]int)
	accumulated := 0
	db := u.Blockchain.Database

	err := db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions

		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Seek(utxoPrefix); it.ValidForPrefix(utxoPrefix); it.Next() {
			item := it.Item()
			k := item.Key()
			var v []byte
			v, err := item.ValueCopy(nil)
			HandleError(err)
			k = bytes.TrimPrefix(k, utxoPrefix)
			txID := hex.EncodeToString(k)
			outs := DeserializeOutputs(v)

			for outIdx, out := range outs.Outputs {
				if out.IsLockedWithKey(pubKeyHash) && accumulated < amount {
					accumulated += out.Value
					unspentOuts[txID] = append(unspentOuts[txID], outIdx)
				}
			}
		}
		return nil
	})
	HandleError(err)
	return accumulated, unspentOuts
}