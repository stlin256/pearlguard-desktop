package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseHashrateNOPools(t *testing.T) {
	body := `<li><div class='w3-row estimate' onclick="window.open('https://example.com/pool', '_blank')"><div class='model'>ExamplePool</div><div class='estimatesDescription'>Fee</div><div class='estimates'>1.00%</div><div class='estimatesDescription'>Payout</div><div class='estimates'>PPLNS</div><div class='estimatesDescription'>Hashrate 4.8%</div><div class='estimates'>1.5 Eh/s</div></div></li>`
	latency := int64(12)
	observations := parseHashrateNOPools(Pool{ID: "catalog", Name: "Catalog", Adapter: "hashrate-no-prl-pools"}, body, &latency, "2026-01-01T00:00:00Z")
	if len(observations) != 1 {
		t.Fatalf("expected one observation, got %d", len(observations))
	}
	obs := observations[0]
	if obs.PoolName != "ExamplePool" || obs.Fee != "1.00%" || obs.Payout != "PPLNS" || obs.Share != "4.8%" || obs.PoolHashrate != "1.5 Eh/s" {
		t.Fatalf("unexpected parsed observation: %#v", obs)
	}
}

func TestNormalizePublicPoolAdapters(t *testing.T) {
	alpha := normalizePoolObservation(Pool{ID: "alpha", Name: "Alpha", Adapter: "alphapool-prl", CoinSymbol: "PRL", RewardMode: "PPLNS"}, map[string]interface{}{
		"feePercent": float64(1),
		"chain":      map[string]interface{}{"height": float64(10)},
		"coins":      []interface{}{map[string]interface{}{"network_hash": "30 EH/s", "ttfLabel": "1h"}},
		"pool":       map[string]interface{}{"miners24h": float64(2), "hashrate": "1 EH/s"},
	}, true, nil, "2026-01-01T00:00:00Z", "")
	if alpha.Fee != "1%" || alpha.Payout != "PPLNS" || alpha.BlockHeight != float64(10) {
		t.Fatalf("unexpected alpha observation: %#v", alpha)
	}

	akoya := normalizePoolObservation(Pool{ID: "akoya", Name: "Akoya", Adapter: "akoyapool-prl", CoinSymbol: "PRL", RewardMode: "PPLTS"}, map[string]interface{}{
		"data": map[string]interface{}{"connected_miners": float64(3), "total_hashrate": float64(220000000000000000), "network_hashrate": float64(31000000000000000000), "current_block_height": float64(20), "pool_fee_percent": float64(2)},
	}, true, nil, "2026-01-01T00:00:00Z", "")
	if akoya.Miners != float64(3) || akoya.Fee != "2%" || akoya.PoolHashrate != "220 PH/s" {
		t.Fatalf("unexpected akoya observation: %#v", akoya)
	}

	nushy := normalizePoolObservation(Pool{ID: "nushy", Name: "Nushy", Adapter: "nushypool-v2", CoinSymbol: "PRL", RewardMode: "FPPS"}, map[string]interface{}{
		"result": map[string]interface{}{"pools": []interface{}{map[string]interface{}{"ticker": "PRL", "payoutSystem": "FPPS", "poolFee": "1.0", "activeMiners": float64(8), "hashrate": map[string]interface{}{"total": float64(2259030183537978)}, "networkBlock": "0x129c7"}}},
	}, true, nil, "2026-01-01T00:00:00Z", "")
	if nushy.Payout != "FPPS" || nushy.BlockHeight != int64(76231) || nushy.Fee != "1%" {
		t.Fatalf("unexpected nushy observation: %#v", nushy)
	}
}

func TestEncryptedJSONRoundTripAndLegacyRead(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	original := map[string]interface{}{"version": float64(1), "label": "local encrypted data"}
	if err := writeJSONFile(path, original); err != nil {
		t.Fatalf("write encrypted JSON: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read encrypted file: %v", err)
	}
	if !json.Valid(raw) || !jsonContains(raw, "encrypted") || jsonContains(raw, "local encrypted data") {
		t.Fatalf("file was not written as an encrypted envelope: %s", string(raw))
	}
	var decoded map[string]interface{}
	if err := readJSONFile(path, &decoded); err != nil {
		t.Fatalf("read encrypted JSON: %v", err)
	}
	if decoded["label"] != original["label"] {
		t.Fatalf("unexpected decrypted value: %#v", decoded)
	}
	legacy := filepath.Join(dir, "legacy.json")
	if err := os.WriteFile(legacy, []byte(`{"version":1,"walletLabel":"Legacy Wallet"}`), 0o600); err != nil {
		t.Fatalf("write legacy JSON: %v", err)
	}
	var config WalletConfig
	if err := readJSONFile(legacy, &config); err != nil {
		t.Fatalf("read legacy JSON: %v", err)
	}
	if config.WalletLabel != "Legacy Wallet" {
		t.Fatalf("unexpected legacy config: %#v", config)
	}
}

func jsonContains(raw []byte, needle string) bool {
	return string(raw) != "" && regexpContains(string(raw), needle)
}

func regexpContains(text string, needle string) bool {
	return len(needle) == 0 || strings.Contains(text, needle)
}

func TestV100SPVChainFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode RPC request: %v", err)
		}
		switch request["method"] {
		case "getblockchaininfo":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"result": nil, "error": map[string]interface{}{"code": -1, "message": "Chain RPC is inactive"}})
		case "getblockcount":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"result": 76255, "error": nil})
		case "getbalance":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"result": 1.25, "error": nil})
		default:
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"result": nil, "error": map[string]interface{}{"code": -32601, "message": "missing"}})
		}
	}))
	defer server.Close()
	config := WalletConfig{Version: 1, Network: "mainnet", RPCURL: server.URL}
	info, spvMode, err := readChainStatus(config)
	if err != nil {
		t.Fatalf("SPV fallback failed: %v", err)
	}
	if !spvMode || info["blocks"] != float64(76255) || info["headers"] != float64(76255) {
		t.Fatalf("unexpected fallback info: spv=%v info=%#v", spvMode, info)
	}
	if _, err := readWalletBalanceRaw(config); err != nil {
		t.Fatalf("wallet balance fallback failed: %v", err)
	}
}

func TestSelectQuoteNodeFlexibleTicker(t *testing.T) {
	data := map[string]interface{}{"data": []interface{}{map[string]interface{}{"market": "BTCUSDT", "last": "1"}, map[string]interface{}{"market": "PRLUSDT", "last": "0.0123", "bid": "0.012"}}}
	node := selectQuoteNode(data, "prlusdt")
	last := quoteNumber(node, "last")
	if last == nil || *last != 0.0123 {
		t.Fatalf("unexpected quote node: %#v", node)
	}
}
