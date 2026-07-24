package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type workPayload struct {
	Difficulty uint32 `json:"difficulty"`
	Height     uint64 `json:"height"`
	Timestamp  int64  `json:"timestamp"`
}

type submitPayload struct {
	Nonce    uint64 `json:"nonce"`
	Miner    string `json:"miner"`
	Solution string `json:"solution"`
}

func main() {
	http.HandleFunc("/pool/work", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		_ = json.NewEncoder(w).Encode(workPayload{Difficulty: 16, Height: 1, Timestamp: time.Now().Unix()})
	})
	http.HandleFunc("/pool/submit", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var p submitPayload
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "accepted", "miner": p.Miner, "nonce": p.Nonce})
	})
	fmt.Println("pool listening on :9090")
	log.Fatal(http.ListenAndServe(":9090", nil))
}
