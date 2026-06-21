package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"embed"
	"encoding/base64"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	stdRuntime "runtime"
	"strconv"
	"strings"
	"time"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const repoURL = "https://github.com/stlin256/pearlguard-desktop"

//go:embed data/*.json src/locales/*.json
var backendFiles embed.FS

type App struct {
	ctx     context.Context
	version string
}

type LocalPaths struct {
	UserData             string `json:"userData"`
	RuntimeDir           string `json:"runtimeDir"`
	WalletConfig         string `json:"walletConfig"`
	WalletConfigPortable string `json:"walletConfigPortable"`
	PoolsConfig          string `json:"poolsConfig"`
	PoolsConfigPortable  string `json:"poolsConfigPortable"`
	StateFile            string `json:"stateFile"`
	ExampleWalletConfig  string `json:"exampleWalletConfig"`
	ExamplePoolsConfig   string `json:"examplePoolsConfig"`
}

type WalletConfig struct {
	Version            int     `json:"version,omitempty"`
	WalletLabel        string  `json:"walletLabel,omitempty"`
	Network            string  `json:"network,omitempty"`
	RPCURL             string  `json:"rpcUrl,omitempty"`
	RPCHost            string  `json:"rpcHost,omitempty"`
	RPCPort            int     `json:"rpcPort,omitempty"`
	RPCUsername        string  `json:"rpcUsername,omitempty"`
	RPCPassword        string  `json:"rpcPassword,omitempty"`
	ReservePRL         float64 `json:"reservePRL,omitempty"`
	ThresholdPRL       float64 `json:"thresholdPRL,omitempty"`
	DestinationAddress string  `json:"destinationAddress,omitempty"`
	RefreshSeconds     int     `json:"refreshSeconds,omitempty"`
	PoolSyncSeconds    int     `json:"poolSyncSeconds,omitempty"`
	AutoRefresh        bool    `json:"autoRefresh,omitempty"`
	UILanguage         string  `json:"uiLanguage,omitempty"`
	ReadOnly           bool    `json:"readOnly"`
}

type Wallet struct {
	Label          string   `json:"label"`
	Network        string   `json:"network"`
	Configured     bool     `json:"configured"`
	Connected      bool     `json:"connected"`
	Synced         bool     `json:"synced"`
	BlockHeight    *float64 `json:"blockHeight"`
	BestPeerHeight *float64 `json:"bestPeerHeight"`
	BalancePRL     float64  `json:"balancePRL"`
	ReservePRL     float64  `json:"reservePRL"`
	ThresholdPRL   float64  `json:"thresholdPRL"`
	Mode           string   `json:"mode"`
}

type Snapshot struct {
	Timestamp    string   `json:"timestamp"`
	BalancePRL   float64  `json:"balancePRL"`
	ReservePRL   float64  `json:"reservePRL"`
	ThresholdPRL float64  `json:"thresholdPRL"`
	BlockHeight  *float64 `json:"blockHeight,omitempty"`
}
type AddressEvent struct {
	Timestamp       string  `json:"timestamp"`
	AddressLabel    string  `json:"addressLabel"`
	Direction       string  `json:"direction"`
	AmountPRL       float64 `json:"amountPRL"`
	BalanceAfterPRL float64 `json:"balanceAfterPRL"`
	TxID            string  `json:"txid"`
	Source          string  `json:"source"`
}
type AuditEvent struct {
	Timestamp string `json:"timestamp"`
	Scope     string `json:"scope"`
	Event     string `json:"event"`
	Status    string `json:"status"`
	Severity  string `json:"severity"`
	Message   string `json:"message"`
}
type RuntimeState struct {
	Source        string         `json:"source"`
	Wallet        Wallet         `json:"wallet"`
	Snapshots     []Snapshot     `json:"snapshots"`
	AddressEvents []AddressEvent `json:"addressEvents"`
	AuditEvents   []AuditEvent   `json:"auditEvents"`
}

