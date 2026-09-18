// Local x402 payer harness v2 (throwaway, /tmp only).
//
// Same as fxpayer but signs the EIP-3009 authorization with a 2-hour
// validAfter backdate. The Kite testnet head currently lags wall clock by
// ~30 min, so the SDK's default now-600s window reverts on-chain with
// AuthorizationNotYetValid. The facilitator's /verify does not care how old
// validAfter is, only that the signature is valid and validBefore is future.
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

	evm "github.com/coinbase/x402/go/mechanisms/evm"
	evmsigner "github.com/coinbase/x402/go/signers/evm"
	"github.com/coinbase/x402/go/types"
	"math/big"
)

const keyFile = "/tmp/fxpayer/payer.key"

const (
	validAfterBackdate = 2 * time.Hour
	validBeforeSkew    = time.Hour
)

func die(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", err)
		os.Exit(1)
	}
}

func signPaymentPayload(signer evm.ClientEvmSigner, requirements types.PaymentRequirements, resource *types.ResourceInfo) types.PaymentPayload {
	now := time.Now().Unix()
	nonce, err := evm.CreateNonce()
	die(err)
	authorization := evm.ExactEIP3009Authorization{
		From:        signer.Address(),
		To:          requirements.PayTo,
		Value:       requirements.Amount,
		ValidAfter:  fmt.Sprintf("%d", now-int64(validAfterBackdate.Seconds())),
		ValidBefore: fmt.Sprintf("%d", now+int64(validBeforeSkew.Seconds())),
		Nonce:       nonce,
	}

	tokenName, _ := requirements.Extra["name"].(string)
	tokenVersion, _ := requirements.Extra["version"].(string)
	if tokenName == "" || tokenVersion == "" {
		die(fmt.Errorf("requirements.extra missing name/version"))
	}
	chainID, err := evm.GetEvmChainId(string(requirements.Network))
	die(err)
	asset := evm.NormalizeAddress(requirements.Asset)

	domain := evm.TypedDataDomain{Name: tokenName, Version: tokenVersion, ChainID: chainID, VerifyingContract: asset}
	tdTypes := map[string][]evm.TypedDataField{
		"EIP712Domain": {
			{Name: "name", Type: "string"},
			{Name: "version", Type: "string"},
			{Name: "chainId", Type: "uint256"},
			{Name: "verifyingContract", Type: "address"},
		},
		"TransferWithAuthorization": {
			{Name: "from", Type: "address"},
			{Name: "to", Type: "address"},
			{Name: "value", Type: "uint256"},
			{Name: "validAfter", Type: "uint256"},
			{Name: "validBefore", Type: "uint256"},
			{Name: "nonce", Type: "bytes32"},
		},
	}
	value := mustBig(authorization.Value)
	_ = value
	message := map[string]interface{}{
		"from":        authorization.From,
		"to":          authorization.To,
		"value":       mustBig(authorization.Value),
		"validAfter":  mustBig(authorization.ValidAfter),
		"validBefore": mustBig(authorization.ValidBefore),
		"nonce":       mustBytes32(authorization.Nonce),
	}
	sig, err := signer.SignTypedData(context.Background(), domain, tdTypes, "TransferWithAuthorization", message)
	die(err)

	payload := &evm.ExactEIP3009Payload{
		Signature:     evm.BytesToHex(sig),
		Authorization: authorization,
	}
	return types.PaymentPayload{
		X402Version: 2,
		Payload:     payload.ToMap(),
		Accepted:    requirements,
		Resource:    resource,
	}
}

func mustBig(s string) *big.Int {
	v, ok := new(big.Int).SetString(s, 10)
	if !ok {
		die(fmt.Errorf("invalid uint256 %q", s))
	}
	return v
}

func mustBytes32(hexStr string) []byte {
	b, err := evm.HexToBytes(hexStr)
	die(err)
	return b
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: fxpayer2 <url>")
		os.Exit(2)
	}
	target := os.Args[1]

	keyHex := strings.TrimSpace(string(mustRead(keyFile)))
	signer, err := evmsigner.NewClientSignerFromPrivateKey(keyHex)
	die(err)
	fmt.Println("PAYER=" + signer.Address())

	// 1. Unpaid GET -> 402 with requirements
	resp, err := http.Get(target)
	die(err)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusPaymentRequired {
		fmt.Printf("STATUS=%d\nBODY=%s\n", resp.StatusCode, body)
		die(fmt.Errorf("expected 402"))
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
		die(fmt.Errorf("no accepts"))
	}
	requirements := required.Accepts[0]

	// 2. Sign with a wide validity window
	payload := signPaymentPayload(signer, requirements, required.Resource)
	payloadJSON, err := json.Marshal(payload)
	die(err)
	fmt.Printf("PAYLOAD=%s\n", payloadJSON)

	// 3. Retry with payment header
	req, err := http.NewRequest("GET", target, nil)
	die(err)
	req.Header.Set("PAYMENT-SIGNATURE", base64.StdEncoding.EncodeToString(payloadJSON))
	t0 := time.Now()
	resp2, err := (&http.Client{Timeout: 120 * time.Second}).Do(req)
	die(err)
	defer resp2.Body.Close()
	body2, _ := io.ReadAll(resp2.Body)

	fmt.Println("TARGET=" + target)
	fmt.Println("STATUS=" + fmt.Sprintf("%d", resp2.StatusCode))
	fmt.Println("ELAPSED=" + time.Since(t0).Round(time.Millisecond).String())
	fmt.Println("BODY=" + string(body2))
	for k, v := range resp2.Header {
		fmt.Printf("HDR %s: %s\n", k, strings.Join(v, ","))
	}
	if pr := resp2.Header.Get("PAYMENT-RESPONSE"); pr != "" {
		dec, derr := base64.StdEncoding.DecodeString(pr)
		if derr == nil {
			var pretty bytes.Buffer
			_ = json.Indent(&pretty, dec, "", "  ")
			fmt.Println("SETTLE_RESPONSE=" + pretty.String())
		} else {
			fmt.Println("SETTLE_RESPONSE_RAW=" + pr)
		}
	}
}

func mustRead(p string) []byte {
	b, err := os.ReadFile(p)
	die(err)
	return b
}
