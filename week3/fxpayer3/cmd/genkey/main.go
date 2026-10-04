// genkey writes a fresh secp256k1 key (bare hex, no 0x) for the fxpayer3
// harness and prints the derived EVM address.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/crypto"
)

func main() {
	key, err := crypto.GenerateKey()
	if err != nil {
		panic(err)
	}
	hexKey := fmt.Sprintf("%x", crypto.FromECDSA(key))
	addr := crypto.PubkeyToAddress(key.PublicKey).Hex()
	out := "/tmp/fxpayer3/payer.key"
	if len(os.Args) > 1 {
		out = os.Args[1]
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o700); err != nil {
		panic(err)
	}
	if err := os.WriteFile(out, []byte(hexKey+"\n"), 0o600); err != nil {
		panic(err)
	}
	fmt.Println("ADDRESS=" + addr)
	fmt.Println("KEYFILE=" + out)
}