type PoolMapping struct {
	Miners          string `json:"miners,omitempty"`
	PoolHashrate    string `json:"poolHashrate,omitempty"`
	NetworkHashrate string `json:"networkHashrate,omitempty"`
	BlockHeight     string `json:"blockHeight,omitempty"`
	EstimatedReward string `json:"estimatedReward,omitempty"`
}
type Pool struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	Adapter    string      `json:"adapter"`
	Enabled    bool        `json:"enabled"`
	Endpoint   string      `json:"endpoint"`
	CoinSymbol string      `json:"coinSymbol"`
	Homepage   string      `json:"homepage,omitempty"`
	Notes      string      `json:"notes,omitempty"`
	Mapping    PoolMapping `json:"mapping,omitempty"`
}
type PoolConfig struct {
	Version     int    `json:"version"`
	Privacy     string `json:"privacy,omitempty"`
	PollSeconds int    `json:"pollSeconds"`
	Pools       []Pool `json:"pools"`
	ConfigPath  string `json:"configPath,omitempty"`
}
type PoolObservation struct {
	Timestamp       string      `json:"timestamp"`
	PoolID          string      `json:"poolId"`
	PoolName        string      `json:"poolName"`
	Adapter         string      `json:"adapter"`
	Reachable       bool        `json:"reachable"`
	Miners          interface{} `json:"miners"`
	PoolHashrate    interface{} `json:"poolHashrate"`
	NetworkHashrate interface{} `json:"networkHashrate"`
	BlockHeight     interface{} `json:"blockHeight"`
	EstimatedReward interface{} `json:"estimatedReward"`
	LatencyMs       *int64      `json:"latencyMs"`
	Message         string      `json:"message"`
}
type PoolSyncResult struct {
	Timestamp    string              `json:"timestamp"`
	Mode         string              `json:"mode"`
	Observations []PoolObservation   `json:"observations"`
	Errors       []map[string]string `json:"errors"`
	ConfigPath   string              `json:"configPath,omitempty"`
}
type RPCResponse struct {
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func NewApp(version string) *App           { return &App{version: version} }
func (a *App) startup(ctx context.Context) { a.ctx = ctx }
func nowISO() string                       { return time.Now().UTC().Format(time.RFC3339Nano) }

func userDataDir() string {
	base, err := os.UserConfigDir()
	if err != nil || base == "" {
		base = "."
	}
	return filepath.Join(base, "PearlGuard Desktop")
}
func runtimeDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exe)
}
func getLocalPaths() LocalPaths {
	data := userDataDir()
	run := runtimeDir()
	return LocalPaths{UserData: data, RuntimeDir: run, WalletConfig: filepath.Join(data, "wallet.config.json"), WalletConfigPortable: filepath.Join(run, "wallet.config.json"), PoolsConfig: filepath.Join(data, "pools.local.json"), PoolsConfigPortable: filepath.Join(run, "pools.local.json"), StateFile: filepath.Join(data, "pearlguard-state.json"), ExampleWalletConfig: "data/wallet.config.example.json", ExamplePoolsConfig: "data/pools.example.json"}
}
func readEmbeddedJSON(name string, out interface{}) error {
	raw, err := backendFiles.ReadFile(name)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, out)
}
func readJSONFile(path string, out interface{}) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, out)
}
func writeJSONFile(path string, value interface{}) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(path, raw, 0o600)
}
func firstExisting(paths ...string) string {
	for _, path := range paths {
		if path == "" {
			continue
		}
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

func defaultConfig() WalletConfig {
	return WalletConfig{Version: 1, WalletLabel: "Local Pearl Wallet", Network: "mainnet", RPCHost: "127.0.0.1", RPCPort: 8335, ReservePRL: 0.02, ThresholdPRL: 1.1, RefreshSeconds: 30, PoolSyncSeconds: 120, AutoRefresh: false, ReadOnly: true}
}
func mergeConfig(config WalletConfig) WalletConfig {
	base := defaultConfig()
	if config.Version != 0 {
		base.Version = config.Version
	}
	if config.WalletLabel != "" {
		base.WalletLabel = config.WalletLabel
	}
	if config.Network != "" {
		base.Network = config.Network
	}
	if config.RPCURL != "" {
		base.RPCURL = config.RPCURL
	}
	if config.RPCHost != "" {
		base.RPCHost = config.RPCHost
	}
	if config.RPCPort != 0 {
		base.RPCPort = config.RPCPort
	}
	if config.RPCUsername != "" {
		base.RPCUsername = config.RPCUsername
	}
	if config.RPCPassword != "" {
		base.RPCPassword = config.RPCPassword
	}
	if config.ReservePRL != 0 {
		base.ReservePRL = config.ReservePRL
	}
	if config.ThresholdPRL != 0 {
		base.ThresholdPRL = config.ThresholdPRL
	}
	if config.DestinationAddress != "" {
		base.DestinationAddress = config.DestinationAddress
	}
	if config.RefreshSeconds != 0 {
		base.RefreshSeconds = config.RefreshSeconds
	}
	if config.PoolSyncSeconds != 0 {
		base.PoolSyncSeconds = config.PoolSyncSeconds
	}
	if config.UILanguage != "" {
		base.UILanguage = config.UILanguage
	}
	base.AutoRefresh = config.AutoRefresh
	base.ReadOnly = true
	return base
}
func readWalletConfig() (WalletConfig, string, bool) {
	paths := getLocalPaths()
	path := firstExisting(paths.WalletConfig, paths.WalletConfigPortable)
	if path == "" {
		return defaultConfig(), "", false
	}
	var config WalletConfig
	if err := readJSONFile(path, &config); err != nil {
		return defaultConfig(), path, false
	}
	return mergeConfig(config), path, true
}
func defaultWallet(config WalletConfig, configured bool) Wallet {
	return Wallet{Label: config.WalletLabel, Network: config.Network, Configured: configured, Connected: false, Synced: false, BalancePRL: 0, ReservePRL: config.ReservePRL, ThresholdPRL: config.ThresholdPRL, Mode: "read-only"}
}
func emptyLocalState() RuntimeState {
	config, _, configured := readWalletConfig()
	source := "local"
	if configured {
		source = "wallet.config.json"
	}
	return RuntimeState{Source: source, Wallet: defaultWallet(config, configured), Snapshots: []Snapshot{}, AddressEvents: []AddressEvent{}, AuditEvents: []AuditEvent{}}
}
func readRuntimeState() RuntimeState {
	paths := getLocalPaths()
	var state RuntimeState
	if err := readJSONFile(paths.StateFile, &state); err == nil && state.Wallet.Label != "" {
		config, _, configured := readWalletConfig()
		state.Wallet.Configured = configured
		state.Wallet.ReservePRL = config.ReservePRL
		state.Wallet.ThresholdPRL = config.ThresholdPRL
		if config.WalletLabel != "" {
			state.Wallet.Label = config.WalletLabel
		}
		if config.Network != "" {
			state.Wallet.Network = config.Network
		}
		return state
	}
	return emptyLocalState()
}
func writeRuntimeState(state RuntimeState) error {
	return writeJSONFile(getLocalPaths().StateFile, state)
}

func readPoolConfig() PoolConfig {
	paths := getLocalPaths()
	path := firstExisting(paths.PoolsConfig, paths.PoolsConfigPortable)
	if path != "" {
		var config PoolConfig
		if err := readJSONFile(path, &config); err == nil {
			config.ConfigPath = path
			return config
		}
	}
	var embedded PoolConfig
	if err := readEmbeddedJSON("data/pools.example.json", &embedded); err != nil {
		return PoolConfig{Version: 1, PollSeconds: 120, Pools: []Pool{}}
	}
	return embedded
}

func sanitizeSettings(input WalletConfig) WalletConfig {
	settings := mergeConfig(input)
	settings.Version = 1
	settings.ReadOnly = true
	if settings.RefreshSeconds < 10 {
		settings.RefreshSeconds = 10
	}
	if settings.PoolSyncSeconds < 30 {
		settings.PoolSyncSeconds = 30
	}
	if settings.ReservePRL < 0 {
		settings.ReservePRL = 0
	}
	if settings.ThresholdPRL < 0.00000001 {
		settings.ThresholdPRL = 1.1
	}
	settings.RPCURL = strings.TrimSpace(settings.RPCURL)
	settings.RPCHost = strings.TrimSpace(settings.RPCHost)
	settings.RPCUsername = strings.TrimSpace(settings.RPCUsername)
	settings.DestinationAddress = strings.TrimSpace(settings.DestinationAddress)
	return settings
}

func (a *App) GetBootstrap() map[string]interface{} {
	config, _, _ := readWalletConfig()
	return map[string]interface{}{"name": "PearlGuard Desktop", "version": a.version, "repoUrl": repoURL, "platform": stdRuntime.GOOS, "locale": detectedLocale(), "mode": "local", "transferDisabled": true, "paths": getLocalPaths(), "settings": config, "state": readRuntimeState(), "poolConfig": readPoolConfig()}
}
func detectedLocale() string {
	raw := strings.ToLower(os.Getenv("PEARLGUARD_LOCALE"))
	if raw == "" {
		raw = strings.ToLower(os.Getenv("LANG"))
	}
	switch {
	case strings.HasPrefix(raw, "ar"):
		return "ar"
	case strings.HasPrefix(raw, "zh"):
		return "zh-CN"
	case strings.HasPrefix(raw, "fr"):
		return "fr"
	case strings.HasPrefix(raw, "ru"):
		return "ru"
	case strings.HasPrefix(raw, "es"):
		return "es"
	default:
		return "en"
	}
}
func (a *App) GetMessages(locale string) map[string]string {
	allowed := map[string]bool{"en": true, "ar": true, "zh-CN": true, "fr": true, "ru": true, "es": true}
	if !allowed[locale] {
		locale = "en"
	}
	var messages map[string]string
	if err := readEmbeddedJSON("src/locales/"+locale+".json", &messages); err != nil {
		_ = readEmbeddedJSON("src/locales/en.json", &messages)
	}
	return messages
}
func coerceJSON(input interface{}, out interface{}) error {
	raw, err := json.Marshal(input)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, out)
}

func (a *App) SaveSettings(input map[string]interface{}) (map[string]interface{}, error) {
	var incoming WalletConfig
	if err := coerceJSON(input, &incoming); err != nil {
		return nil, err
	}
	settings := sanitizeSettings(incoming)
	paths := getLocalPaths()
	if err := writeJSONFile(paths.WalletConfig, settings); err != nil {
		return nil, err
	}
	state := readRuntimeState()
	state.Wallet.Label = settings.WalletLabel
	state.Wallet.Network = settings.Network
	state.Wallet.Configured = true
	state.Wallet.ReservePRL = settings.ReservePRL
	state.Wallet.ThresholdPRL = settings.ThresholdPRL
	state.AuditEvents = append(state.AuditEvents, AuditEvent{Timestamp: nowISO(), Scope: "settings", Event: "save", Status: "ok", Severity: "info", Message: "Local settings were saved."})
	_ = writeRuntimeState(state)
	return map[string]interface{}{"ok": true, "settings": settings, "state": state, "paths": paths}, nil
}

func rpcEndpoint(config WalletConfig) string {
	if strings.TrimSpace(config.RPCURL) != "" {
		return strings.TrimSpace(config.RPCURL)
	}
	host := config.RPCHost
	if host == "" {
		host = "127.0.0.1"
	}
	port := config.RPCPort
	if port == 0 {
		port = 8335
	}
	return fmt.Sprintf("http://%s:%d", host, port)
}
func jsonRPC(config WalletConfig, method string, params []interface{}) (json.RawMessage, error) {
	payload := map[string]interface{}{"jsonrpc": "1.0", "id": "pearlguard", "method": method, "params": params}
	raw, _ := json.Marshal(payload)
	request, err := http.NewRequest(http.MethodPost, rpcEndpoint(config), bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	request.Header.Set("content-type", "application/json")
	request.Header.Set("user-agent", "PearlGuard-Desktop/0.3.0")
	if config.RPCUsername != "" || config.RPCPassword != "" {
		token := base64.StdEncoding.EncodeToString([]byte(config.RPCUsername + ":" + config.RPCPassword))
		request.Header.Set("authorization", "Basic "+token)
	}
	client := &http.Client{Timeout: 6500 * time.Millisecond}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	var rpc RPCResponse
	if err := json.Unmarshal(body, &rpc); err != nil {
		return nil, fmt.Errorf("RPC %s returned invalid JSON", method)
	}
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return nil, fmt.Errorf("RPC %s failed with HTTP %d", method, response.StatusCode)
	}
	if rpc.Error != nil {
		return nil, errors.New(rpc.Error.Message)
	}
	return rpc.Result, nil
}

func numberFrom(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case json.Number:
		n, err := v.Float64()
		return n, err == nil
	case string:
		if v == "" {
			return 0, false
		}
		n, err := strconv.ParseFloat(v, 64)
		return n, err == nil
	default:
		return 0, false
	}
}
func floatPointer(value interface{}) *float64 {
	n, ok := numberFrom(value)
	if !ok {
		return nil
	}
	return &n
}

