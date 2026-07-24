package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	blockchain "ai_block_chain_go/blockchain"
)

func TestTokenMetadataEndpoint(t *testing.T) {
	cfg := ServerConfigFromEnv()
	cfg.EnableAuth = false
	_ = blockchain.NewBlockchain(blockchain.ProofOfStake, t.TempDir(), "tdr-testnet-1")

	mux := http.NewServeMux()
	mux.HandleFunc("/api/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"currency_name":      blockchain.CurrencyName,
			"currency_symbol":    blockchain.CurrencySymbol(),
			"currency_subunit":   blockchain.CurrencySubunit,
			"subunit_factor":     blockchain.HogohogoPerTender,
			"base_fee":           blockchain.BaseFee,
			"base_fee_formatted": blockchain.FormatAmount(blockchain.BaseFee),
			"burn_rate_percent":  blockchain.BurnRatePercent,
			"reward_rate_percent": blockchain.RewardRatePercent,
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/token", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["currency_name"] != "TENDER" {
		t.Fatalf("expected currency_name TENDER, got %v", resp["currency_name"])
	}
	if resp["currency_subunit"] != "HOGOHOGO" {
		t.Fatalf("expected currency_subunit HOGOHOGO, got %v", resp["currency_subunit"])
	}
	if resp["subunit_factor"] != float64(10000000) {
		t.Fatalf("expected subunit_factor 10000000, got %v", resp["subunit_factor"])
	}
	if resp["base_fee_formatted"] == nil {
		t.Fatalf("expected base_fee_formatted to be present")
	}
}
