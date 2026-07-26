package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/ethereum/go-ethereum/common"
)

func main() {
	port := getEnvOrDefault("RELAYER_PORT", "9090")
	nodeAPIBase := getEnvOrDefault("TENDER_NODE_URL", "http://localhost:8080")
	nodeAPIKey := getEnvOrDefault("TENDER_API_KEY", "")
	ethereumRPC := getEnvOrDefault("TENDER_ETHEREUM_RPC_URL", "")
	ethereumContract := getEnvOrDefault("TENDER_ETHEREUM_LOCK_CONTRACT", "")
	ethereumTopic := getEnvOrDefault("TENDER_ETHEREUM_LOCK_TOPIC", "")

	privKey, pubKey, err := loadOrGenerateKey("relayer-key.json")
	if err != nil {
		log.Fatalf("failed to load/generate relayer key: %v", err)
	}

	tokenAdapter := NewTokenAdapter(nodeAPIBase, nodeAPIKey)

	r := &relayer{
		privKey:         privKey,
		pubKey:          pubKey,
		pubHex:          hex.EncodeToString(pubKey),
		processedHashes: make(map[string]struct{}),
		tokenAdapter:    tokenAdapter,
		nodeAPIBase:     nodeAPIBase,
		nodeAPIKey:      nodeAPIKey,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/relay/lock", r.handleLock)
	mux.HandleFunc("/relay/mint", r.handleMint)
	mux.HandleFunc("/relay/status", r.handleStatus)
	mux.HandleFunc("/relayer/health", r.handleHealth)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if ethereumRPC != "" && ethereumContract != "" && ethereumTopic != "" {
		listener, err := newEthereumListener(ethereumRPC, r)
		if err != nil {
			log.Fatalf("failed to create ethereum listener: %v", err)
		}
		listener.contract = common.HexToAddress(ethereumContract)
		listener.topic = common.HexToHash(ethereumTopic)
		go func() {
			_ = listener.start(ctx)
		}()
		go func() {
			<-ctx.Done()
			listener.stop()
		}()
	} else {
		log.Println("ethereum relay disabled; set TENDER_ETHEREUM_RPC_URL, TENDER_ETHEREUM_LOCK_CONTRACT, TENDER_ETHEREUM_LOCK_TOPIC to enable")
	}

	log.Printf("bridge relayer listening on :%s", port)
	log.Printf("relayer public key: %s", r.pubHex)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}

func loadOrGenerateKey(path string) (ed25519.PrivateKey, ed25519.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err == nil {
		var kf keyFile
		if err := json.Unmarshal(data, &kf); err != nil {
			return nil, nil, err
		}
		priv, err := hex.DecodeString(kf.PrivateKey)
		if err != nil {
			return nil, nil, err
		}
		pub, err := hex.DecodeString(kf.PublicKey)
		if err != nil {
			return nil, nil, err
		}
		if len(priv) != ed25519.PrivateKeySize {
			return nil, nil, fmt.Errorf("invalid private key size")
		}
		return ed25519.PrivateKey(priv), ed25519.PublicKey(pub), nil
	}

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	kf := keyFile{
		PublicKey:  hex.EncodeToString(pub),
		PrivateKey: hex.EncodeToString(priv),
	}
	out, _ := json.MarshalIndent(kf, "", "  ")
	if err := os.WriteFile(path, out, 0o600); err != nil {
		return nil, nil, err
	}
	log.Printf("saved new relayer key to %s", path)
	return priv, pub, nil
}

func getEnvOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
