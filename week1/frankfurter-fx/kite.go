package main

import (
	"fmt"
	"math/big"

	x402 "github.com/coinbase/x402/go"
)

// KiteChain describes one Kite network the wrapper can charge on.
// Only the stablecoin that the Kite facilitator settles for that network is
// listed here: the payer signs an EIP-3009 transferWithAuthorization, so the
// asset must implement EIP-3009 and the EIP-712 domain (name/version) must
// match the token contract exactly or the facilitator rejects the signature.
type KiteChain struct {
	Network       x402.Network // CAIP-2 identifier
	RPCURL        string
	AssetAddress  string // stablecoin contract
	AssetSymbol   string
	AssetDecimals int
	EIP712Name    string
	EIP712Version string
}

var (
	// KiteMainnet: Bridged USDC (USDC.e), 6 decimals.
	KiteMainnet = KiteChain{
		Network:       "eip155:2366",
		RPCURL:        "https://rpc.gokite.ai",
		AssetAddress:  "0x7aB6f3ed87C42eF0aDb67Ed95090f8bF5240149e",
		AssetSymbol:   "USDC.e",
		AssetDecimals: 6,
		EIP712Name:    "Bridged USDC (Kite AI)",
		EIP712Version: "2",
	}

	// KiteTestnet: pieUSD test stablecoin, 18 decimals. This is what a Kite
	// Passport agent in sandbox mode pays with.
	KiteTestnet = KiteChain{
		Network:       "eip155:2368",
		RPCURL:        "https://rpc-testnet.gokite.ai",
		AssetAddress:  "0x38129cf4CE5E183eFF248F42A7D345Bb1B47621A",
		AssetSymbol:   "pieUSD",
		AssetDecimals: 18,
		EIP712Name:    "pieUSD",
		EIP712Version: "1",
	}
)

// FacilitatorURL is the x402 facilitator that verifies and settles payments on
// both Kite networks. The x402 SDK appends /verify, /settle and /supported, so
// the /v2 prefix must stay in the base URL.
const FacilitatorURL = "https://facilitator.pieverse.io/v2"

// KiteChainByName resolves the KITE_NETWORK environment value.
func KiteChainByName(name string) (KiteChain, error) {
	switch name {
	case "", "mainnet":
		return KiteMainnet, nil
	case "testnet":
		return KiteTestnet, nil
	}
	return KiteChain{}, fmt.Errorf("unknown KITE_NETWORK %q (want mainnet or testnet)", name)
}

// MoneyParser lets routes be priced as "$0.001" on Kite networks. The x402 SDK
// only knows default stablecoins for a fixed set of chains; this parser adds
// the Kite ones and pins the EIP-712 domain the facilitator expects.
func (k KiteChain) MoneyParser() x402.MoneyParser {
	return func(amount float64, network x402.Network) (*x402.AssetAmount, error) {
		if network != k.Network {
			return nil, nil // not ours; let the next parser try
		}
		if amount <= 0 {
			return nil, fmt.Errorf("price must be positive, got %v", amount)
		}
		scale := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(k.AssetDecimals)), nil))
		units, _ := new(big.Float).Mul(big.NewFloat(amount), scale).Int(nil)
		if units.Sign() <= 0 {
			return nil, fmt.Errorf("price %v is below one unit of %s", amount, k.AssetSymbol)
		}
		return &x402.AssetAmount{
			Asset:  k.AssetAddress,
			Amount: units.String(),
			Extra: map[string]interface{}{
				"name":    k.EIP712Name,
				"version": k.EIP712Version,
			},
		}, nil
	}
}
