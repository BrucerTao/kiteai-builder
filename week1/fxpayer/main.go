// Local x402 payer harness (throwaway, lives in /tmp — NOT part of the service).
//
// Why this exists: `kpass agent:session execute` cannot target localhost. The
// passport backend reconstructs and fetches the merchant URL server-side (the
// CLI only sends url+method+hashes+session_id), and the agent EVM key is
// custodial on the backend — agent.json holds no private key. So a real paid
// call against http://localhost:8080 must use a payer we control.
//
// This program is a minimal x402 V2 payer built on the same SDK the merchant
// uses. It signs an EIP-3009 transferWithAuthorization for pieUSD with a local
// key; the merchant's facilitator submits it on-chain (gasless for the payer).
package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	x402 "github.com/coinbase/x402/go"
	x402http "github.com/coinbase/x402/go/http"
	exactclient "github.com/coinbase/x402/go/mechanisms/evm/exact/client"
	evmsigner "github.com/coinbase/x402/go/signers/evm"
	"github.com/ethereum/go-ethereum/crypto"
)

const keyFile = "/tmp/fxpayer/payer.key"

func die(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", err)
		os.Exit(1)
	}
}

func gen() {
	pk, err := crypto.GenerateKey()
	die(err)
	keyHex := "0x" + strings.ToUpper(fmt.Sprintf("%x", crypto.FromECDSA(pk)))
	addr := crypto.PubkeyToAddress(pk.PublicKey)
	die(os.WriteFile(keyFile, []byte(keyHex+"\n"), 0o600))
	fmt.Println("PAYER_ADDRESS=" + addr.Hex())
	fmt.Println("KEYFILE=" + keyFile)
}

func pay(rawURL, network string) {
	b, err := os.ReadFile(keyFile)
	die(err)
	keyHex := strings.TrimSpace(string(b))

	signer, err := evmsigner.NewClientSignerFromPrivateKey(keyHex)
	die(err)
	fmt.Println("PAYER_ADDRESS=" + signer.Address())

	scheme := exactclient.NewExactEvmScheme(signer, &exactclient.ExactEvmSchemeConfig{})
	client := x402.Newx402Client().Register(x402.Network(network), scheme)
	hc := x402http.Newx402HTTPClient(client)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	t0 := time.Now()
	resp, err := hc.GetWithPayment(ctx, rawURL)
	die(err)
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	fmt.Println("TARGET=" + rawURL)
	fmt.Println("STATUS=" + fmt.Sprintf("%d", resp.StatusCode))
	fmt.Println("ELAPSED=" + time.Since(t0).Round(time.Millisecond).String())
	fmt.Println("BODY=" + string(body))

	for k, v := range resp.Header {
		fmt.Printf("HDR %s: %s\n", k, strings.Join(v, ","))
	}

	if pr := resp.Header.Get("PAYMENT-RESPONSE"); pr != "" {
		dec, derr := base64.StdEncoding.DecodeString(pr)
		if derr == nil {
			var sr map[string]any
			if json.Unmarshal(dec, &sr) == nil {
				pretty, _ := json.MarshalIndent(sr, "", "  ")
				fmt.Println("SETTLE_RESPONSE=" + string(pretty))
			} else {
				fmt.Println("SETTLE_RESPONSE_RAW=" + string(dec))
			}
		}
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: fxpayer gen | pay <url> [network]")
		os.Exit(2)
	}
	switch os.Args[1] {
	case "gen":
		gen()
	case "pay":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "pay needs a url")
			os.Exit(2)
		}
		network := "eip155:2368"
		if len(os.Args) >= 4 {
			network = os.Args[3]
		}
		_ = http.DefaultClient
		pay(os.Args[2], network)
	default:
		fmt.Fprintln(os.Stderr, "unknown mode:", os.Args[1])
		os.Exit(2)
	}
}