func (a *App) TestRPCConnection(input map[string]interface{}) map[string]interface{} {
	var settings WalletConfig
	if len(input) == 0 {
		settings, _, _ = readWalletConfig()
	} else if err := coerceJSON(input, &settings); err != nil {
		return map[string]interface{}{"ok": false, "message": err.Error()}
	}
	settings = sanitizeSettings(settings)
	result, err := jsonRPC(settings, "getblockchaininfo", []interface{}{})
	if err != nil {
		return map[string]interface{}{"ok": false, "message": err.Error()}
	}
	var info map[string]interface{}
	_ = json.Unmarshal(result, &info)
	return map[string]interface{}{"ok": true, "message": "RPC connection succeeded.", "chain": info["chain"], "blocks": info["blocks"], "headers": info["headers"]}
}

func (a *App) ReadWalletStatus() map[string]interface{} {
	config, configPath, configured := readWalletConfig()
	state := readRuntimeState()
	if !configured {
		paths := getLocalPaths()
		return map[string]interface{}{"ok": false, "configured": false, "configPath": paths.WalletConfig, "message": "Wallet settings have not been saved yet."}
	}
	blockchainRaw, blockErr := jsonRPC(config, "getblockchaininfo", []interface{}{})
	balanceRaw, balanceErr := jsonRPC(config, "getbalance", []interface{}{})
	if blockErr != nil && balanceErr != nil {
		return map[string]interface{}{"ok": false, "configured": true, "configPath": configPath, "message": blockErr.Error()}
	}
	var block map[string]interface{}
	if blockErr == nil {
		_ = json.Unmarshal(blockchainRaw, &block)
	}
	var balanceValue interface{}
	if balanceErr == nil {
		_ = json.Unmarshal(balanceRaw, &balanceValue)
	}
	balance, ok := numberFrom(balanceValue)
	if !ok {
		balance = state.Wallet.BalancePRL
	}
	blocks := floatPointer(block["blocks"])
	headers := floatPointer(block["headers"])
	synced := false
	if blocks != nil && headers != nil {
		synced = *blocks >= *headers
	}
	network := config.Network
	if chain, ok := block["chain"].(string); ok && chain != "" {
		network = chain
	}
	wallet := state.Wallet
	wallet.Label = config.WalletLabel
	wallet.Network = network
	wallet.Configured = true
	wallet.Connected = true
	wallet.Synced = synced
	wallet.BlockHeight = blocks
	wallet.BestPeerHeight = headers
	wallet.BalancePRL = balance
	wallet.ReservePRL = config.ReservePRL
	wallet.ThresholdPRL = config.ThresholdPRL
	wallet.Mode = "read-only"
	snapshot := Snapshot{Timestamp: nowISO(), BalancePRL: wallet.BalancePRL, ReservePRL: wallet.ReservePRL, ThresholdPRL: wallet.ThresholdPRL, BlockHeight: wallet.BlockHeight}
	state.Source = "wallet-rpc"
	state.Wallet = wallet
	state.Snapshots = append(state.Snapshots, snapshot)
	state.AuditEvents = append(state.AuditEvents, AuditEvent{Timestamp: snapshot.Timestamp, Scope: "wallet", Event: "read-status", Status: "ok", Severity: "info", Message: "Wallet status refreshed."})
	_ = writeRuntimeState(state)
	return map[string]interface{}{"ok": true, "state": state, "configPath": configPath}
}

