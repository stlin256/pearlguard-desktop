package main

import "testing"

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
