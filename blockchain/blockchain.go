package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/dgraph-io/badger/v4"
	"log"
	"os"
	"runtime"
)

const (
	dbPath      = "./tmp/blocks"
	dbFile      = "./tmp/blocks/MANIFEST"
	genesisData = "This is where it all started"
)

type Blockchain struct {
	LastHash []byte
	Database *badger.DB
}

func InitBlockchain(address string) *Blockchain {
	var lastHash []byte

	if DBExists() {
		fmt.Println("Blockchain already exists!")
		runtime.Goexit()
	}

	opts := badger.DefaultOptions(dbPath)
	opts.Logger = nil
	db, err := badger.Open(opts)

	if err != nil {
		log.Panic(err)
	}

	err = db.Update(func(txn *badger.Txn) error {
		cbtx := CoinbaseTx(address, genesisData)
		genesis := Genesis(cbtx)
		fmt.Println("Genesis created")

		err := txn.Set(genesis.Hash, genesis.Serialize())

		if err != nil {
			log.Panic(err)
		}

		err = txn.Set([]byte("lh"), genesis.Hash)
		lastHash = genesis.Hash

		return err
	})

	if err != nil {
		log.Panic(err)
	}

	return &Blockchain{lastHash, db}
}

func ContinueBlockchain(address string) *Blockchain {
	if !DBExists() {
		fmt.Println("No existing blockchain found")
		runtime.Goexit()
	}

	var lastHash []byte

	opts := badger.DefaultOptions(dbPath)
	opts.Logger = nil
	db, err := badger.Open(opts)

	if err != nil {
		log.Panic(err)
	}

	err = db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("lh"))

		if err != nil {
			return err
		}

		err = item.Value(func(val []byte) error {
			lastHash = val

			return nil
		})

		return err
	})

	if err != nil {
		log.Panic(err)
	}

	return &Blockchain{lastHash, db}
}

func DBExists() bool {
	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		return false
	}

	return true
}

func (chain *Blockchain) AddBlock(transactions []*Transaction) {
	var lastHash []byte

	err := chain.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("lh"))

		if err != nil {
			log.Panic(err)
		}

		err = item.Value(func(val []byte) error {
			lastHash = append([]byte{}, val...)

			return nil
		})

		return err
	})

	if err != nil {
		log.Panic(err)
	}

	newBlock := CreateBlock(transactions, lastHash)

	err = chain.Database.Update(func(txn *badger.Txn) error {
		err := txn.Set(newBlock.Hash, newBlock.Serialize())

		if err != nil {
			return err
		}

		err = txn.Set([]byte("lh"), newBlock.Hash)
		chain.LastHash = newBlock.Hash

		return err
	})

	if err != nil {
		log.Panic(err)
	}
}

func (chain *Blockchain) FindUnspentTransactions(publicKeyHash []byte) []Transaction {
	var unspentTxs []Transaction

	spentTxOutputs := make(map[string][]int)

	iter := chain.Iterator()

	for {
		block := iter.Next()

		for _, tx := range block.Transactions {
			txID := hex.EncodeToString(tx.ID)

		Outputs:
			for outIdx, out := range tx.Outputs {
				if spentTxOutputs[txID] != nil {
					for _, spentOut := range spentTxOutputs[txID] {
						if spentOut == outIdx {
							continue Outputs
						}

					}
				}

				if out.IsLockedWithKey(publicKeyHash) {
					unspentTxs = append(unspentTxs, *tx)
				}
			}

			if !tx.IsCoinbase() {
				for _, in := range tx.Inputs {
					if in.UsesKey(publicKeyHash) {
						inTxID := hex.EncodeToString(in.ID)
						spentTxOutputs[inTxID] = append(spentTxOutputs[inTxID], in.Out)
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

func (chain *Blockchain) FindUTXO(pubKeyHash []byte) []TxOutput {
	var UTXOs []TxOutput

	transactions := chain.FindUnspentTransactions(pubKeyHash)

	for _, tx := range transactions {
		for _, out := range tx.Outputs {
			if out.IsLockedWithKey(pubKeyHash) {
				UTXOs = append(UTXOs, out)
			}
		}
	}

	return UTXOs
}

func (chain *Blockchain) FindSpendableOutputs(pubKeyHash []byte, amount int) (int, map[string][]int) {
	unspentOuts := make(map[string][]int)
	unspentTxs := chain.FindUnspentTransactions(pubKeyHash)
	accumulated := 0

Work:
	for _, tx := range unspentTxs {
		txID := hex.EncodeToString(tx.ID)

		for outIdx, out := range tx.Outputs {
			accumulated += out.Value
			unspentOuts[txID] = append(unspentOuts[txID], outIdx)

			if accumulated >= amount {
				break Work
			}
		}
	}

	return accumulated, unspentOuts
}

type Iterator struct {
	CurrentHash []byte
	Database    *badger.DB
}

func (chain *Blockchain) Iterator() *Iterator {
	iter := &Iterator{chain.LastHash, chain.Database}

	return iter
}

func (iterator *Iterator) Next() *Block {
	var block *Block

	err := iterator.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get(iterator.CurrentHash)

		if err != nil {
			return err
		}

		err = item.Value(func(val []byte) error {
			block = Deserialize(val)

			return nil
		})

		return err
	})

	if err != nil {
		log.Panic(err)
	}

	iterator.CurrentHash = block.PrevHash

	return block
}

func (chain *Blockchain) ShutdownDB() {
	err := chain.Database.Close()

	if err != nil {
		log.Panic(err)
	}
}
func (chain *Blockchain) FindTransaction(ID []byte) (Transaction, error) {
	iter := chain.Iterator()

	for {
		block := iter.Next()

		for _, tx := range block.Transactions {
			if bytes.Compare(tx.ID, ID) == 0 {
				return *tx, nil
			}
		}

		if len(block.PrevHash) == 0 {
			break
		}
	}

	return Transaction{}, errors.New("transaction does not exist")
}

func (chain *Blockchain) SignTransaction(tx *Transaction, privKey ecdsa.PrivateKey) {
	prevTxs := make(map[string]Transaction)

	for _, in := range tx.Inputs {
		prevTx, err := chain.FindTransaction(in.ID)

		if err != nil {
			log.Panic(err)
		}

		prevTxs[hex.EncodeToString(in.ID)] = prevTx
	}

	tx.Sign(privKey, prevTxs)
}

func (chain *Blockchain) VerifyTransaction(tx *Transaction) bool {
	prevTxs := make(map[string]Transaction)

	for _, in := range tx.Inputs {
		prevTx, err := chain.FindTransaction(in.ID)

		if err != nil {
			log.Panic(err)
		}

		prevTxs[hex.EncodeToString(in.ID)] = prevTx
	}

	return tx.Verify(prevTxs)
}