func (a *App) DryRunSweepCheck(input map[string]interface{}) map[string]interface{} {
	balance, _ := numberFrom(input["balancePRL"])
	reserve, ok := numberFrom(input["reservePRL"])
	if !ok {
		reserve = 0
	}
	threshold, ok := numberFrom(input["thresholdPRL"])
	if !ok {
		threshold = 1.1
	}
	sweepable := balance - reserve
	if sweepable < 0 {
		sweepable = 0
	}
	sweepable, _ = strconv.ParseFloat(fmt.Sprintf("%.8f", sweepable), 64)
	reached := sweepable > threshold
	decision := "hold"
	message := "Threshold is not reached."
	if reached {
		decision = "ready"
		message = "Threshold is reached. Review local policy before using a broadcast-capable tool."
	}
	return map[string]interface{}{"mode": "read-only-check", "transferDisabled": true, "transferRequests": 0, "balancePRL": balance, "reservePRL": reserve, "thresholdPRL": threshold, "sweepablePRL": sweepable, "thresholdReached": reached, "decision": decision, "message": message}
}

func pick(data interface{}, dotted string) interface{} {
	if dotted == "" {
		return nil
	}
	current := data
	for _, key := range strings.Split(dotted, ".") {
		node, ok := current.(map[string]interface{})
		if !ok {
			return nil
		}
		current = node[key]
	}
	return current
}
func firstDefined(values ...interface{}) interface{} {
	for _, value := range values {
		if value == nil {
			continue
		}
		if text, ok := value.(string); ok && text == "" {
			continue
		}
		return value
	}
	return nil
}
func chooseMessage(primary string, fallback string) string {
	if strings.TrimSpace(primary) != "" {
		return primary
	}
	return fallback
}
func normalizeHashrate(value interface{}) interface{} {
	if value == nil {
		return nil
	}
	if text, ok := value.(string); ok {
		return text
	}
	n, ok := numberFrom(value)
	if !ok {
		return fmt.Sprintf("%v", value)
	}
	units := []string{"H/s", "KH/s", "MH/s", "GH/s", "TH/s", "PH/s"}
	index := 0
	for n >= 1000 && index < len(units)-1 {
		n /= 1000
		index++
	}
	precision := 2
	if n >= 100 {
		precision = 0
	} else if n >= 10 {
		precision = 1
	}
	return fmt.Sprintf("%.*f %s", precision, n, units[index])
}
func coinNode(data map[string]interface{}, symbol string) map[string]interface{} {
	keys := []string{symbol, strings.ToUpper(symbol), strings.ToLower(symbol), "PRL", "pearl"}
	for _, key := range keys {
		if key == "" {
			continue
		}
		if node, ok := data[key].(map[string]interface{}); ok {
			return node
		}
	}
	return data
}

