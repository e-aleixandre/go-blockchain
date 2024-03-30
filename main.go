package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
	"os"
)

func main() {
	defer os.Exit(0)

	curve := elliptic.P256()

	priv, _ := ecdsa.GenerateKey(curve, rand.Reader)

	pub := priv.PublicKey

	fmt.Printf(pub.X.String())

	// cmd := cli.CommandLine{}
	// cmd.Run()
}
