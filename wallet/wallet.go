package wallet

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
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
	PrivateKey *ecdsa.PrivateKey
	PublicKey  []byte
}

func NewKeyPair() (*ecdsa.PrivateKey, []byte) {
	curve := elliptic.P256()

	private, err := ecdsa.GenerateKey(curve, rand.Reader)

	if err != nil {
		log.Panic(err)
	}

	publicKey := append(private.PublicKey.X.Bytes(), private.PublicKey.Y.Bytes()...)

	return private, publicKey
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

func Checksum(publicKeyHash []byte) []byte {
	hashed := sha256.Sum256(publicKeyHash)
	hashed = sha256.Sum256(hashed[:])

	return hashed[:checksumLength]
}

func (w *Wallet) Address() []byte {
	pubHash := PublicKeyHash(w.PublicKey)
	versionedHash := append([]byte{version}, pubHash...)
	checksum := Checksum(versionedHash)

	fullHash := append(versionedHash, checksum...)
	address := Base58Encode(fullHash)

	fmt.Printf("pub key: %x\n", w.PublicKey)
	fmt.Printf("pub hash: %x\n", pubHash)
	fmt.Printf("address: %s\n", address)

	return address
}

func ValidateAddress(address string) bool {
	pubKeyHash := Base58Decode([]byte(address))
	actualChecksum := pubKeyHash[len(pubKeyHash)-checksumLength:]
	version := pubKeyHash[0]
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-checksumLength]
	targetChecksum := Checksum(append([]byte{version}, pubKeyHash...))

	return bytes.Compare(actualChecksum, targetChecksum) == 0
}
