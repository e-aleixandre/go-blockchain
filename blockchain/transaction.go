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
	"github.com/e-aleixandre/go-blockchain/wallet"
	"log"
	"math/big"
	"strings"
)

type Transaction struct {
	ID      []byte
	Inputs  []TxInput
	Outputs []TxOutput
}

func (tx *Transaction) Serialize() []byte {
	var encoded bytes.Buffer

	enc := gob.NewEncoder(&encoded)
	err := enc.Encode(tx)

	if err != nil {
		log.Panic(err)
	}

	return encoded.Bytes()

}

func (tx *Transaction) Hash() []byte {
	var hash [32]byte

	txCopy := *tx
	txCopy.ID = []byte{}

	hash = sha256.Sum256(txCopy.Serialize())

	return hash[:]
}

func (tx *Transaction) SetId() {
	var encoded bytes.Buffer
	var hash [32]byte

	encode := gob.NewEncoder(&encoded)
	err := encode.Encode(tx)

	if err != nil {
		log.Panic(err)
	}

	hash = sha256.Sum256(encoded.Bytes())
	tx.ID = hash[:]
}

func NewTransaction(from, to string, amount int, chain *Blockchain) *Transaction {
	var inputs []TxInput
	var outputs []TxOutput

	wallets, err := wallet.CreateWallets()

	if err != nil {
		log.Panic(err)
	}

	w := wallets.GetWallet(from)
	pubKeyhash := wallet.PublicKeyHash(w.PublicKey)

	acc, validOutputs := chain.FindSpendableOutputs(pubKeyhash, amount)

	if acc < amount {
		log.Panic("Error: not enough balance")
	}

	for txid, outs := range validOutputs {
		txID, err := hex.DecodeString(txid)

		if err != nil {
			log.Panic(err)
		}

		for _, out := range outs {
			input := TxInput{txID, out, nil, w.PublicKey}
			inputs = append(inputs, input)
		}
	}
	outputs = append(outputs, *NewTxOutput(amount, to))

	if acc > amount {
		outputs = append(outputs, TxOutput{acc - amount, pubKeyhash})
	}

	tx := Transaction{nil, inputs, outputs}
	tx.ID = tx.Hash()
	chain.SignTransaction(&tx, *w.PrivateKey)

	return &tx
}

func CoinbaseTx(to, data string) *Transaction {
	if data == "" {
		data = fmt.Sprintf("Coins to %s", to)
	}

	txin := TxInput{[]byte{}, -1, nil, []byte(data)}
	txout := NewTxOutput(100, to)

	tx := Transaction{[]byte{}, []TxInput{txin}, []TxOutput{*txout}}
	tx.SetId()

	return &tx
}

func (tx *Transaction) IsCoinbase() bool {
	return len(tx.Inputs) == 1 && len(tx.Inputs[0].ID) == 0 && tx.Inputs[0].Out == -1
}

func (tx *Transaction) TrimmedCopy() Transaction {
	var inputs []TxInput
	var outputs []TxOutput

	for _, in := range tx.Inputs {
		inputs = append(inputs, TxInput{in.ID, in.Out, nil, nil})
	}

	for _, out := range tx.Outputs {
		outputs = append(outputs, TxOutput{out.Value, out.PubKeyHash})
	}

	txCopy := Transaction{tx.ID, inputs, outputs}

	return txCopy
}

func (tx *Transaction) Sign(privKey ecdsa.PrivateKey, prevTXs map[string]Transaction) {
	if tx.IsCoinbase() {
		return
	}

	for _, in := range tx.Inputs {
		if prevTXs[hex.EncodeToString(in.ID)].ID == nil {
			log.Panic("Error: Previous transaction does not exist")
		}

		txCopy := tx.TrimmedCopy()

		for inId, in := range txCopy.Inputs {
			prevTx := prevTXs[hex.EncodeToString(in.ID)]
			txCopy.Inputs[inId].Signature = nil
			txCopy.Inputs[inId].PubKey = prevTx.Outputs[in.Out].PubKeyHash
			txCopy.ID = txCopy.Hash()
			txCopy.Inputs[inId].PubKey = nil

			r, s, err := ecdsa.Sign(rand.Reader, &privKey, txCopy.ID)

			if err != nil {
				log.Panic(err)
			}

			signature := append(r.Bytes(), s.Bytes()...)

			tx.Inputs[inId].Signature = signature
		}

	}
}

func (tx *Transaction) Verify(prevTxs map[string]Transaction) bool {
	if tx.IsCoinbase() {
		return true
	}

	for _, in := range tx.Inputs {
		if prevTxs[hex.EncodeToString(in.ID)].ID == nil {
			log.Panic("Error: previous transaction does not exist")
		}
	}

	txCopy := tx.TrimmedCopy()
	curve := elliptic.P256()

	for inId, in := range tx.Inputs {
		prevTx := prevTxs[hex.EncodeToString(in.ID)]
		txCopy.Inputs[inId].Signature = nil
		txCopy.Inputs[inId].PubKey = prevTx.Outputs[in.Out].PubKeyHash
		txCopy.ID = txCopy.Hash()
		txCopy.Inputs[inId].PubKey = nil

		r := big.Int{}
		s := big.Int{}
		signatureLength := len(in.Signature)
		r.SetBytes(in.Signature[:signatureLength/2])
		s.SetBytes(in.Signature[signatureLength/2:])

		x := big.Int{}
		y := big.Int{}
		keyLength := len(in.PubKey)
		x.SetBytes(in.PubKey[:keyLength/2])
		y.SetBytes(in.PubKey[keyLength/2:])

		rawPubKey := ecdsa.PublicKey{curve, &x, &y}

		if !ecdsa.Verify(&rawPubKey, txCopy.ID, &r, &s) {
			return false
		}
	}

	return true
}

func (tx *Transaction) String() string {
	var lines []string

	lines = append(lines, fmt.Sprintf("--- Transaction %x: ", tx.ID))
	for i, input := range tx.Inputs {
		lines = append(lines, fmt.Sprintf("\tInput %d:", i))
		lines = append(lines, fmt.Sprintf("\t\tTxID: %x", input.ID))
		lines = append(lines, fmt.Sprintf("\t\tOut: %d", input.Out))
		lines = append(lines, fmt.Sprintf("\t\tSignature: %x", input.Signature))
		lines = append(lines, fmt.Sprintf("\t\tPubKey: %x", input.PubKey))
	}

	for i, output := range tx.Outputs {
		lines = append(lines, fmt.Sprintf("\tOutput %d:", i))
		lines = append(lines, fmt.Sprintf("\t\tValue: %d", output.Value))
		lines = append(lines, fmt.Sprintf("\t\tScript: %x", output.PubKeyHash))
	}

	return strings.Join(lines, "\n")
}