func normalizePoolObservation(pool Pool, data map[string]interface{}, reachable bool, latency *int64, timestamp string, message string) PoolObservation {
	values := map[string]interface{}{}
	if data != nil {
		switch pool.Adapter {
		case "miningcore-pool":
			stats, _ := firstDefined(data["poolStats"], pick(data, "pool.poolStats"), data["stats"], data).(map[string]interface{})
			network, _ := firstDefined(data["networkStats"], pick(data, "pool.networkStats"), data["network"], map[string]interface{}{}).(map[string]interface{})
			values["miners"] = firstDefined(stats["connectedMiners"], stats["miners"], stats["workers"])
			values["poolHashrate"] = normalizeHashrate(firstDefined(stats["poolHashRate"], stats["hashrate"], stats["poolHashrate"]))
			values["networkHashrate"] = normalizeHashrate(firstDefined(network["networkHashRate"], network["hashrate"], network["networkHashrate"]))
			values["blockHeight"] = firstDefined(network["blockHeight"], stats["blockHeight"], data["blockHeight"])
			values["estimatedReward"] = firstDefined(stats["estimatedReward"], stats["reward"], data["estimatedReward"])
			message = chooseMessage(message, "Miningcore-compatible pool response normalized.")
		case "nomp-pool":
			stats, _ := firstDefined(data["pool_stats"], data["poolStats"], data["stats"], data).(map[string]interface{})
			values["miners"] = firstDefined(stats["workers"], stats["miners"], data["workers"])
			values["poolHashrate"] = normalizeHashrate(firstDefined(stats["hashrate"], stats["poolHashrate"], data["hashrate"]))
			values["networkHashrate"] = normalizeHashrate(firstDefined(stats["networkHashrate"], data["networkHashrate"], data["nethash"]))
			values["blockHeight"] = firstDefined(stats["height"], data["height"], data["blockHeight"])
			values["estimatedReward"] = firstDefined(stats["reward"], data["reward"], data["estimate"])
			message = chooseMessage(message, "NOMP-compatible pool response normalized.")
		case "generic-json":
			values["miners"] = firstDefined(pick(data, pool.Mapping.Miners), data["miners"], data["workers"])
			values["poolHashrate"] = normalizeHashrate(firstDefined(pick(data, pool.Mapping.PoolHashrate), data["poolHashrate"], data["hashrate"]))
			values["networkHashrate"] = normalizeHashrate(firstDefined(pick(data, pool.Mapping.NetworkHashrate), data["networkHashrate"], data["nethash"]))
			values["blockHeight"] = firstDefined(pick(data, pool.Mapping.BlockHeight), data["blockHeight"], data["height"])
			values["estimatedReward"] = firstDefined(pick(data, pool.Mapping.EstimatedReward), data["estimatedReward"], data["reward"])
			message = chooseMessage(message, "Generic JSON pool response normalized.")
		default:
			node := coinNode(data, pool.CoinSymbol)
			values["miners"] = firstDefined(node["workers"], node["miners"], node["workerCount"])
			values["poolHashrate"] = normalizeHashrate(firstDefined(node["hashrate"], node["hashrate_shared"], node["pool_hashrate"]))
			values["networkHashrate"] = normalizeHashrate(firstDefined(node["network_hashrate"], node["networkHashrate"], node["nethash"]))
			values["blockHeight"] = firstDefined(node["height"], node["lastblock"], node["blockHeight"])
			values["estimatedReward"] = firstDefined(node["estimate_current"], node["estimate"], node["reward"])
			message = chooseMessage(message, "Yiimp-compatible pool response normalized.")
		}
	}
	if message == "" {
		message = "No live data available."
	}
	return PoolObservation{Timestamp: timestamp, PoolID: pool.ID, PoolName: pool.Name, Adapter: pool.Adapter, Reachable: reachable && data != nil, Miners: values["miners"], PoolHashrate: values["poolHashrate"], NetworkHashrate: values["networkHashrate"], BlockHeight: values["blockHeight"], EstimatedReward: values["estimatedReward"], LatencyMs: latency, Message: message}
}

