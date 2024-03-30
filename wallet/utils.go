package wallet

import "github.com/btcsuite/btcutil/base58"

func Base58Encode(input []byte) []byte {
	encode := base58.Encode(input)

	return []byte(encode)
}

func Base58Decode(input []byte) []byte {
	decode := base58.Decode(string(input))

	return decode
}
