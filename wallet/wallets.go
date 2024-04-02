package wallet

import (
	"bytes"
	"crypto/ecdh"
	"encoding/gob"
	"fmt"
	"log"
	"os"
)

const walletsFile = "./tmp/wallets.data"

type Wallets struct {
	Wallets map[string]*Wallet
}

func CreateWallets() (*Wallets, error) {
	wallets := Wallets{}
	wallets.Wallets = make(map[string]*Wallet)

	err := wallets.LoadFile()

	return &wallets, err
}

func (ws *Wallets) GetWallet(address string) Wallet {
	return *ws.Wallets[address]
}

func (ws *Wallets) GetAllAddresses() []string {
	var addresses []string

	for address := range ws.Wallets {
		addresses = append(addresses, address)
	}

	return addresses
}

func (ws *Wallets) AddWallet() string {
	wallet := MakeWallet()
	address := fmt.Sprintf("%s", wallet.Address())

	ws.Wallets[address] = wallet

	return address
}

func (ws *Wallets) SaveFile() {
	var content bytes.Buffer

	gob.Register(ws)

	encoder := gob.NewEncoder(&content)
	err := encoder.Encode(ws)

	if err != nil {
		log.Panic(err)
	}

	err = os.WriteFile(walletsFile, content.Bytes(), 0644)

	if err != nil {
		log.Panic(err)
	}
}

func (ws *Wallets) LoadFile() error {
	if _, err := os.Stat(walletsFile); os.IsNotExist(err) {
		return err
	}

	var wallets Wallets

	fileContent, err := os.ReadFile(walletsFile)

	if err != nil {
		return err
	}

	gob.Register(ecdh.P256())
	decoder := gob.NewDecoder(bytes.NewReader(fileContent))
	err = decoder.Decode(&wallets)

	ws.Wallets = wallets.Wallets

	return nil
}