func fetchJSON(endpoint string) (map[string]interface{}, int64, error) {
	started := time.Now()
	request, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, 0, err
	}
	request.Header.Set("accept", "application/json")
	request.Header.Set("user-agent", "PearlGuard-Desktop/0.3.0")
	client := &http.Client{Timeout: 6500 * time.Millisecond}
	response, err := client.Do(request)
	if err != nil {
		return nil, 0, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, 0, err
	}
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return nil, 0, fmt.Errorf("HTTP %d", response.StatusCode)
	}
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, 0, fmt.Errorf("endpoint did not return JSON")
	}
	return data, time.Since(started).Milliseconds(), nil
}

func (a *App) SyncPools(options map[string]interface{}) PoolSyncResult {
	timestamp := nowISO()
	config := readPoolConfig()
	observations := []PoolObservation{}
	errorsList := []map[string]string{}
	for _, pool := range config.Pools {
		if !pool.Enabled || strings.TrimSpace(pool.Endpoint) == "" {
			observations = append(observations, normalizePoolObservation(pool, nil, false, nil, timestamp, "Pool is saved but disabled or missing an endpoint."))
			continue
		}
		data, latency, err := fetchJSON(pool.Endpoint)
		if err != nil {
			msg := err.Error()
			errorsList = append(errorsList, map[string]string{"poolId": pool.ID, "message": msg})
			observations = append(observations, normalizePoolObservation(pool, nil, false, nil, timestamp, msg))
			continue
		}
		observations = append(observations, normalizePoolObservation(pool, data, true, &latency, timestamp, ""))
	}
	return PoolSyncResult{Timestamp: timestamp, Mode: "local", Observations: observations, Errors: errorsList, ConfigPath: config.ConfigPath}
}

