// Direct facilitator diagnostic: fetch 402 requirements from the local debug
// service, sign an EIP-3009 payment with the local payer key, then call the
// facilitator /verify and /settle endpoints raw and print full JSON responses
// (the gin middleware masks settlement failure details).
package main

import (
	"bytes"
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
	exactclient "github.com/coinbase/x402/go/mechanisms/evm/exact/client"
	evmsigner "github.com/coinbase/x402/go/signers/evm"
	"github.com/coinbase/x402/go/types"
)

const (
	target      = "http://localhost:8081/v1/latest?base=USD"
	facilitator = "https://facilitator.pieverse.io/v2"
	keyFile     = "/tmp/fxpayer/payer.key"
)

func die(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", err)
		os.Exit(1)
	}
}

func post(endpoint string, body []byte) {
	req, err := http.NewRequest("POST", facilitator+endpoint, bytes.NewReader(body))
	die(err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 60 * time.Second}).Do(req)
	die(err)
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var pretty bytes.Buffer
	_ = json.Indent(&pretty, raw, "", "  ")
	fmt.Printf("=== POST %s -> HTTP %d ===\n%s\n\n", endpoint, resp.StatusCode, pretty.String())
}

func main() {
	keyHex := strings.TrimSpace(string(mustRead(keyFile)))
	signer, err := evmsigner.NewClientSignerFromPrivateKey(keyHex)
	die(err)
	fmt.Println("PAYER=" + signer.Address())

	resp, err := http.Get(target)
	die(err)
	raw402, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusPaymentRequired {
		die(fmt.Errorf("expected 402 from target, got %d: %s", resp.StatusCode, raw402))
	}
	prHeader := resp.Header.Get("PAYMENT-REQUIRED")
	if prHeader == "" {
		die(fmt.Errorf("missing PAYMENT-REQUIRED header"))
	}
	decoded, err := base64.StdEncoding.DecodeString(prHeader)
	die(err)
	var required types.PaymentRequired
	die(json.Unmarshal(decoded, &required))
	if len(required.Accepts) == 0 {
		die(fmt.Errorf("no accepts in 402"))
	}
	requirements := required.Accepts[0]
	reqJSON, _ := json.Marshal(requirements)
	fmt.Printf("=== REQUIREMENTS ===\n%s\n\n", reqJSON)

	client := x402.Newx402Client().Register(x402.Network(requirements.Network),
		exactclient.NewExactEvmScheme(signer, &exactclient.ExactEvmSchemeConfig{}))

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	payload, err := client.CreatePaymentPayload(ctx, requirements, required.Resource, nil)
	die(err)
	payloadJSON, _ := json.Marshal(payload)
	fmt.Printf("=== PAYMENT PAYLOAD ===\n%s\n\n", payloadJSON)

	body, err := json.Marshal(map[string]interface{}{
		"x402Version":         2,
		"paymentPayload":      payload,
		"paymentRequirements": requirements,
	})
	die(err)

	post("/verify", body)
	post("/settle", body)
}

func mustRead(p string) []byte {
	b, err := os.ReadFile(p)
	die(err)
	return b
}
