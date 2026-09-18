// Kite x402 service template (Go + Gin).
//
// Wraps an existing HTTP API behind x402 payments settled on the Kite chain.
// Requests to /v1/* return HTTP 402 until the caller attaches a valid
// PAYMENT-SIGNATURE; the payment is verified by the facilitator, the request
// is proxied to UPSTREAM_URL, and the payment is settled only if the upstream
// answered with a non-error status.
package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	x402http "github.com/coinbase/x402/go/http"
	ginmw "github.com/coinbase/x402/go/http/gin"
	evm "github.com/coinbase/x402/go/mechanisms/evm/exact/server"
	"github.com/gin-gonic/gin"
)

func env(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func main() {
	payTo := env("PAY_TO", "")
	if payTo == "" {
		log.Fatal("PAY_TO is required: the Kite wallet address that receives payments")
	}
	chain, err := KiteChainByName(env("KITE_NETWORK", "mainnet"))
	if err != nil {
		log.Fatal(err)
	}
	upstream, err := url.Parse(env("UPSTREAM_URL", ""))
	if err != nil || upstream.Host == "" {
		log.Fatal("UPSTREAM_URL is required, e.g. https://api.example.com")
	}
	price := env("PRICE_USD", "$0.001")
	if !strings.HasPrefix(price, "$") {
		price = "$" + price
	}

	// 1. Which routes cost money, and how much. Everything under /v1/ is paid;
	//    /healthz stays free so load balancers can probe the service.
	routes := x402http.RoutesConfig{
		"/v1/*": {
			Accepts: x402http.PaymentOptions{{
				Scheme:            "exact",
				PayTo:             payTo,
				Price:             price,
				Network:           chain.Network,
				MaxTimeoutSeconds: 60,
			}},
			Description: env("SERVICE_DESCRIPTION", "Paid API wrapped for the Kite network"),
			MimeType:    "application/json",
		},
	}

	// 2. Facilitator + Kite pricing.
	facilitator := x402http.NewHTTPFacilitatorClient(&x402http.FacilitatorConfig{
		URL:     env("FACILITATOR_URL", FacilitatorURL),
		Timeout: 30 * time.Second,
	})
	scheme := evm.NewExactEvmScheme().RegisterMoneyParser(chain.MoneyParser())

	// 3. Reverse proxy to the API you are wrapping. The upstream credential is
	//    injected here and never reaches the caller.
	proxy := httputil.NewSingleHostReverseProxy(upstream)
	upstreamAuthHeader := env("UPSTREAM_AUTH_HEADER", "Authorization")
	upstreamAuthValue := env("UPSTREAM_AUTH_VALUE", "")
	baseDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		baseDirector(req)
		req.Host = upstream.Host
		req.Header.Del("PAYMENT-SIGNATURE")
		if upstreamAuthValue != "" {
			req.Header.Set(upstreamAuthHeader, upstreamAuthValue)
		}
	}

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true, "network": chain.Network, "asset": chain.AssetSymbol, "price": price})
	})

	paid := r.Group("/v1")
	paid.Use(ginmw.X402Payment(ginmw.Config{
		Routes:                 routes,
		Facilitator:            facilitator,
		Schemes:                []ginmw.SchemeConfig{{Network: chain.Network, Server: scheme}},
		SyncFacilitatorOnStart: true,
		Timeout:                60 * time.Second,
	}))
	paid.Any("/*path", func(c *gin.Context) {
		// Strip the /v1 prefix so /v1/forecast reaches the upstream as /forecast.
		c.Request.URL.Path = strings.TrimPrefix(c.Request.URL.Path, "/v1")
		proxy.ServeHTTP(c.Writer, c.Request)
	})

	addr := ":" + env("PORT", "8080")
	log.Printf("kite x402 service on %s -> %s (network %s, %s per call to %s)", addr, upstream, chain.Network, price, payTo)
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
	}
}
