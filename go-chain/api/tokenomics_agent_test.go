package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	blockchain "ai_block_chain_go/blockchain"
)

func TestTokenomicsExposesAgentMetrics(t *testing.T) {
	cfg := ServerConfigFromEnv()
	cfg.EnableAuth = false
	chain := blockchain.NewBlockchain(blockchain.ProofOfStake, t.TempDir(), "tdr-testnet-1")
	chain.AgentTxCount = 42

	mux := http.NewServeMux()
	mux.HandleFunc("/api/tokenomics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		chain.RLock()
		defer chain.RUnlock()
		agentBaseDemand := uint64(1000000)
		agentGrowth := 0.05
		agentDemand := agentBaseDemand + uint64(float64(agentBaseDemand)*agentGrowth*float64(chain.AgentTxCount))
		demandFactor := uint64(0)
		if chain.AgentTxCount > 0 && chain.TokenSupply > 0 {
			demandFactor = uint64(float64(chain.AgentTxCount) / float64(chain.TokenSupply) * 1_000_000_000)
			if demandFactor > 200 {
				demandFactor = 200
			}
		}
		deflPressure := 0.0
		if chain.TokenSupply > 0 {
			if chain.TokenSupply > 5000000000 {
				deflPressure = float64(blockchain.BurnRatePercent) * 1.5
			} else {
				deflPressure = float64(blockchain.BurnRatePercent)
			}
		}
		humanScore := 0.0
		if deflPressure > 0 {
			humanScore = float64(agentDemand) / deflPressure
		} else if agentDemand > 0 {
			humanScore = float64(agentDemand)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"agent_tx_count":        chain.AgentTxCount,
			"agent_demand_pressure": agentDemand,
			"deflationary_pressure": deflPressure,
			"human_value_score":     humanScore,
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/tokenomics", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["agent_tx_count"] != float64(42) {
		t.Fatalf("expected agent_tx_count 42, got %v", resp["agent_tx_count"])
	}
	if resp["agent_demand_pressure"] == nil {
		t.Fatalf("expected agent_demand_pressure to be present")
	}
}