func (a *App) SavePoolConfig(input map[string]interface{}) (map[string]interface{}, error) {
	var config PoolConfig
	if err := coerceJSON(input, &config); err != nil {
		return nil, err
	}
	if config.Version == 0 {
		config.Version = 1
	}
	if config.PollSeconds < 30 {
		config.PollSeconds = 120
	}
	for index := range config.Pools {
		if strings.TrimSpace(config.Pools[index].ID) == "" {
			seed := sha256.Sum256([]byte(config.Pools[index].Name + time.Now().String()))
			config.Pools[index].ID = "pool-" + hex.EncodeToString(seed[:])[:8]
		}
		if strings.TrimSpace(config.Pools[index].Adapter) == "" {
			config.Pools[index].Adapter = "generic-json"
		}
		if strings.TrimSpace(config.Pools[index].CoinSymbol) == "" {
			config.Pools[index].CoinSymbol = "PRL"
		}
	}
	paths := getLocalPaths()
	config.Privacy = "Local pool endpoints are stored outside the repository."
	config.ConfigPath = ""
	if err := writeJSONFile(paths.PoolsConfig, config); err != nil {
		return nil, err
	}
	config.ConfigPath = paths.PoolsConfig
	return map[string]interface{}{"ok": true, "poolConfig": config}, nil
}

