package wallet

import (
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"golang.org/x/crypto/ripemd160"
	"log"
)

const (
	checksumLength = 4
	version        = byte(0x00)
)

type Wallet struct {
	PrivateKey *ecdh.PrivateKey
	PublicKey  []byte
}

func NewKeyPair() (*ecdh.PrivateKey, []byte) {
	curve := ecdh.P256()

	private, err := curve.GenerateKey(rand.Reader)

	if err != nil {
		log.Panic(err)
	}

	return private, private.PublicKey().Bytes()
}

func MakeWallet() *Wallet {
	private, public := NewKeyPair()
	wallet := Wallet{private, public}

	return &wallet
}

func PublicKeyHash(pubKey []byte) []byte {
	pubHash := sha256.Sum256(pubKey)

	hasher := ripemd160.New()
	hasher.Write(pubHash[:])
	publicRipMD := hasher.Sum(nil)

	return publicRipMD
}

func CheckSum(publicKeyHash []byte) []byte {
	hashed := sha256.Sum256(publicKeyHash)
	hashed = sha256.Sum256(hashed[:])

	return hashed[:checksumLength]
}

func (w *Wallet) Address() []byte {
	pubHash := PublicKeyHash(w.PublicKey)
	versionedHash := append([]byte{version}, pubHash...)
	checksum := CheckSum(versionedHash)

	fullHash := append(versionedHash, checksum...)
	address := Base58Encode(fullHash)

	fmt.Printf("pub key: %x\n", w.PublicKey)
	fmt.Printf("pub hash: %x\n", pubHash)
	fmt.Printf("address: %s\n", address)

	return address
}