func (a *App) ImportAuditCsv() map[string]interface{} {
	if a.ctx == nil {
		return map[string]interface{}{"canceled": true}
	}
	filePath, err := wailsRuntime.OpenFileDialog(a.ctx, wailsRuntime.OpenDialogOptions{Title: "Import PearlGuard audit CSV", Filters: []wailsRuntime.FileFilter{{DisplayName: "CSV files (*.csv)", Pattern: "*.csv"}}})
	if err != nil || filePath == "" {
		return map[string]interface{}{"canceled": true}
	}
	file, err := os.Open(filePath)
	if err != nil {
		return map[string]interface{}{"canceled": true, "message": err.Error()}
	}
	defer file.Close()
	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	rows, err := reader.ReadAll()
	if err != nil || len(rows) == 0 {
		return map[string]interface{}{"canceled": true, "message": "CSV file could not be read."}
	}
	headers := rows[0]
	records := []map[string]string{}
	for _, row := range rows[1:] {
		record := map[string]string{}
		for index, header := range headers {
			if index < len(row) {
				record[header] = row[index]
			}
		}
		records = append(records, record)
	}
	state := stateFromAuditRows(records, filePath)
	_ = writeRuntimeState(state)
	return map[string]interface{}{"canceled": false, "importedRows": len(records), "state": state}
}

func stateFromAuditRows(rows []map[string]string, filePath string) RuntimeState {
	base := emptyLocalState()
	snapshots := []Snapshot{}
	addressEvents := []AddressEvent{}
	auditEvents := []AuditEvent{}
	for _, row := range rows {
		timestamp := firstText(row["timestamp"], row["time"], nowISO())
		balance, hasBalance := parseFloatText(row["balancePRL"])
		amount, hasAmount := parseFloatText(row["amountPRL"])
		event := firstText(row["event"], "audit")
		status := firstText(row["status"], "observed")
		auditEvents = append(auditEvents, AuditEvent{Timestamp: timestamp, Scope: event, Event: event, Status: status, Severity: "info", Message: firstText(row["message"], event+" "+status)})
		if hasBalance {
			reserve, ok := parseFloatText(row["reservePRL"])
			if !ok {
				reserve = base.Wallet.ReservePRL
			}
			threshold, ok := parseFloatText(row["minAmountPRL"])
			if !ok {
				threshold = base.Wallet.ThresholdPRL
			}
			block := parseOptionalFloat(row["blockHeight"])
			snapshots = append(snapshots, Snapshot{Timestamp: timestamp, BalancePRL: balance, ReservePRL: reserve, ThresholdPRL: threshold, BlockHeight: block})
		}
		if hasAmount && amount != 0 {
			direction := "in"
			if amount < 0 || status == "sent" {
				direction = "out"
				amount = abs(amount)
			}
			addressEvents = append(addressEvents, AddressEvent{Timestamp: timestamp, AddressLabel: firstText(row["addressLabel"], "Local observation"), Direction: direction, AmountPRL: amount, BalanceAfterPRL: balance, TxID: row["txid"], Source: filepath.Base(filePath)})
		}
	}
	if len(snapshots) > 0 {
		last := snapshots[len(snapshots)-1]
		base.Wallet.BalancePRL = last.BalancePRL
		base.Wallet.ReservePRL = last.ReservePRL
		base.Wallet.ThresholdPRL = last.ThresholdPRL
		base.Wallet.BlockHeight = last.BlockHeight
		base.Wallet.Synced = last.BlockHeight != nil
	}
	base.Source = filepath.Base(filePath)
	base.Snapshots = snapshots
	base.AddressEvents = addressEvents
	base.AuditEvents = auditEvents
	return base
}

func firstText(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
func parseFloatText(value string) (float64, bool) {
	if strings.TrimSpace(value) == "" {
		return 0, false
	}
	n, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	return n, err == nil
}
func parseOptionalFloat(value string) *float64 {
	n, ok := parseFloatText(value)
	if !ok {
		return nil
	}
	return &n
}
func abs(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}
