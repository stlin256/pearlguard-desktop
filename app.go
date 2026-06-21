package main

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"embed"
	"encoding/base64"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	stdRuntime "runtime"
	"strconv"
	"strings"
	"sync"
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
	Version             int     `json:"version,omitempty"`
	WalletLabel         string  `json:"walletLabel,omitempty"`
	Network             string  `json:"network,omitempty"`
	RPCURL              string  `json:"rpcUrl,omitempty"`
	RPCHost             string  `json:"rpcHost,omitempty"`
	RPCPort             int     `json:"rpcPort,omitempty"`
	RPCUsername         string  `json:"rpcUsername,omitempty"`
	RPCPassword         string  `json:"rpcPassword,omitempty"`
	WalletPassword      string  `json:"walletPassword,omitempty"`
	WalletName          string  `json:"walletName,omitempty"`
	ReservePRL          float64 `json:"reservePRL,omitempty"`
	ThresholdPRL        float64 `json:"thresholdPRL,omitempty"`
	DestinationAddress  string  `json:"destinationAddress,omitempty"`
	AutoTransferEnabled bool    `json:"autoTransferEnabled,omitempty"`
	RefreshSeconds      int     `json:"refreshSeconds,omitempty"`
	PoolSyncSeconds     int     `json:"poolSyncSeconds,omitempty"`
	AutoRefresh         bool    `json:"autoRefresh,omitempty"`
	MiningPoolEnabled   bool    `json:"miningPoolEnabled,omitempty"`
	ProxyURL            string  `json:"proxyUrl,omitempty"`
	UILanguage          string  `json:"uiLanguage,omitempty"`
	ReadOnly            bool    `json:"readOnly"`
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
	Method     string      `json:"method,omitempty"`
	Body       interface{} `json:"body,omitempty"`
	CoinSymbol string      `json:"coinSymbol"`
	RewardMode string      `json:"rewardMode,omitempty"`
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
	Homepage        string      `json:"homepage,omitempty"`
	Fee             string      `json:"fee,omitempty"`
	Payout          string      `json:"payout,omitempty"`
	Share           string      `json:"share,omitempty"`
	Message         string      `json:"message"`
}
type PoolSyncResult struct {
	Timestamp    string              `json:"timestamp"`
	Mode         string              `json:"mode"`
	Observations []PoolObservation   `json:"observations"`
	Errors       []map[string]string `json:"errors"`
	ConfigPath   string              `json:"configPath,omitempty"`
}
type MarketQuote struct {
	OK            bool     `json:"ok"`
	Source        string   `json:"source"`
	Market        string   `json:"market"`
	Last          *float64 `json:"last,omitempty"`
	Bid           *float64 `json:"bid,omitempty"`
	Ask           *float64 `json:"ask,omitempty"`
	High24h       *float64 `json:"high24h,omitempty"`
	Low24h        *float64 `json:"low24h,omitempty"`
	Volume24h     *float64 `json:"volume24h,omitempty"`
	ChangePercent *float64 `json:"changePercent,omitempty"`
	Timestamp     string   `json:"timestamp"`
	LatencyMs     *int64   `json:"latencyMs,omitempty"`
	Message       string   `json:"message"`
}

type encryptedEnvelope struct {
	Encrypted bool   `json:"encrypted"`
	Version   int    `json:"version"`
	Alg       string `json:"alg"`
	Nonce     string `json:"nonce"`
	Data      string `json:"data"`
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
func localStorageKey() []byte {
	base, _ := os.UserConfigDir()
	host, _ := os.Hostname()
	material := strings.Join([]string{"PearlGuard Desktop", "local-json-v1", stdRuntime.GOOS, base, host}, "\x00")
	sum := sha256.Sum256([]byte(material))
	key := make([]byte, len(sum))
	copy(key, sum[:])
	return key
}
func encryptJSON(raw []byte) ([]byte, error) {
	block, err := aes.NewCipher(localStorageKey())
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	envelope := encryptedEnvelope{Encrypted: true, Version: 1, Alg: "AES-256-GCM", Nonce: base64.StdEncoding.EncodeToString(nonce), Data: base64.StdEncoding.EncodeToString(gcm.Seal(nil, nonce, raw, nil))}
	sealed, err := json.MarshalIndent(envelope, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(sealed, '\n'), nil
}
func decryptJSON(env encryptedEnvelope) ([]byte, error) {
	if !env.Encrypted || env.Version != 1 || env.Alg != "AES-256-GCM" {
		return nil, fmt.Errorf("unsupported encrypted storage envelope")
	}
	nonce, err := base64.StdEncoding.DecodeString(env.Nonce)
	if err != nil {
		return nil, err
	}
	ciphertext, err := base64.StdEncoding.DecodeString(env.Data)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(localStorageKey())
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return gcm.Open(nil, nonce, ciphertext, nil)
}
func readJSONFile(path string, out interface{}) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var envelope encryptedEnvelope
	if err := json.Unmarshal(raw, &envelope); err == nil && envelope.Encrypted {
		raw, err = decryptJSON(envelope)
		if err != nil {
			return err
		}
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
	sealed, err := encryptJSON(raw)
	if err != nil {
		return err
	}
	return os.WriteFile(path, sealed, 0o600)
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
	return WalletConfig{Version: 1, WalletLabel: "Local Pearl Wallet", Network: "mainnet", RPCHost: "127.0.0.1", RPCPort: 8335, ReservePRL: 0.02, ThresholdPRL: 1.1, RefreshSeconds: 30, PoolSyncSeconds: 120, AutoRefresh: false, MiningPoolEnabled: false, ReadOnly: true}
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
	if config.WalletPassword != "" {
		base.WalletPassword = config.WalletPassword
	}
	if config.WalletName != "" {
		base.WalletName = config.WalletName
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
	base.AutoTransferEnabled = config.AutoTransferEnabled
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
	base.MiningPoolEnabled = config.MiningPoolEnabled
	if config.ProxyURL != "" {
		base.ProxyURL = config.ProxyURL
	}
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
	settings.AutoRefresh = input.AutoRefresh
	settings.AutoTransferEnabled = input.AutoTransferEnabled
	settings.MiningPoolEnabled = input.MiningPoolEnabled
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
	settings.WalletName = strings.Trim(strings.TrimSpace(settings.WalletName), "/")
	settings.ProxyURL = normalizeProxyURL(settings.ProxyURL)
	settings.Network = normalizeNetwork(settings.Network)
	settings.DestinationAddress = strings.TrimSpace(settings.DestinationAddress)
	return settings
}

func (a *App) GetBootstrap() map[string]interface{} {
	config, _, _ := readWalletConfig()
	return map[string]interface{}{"name": "PearlGuard Desktop", "version": a.version, "repoUrl": repoURL, "platform": stdRuntime.GOOS, "locale": detectedLocale(), "mode": "local", "transferDisabled": true, "paths": getLocalPaths(), "settings": config, "state": readRuntimeState(), "poolConfig": readPoolConfig()}
}
func detectedLocale() string {
	for _, raw := range []string{os.Getenv("PEARLGUARD_LOCALE"), osLocale(), os.Getenv("LANGUAGE"), os.Getenv("LC_ALL"), os.Getenv("LC_MESSAGES"), os.Getenv("LANG")} {
		if normalized := normalizeLocale(raw); normalized != "" {
			return normalized
		}
	}
	return "en"
}
func normalizeLocale(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	raw = strings.ReplaceAll(raw, "_", "-")
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
	case strings.HasPrefix(raw, "en"):
		return "en"
	default:
		return ""
	}
}

func normalizeNetwork(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "test", "testnet":
		return "testnet"
	case "regtest", "local":
		return "regtest"
	default:
		return "mainnet"
	}
}
func normalizeProxyURL(raw string) string {
	text := strings.TrimSpace(raw)
	if text == "" {
		return ""
	}
	if !strings.Contains(text, "://") {
		text = "http://" + text
	}
	parsed, err := url.Parse(text)
	if err != nil || parsed.Host == "" {
		return ""
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" && scheme != "socks5" && scheme != "socks5h" {
		return ""
	}
	return parsed.String()
}

func isLoopbackEndpoint(endpoint string) bool {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return false
	}
	host := strings.Trim(strings.ToLower(parsed.Hostname()), "[]")
	return host == "localhost" || host == "::1" || strings.HasPrefix(host, "127.") || host == "0.0.0.0"
}

func httpClientFor(config WalletConfig, timeout time.Duration, endpoint string) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if proxyText := normalizeProxyURL(config.ProxyURL); proxyText != "" && !isLoopbackEndpoint(endpoint) {
		if parsed, err := url.Parse(proxyText); err == nil {
			transport.Proxy = http.ProxyURL(parsed)
		}
	}
	return &http.Client{Timeout: timeout, Transport: transport}
}

func appHTTPClient(timeout time.Duration, endpoint string) *http.Client {
	config, _, _ := readWalletConfig()
	return httpClientFor(config, timeout, endpoint)
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

func rpcEndpoint(config WalletConfig, walletScoped bool) string {
	endpoint := strings.TrimSpace(config.RPCURL)
	if endpoint == "" {
		host := config.RPCHost
		if host == "" {
			host = "127.0.0.1"
		}
		port := config.RPCPort
		if port == 0 {
			port = 8335
		}
		endpoint = fmt.Sprintf("http://%s:%d", host, port)
	}
	if walletScoped && config.WalletName != "" && !strings.Contains(endpoint, "/wallet/") {
		endpoint = strings.TrimRight(endpoint, "/") + "/wallet/" + url.PathEscape(config.WalletName)
	}
	return endpoint
}

func readChainStatus(config WalletConfig) (map[string]interface{}, bool, error) {
	raw, err := jsonRPC(config, "getblockchaininfo", []interface{}{})
	if err == nil {
		var info map[string]interface{}
		_ = json.Unmarshal(raw, &info)
		return info, false, nil
	}
	primaryErr := err
	countRaw, countErr := jsonRPC(config, "getblockcount", []interface{}{})
	if countErr != nil {
		return nil, false, primaryErr
	}
	var countValue interface{}
	_ = json.Unmarshal(countRaw, &countValue)
	count, ok := numberFrom(countValue)
	if !ok {
		return nil, false, primaryErr
	}
	return map[string]interface{}{"chain": config.Network, "blocks": count, "headers": count}, true, nil
}

func readWalletBalanceRaw(config WalletConfig) (json.RawMessage, error) {
	var scopedErr error
	if config.WalletName != "" {
		if raw, err := jsonRPCWallet(config, "getbalance", []interface{}{}); err == nil {
			return raw, nil
		} else {
			scopedErr = err
		}
	}
	raw, err := jsonRPC(config, "getbalance", []interface{}{})
	if err == nil {
		return raw, nil
	}
	if scopedErr != nil {
		return nil, fmt.Errorf("wallet-scoped RPC failed and root wallet RPC also failed: %w", err)
	}
	return nil, err
}

func jsonRPC(config WalletConfig, method string, params []interface{}) (json.RawMessage, error) {
	return jsonRPCScoped(config, method, params, false)
}

func jsonRPCWallet(config WalletConfig, method string, params []interface{}) (json.RawMessage, error) {
	return jsonRPCScoped(config, method, params, true)
}

func jsonRPCScoped(config WalletConfig, method string, params []interface{}, walletScoped bool) (json.RawMessage, error) {
	payload := map[string]interface{}{"jsonrpc": "1.0", "id": "pearlguard", "method": method, "params": params}
	raw, _ := json.Marshal(payload)
	endpoint := rpcEndpoint(config, walletScoped)
	request, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	request.Header.Set("content-type", "application/json")
	request.Header.Set("user-agent", "PearlGuard-Desktop/0.3.0")
	if config.RPCUsername != "" || config.RPCPassword != "" {
		token := base64.StdEncoding.EncodeToString([]byte(config.RPCUsername + ":" + config.RPCPassword))
		request.Header.Set("authorization", "Basic "+token)
	}
	client := httpClientFor(config, 6500*time.Millisecond, endpoint)
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", endpoint, err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	var rpc RPCResponse
	if err := json.Unmarshal(body, &rpc); err != nil {
		return nil, fmt.Errorf("RPC %s returned invalid JSON from %s", method, endpoint)
	}
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return nil, fmt.Errorf("RPC %s failed with HTTP %d at %s", method, response.StatusCode, endpoint)
	}
	if rpc.Error != nil {
		return nil, fmt.Errorf("RPC %s failed at %s: %s", method, endpoint, rpc.Error.Message)
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
	info, spvMode, err := readChainStatus(settings)
	if err != nil {
		return map[string]interface{}{"ok": false, "chainOk": false, "walletOk": false, "message": err.Error()}
	}
	balanceRaw, walletErr := readWalletBalanceRaw(settings)
	if walletErr != nil {
		return map[string]interface{}{"ok": false, "chainOk": true, "walletOk": false, "spvMode": spvMode, "message": "Chain height is readable, but wallet balance RPC failed. " + walletErr.Error(), "chain": info["chain"], "blocks": info["blocks"], "headers": info["headers"]}
	}
	var balance interface{}
	_ = json.Unmarshal(balanceRaw, &balance)
	message := "RPC connection succeeded."
	if spvMode {
		message = "Wallet RPC connection succeeded in SPV mode."
	}
	return map[string]interface{}{"ok": true, "chainOk": true, "walletOk": true, "spvMode": spvMode, "message": message, "chain": info["chain"], "blocks": info["blocks"], "headers": info["headers"], "balance": balance}
}

func (a *App) ReadWalletStatus() map[string]interface{} {
	config, configPath, configured := readWalletConfig()
	state := readRuntimeState()
	if !configured {
		paths := getLocalPaths()
		return map[string]interface{}{"ok": false, "configured": false, "configPath": paths.WalletConfig, "message": "Wallet settings have not been saved yet."}
	}
	block, spvMode, blockErr := readChainStatus(config)
	balanceRaw, balanceErr := readWalletBalanceRaw(config)
	if blockErr != nil {
		return map[string]interface{}{"ok": false, "configured": true, "configPath": configPath, "message": blockErr.Error()}
	}
	if balanceErr != nil {
		return map[string]interface{}{"ok": false, "configured": true, "configPath": configPath, "spvMode": spvMode, "message": "Wallet RPC failed. " + balanceErr.Error()}
	}
	var balanceValue interface{}
	_ = json.Unmarshal(balanceRaw, &balanceValue)
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
	if spvMode {
		wallet.Mode = "read-only spv"
	}
	snapshot := Snapshot{Timestamp: nowISO(), BalancePRL: wallet.BalancePRL, ReservePRL: wallet.ReservePRL, ThresholdPRL: wallet.ThresholdPRL, BlockHeight: wallet.BlockHeight}
	state.Source = "wallet-rpc"
	state.Wallet = wallet
	state.Snapshots = append(state.Snapshots, snapshot)
	state.AuditEvents = append(state.AuditEvents, AuditEvent{Timestamp: snapshot.Timestamp, Scope: "wallet", Event: "read-status", Status: "ok", Severity: "info", Message: "Wallet status refreshed."})
	_ = writeRuntimeState(state)
	return map[string]interface{}{"ok": true, "state": state, "configPath": configPath, "spvMode": spvMode}
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
func asMap(value interface{}) map[string]interface{} {
	if node, ok := value.(map[string]interface{}); ok {
		return node
	}
	return map[string]interface{}{}
}
func asSlice(value interface{}) []interface{} {
	if nodes, ok := value.([]interface{}); ok {
		return nodes
	}
	return []interface{}{}
}
func formatPercent(value interface{}, multiplier float64) string {
	if value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		text = strings.TrimSpace(text)
		if text == "" {
			return ""
		}
		if strings.Contains(text, "%") {
			return text
		}
		if parsed, err := strconv.ParseFloat(text, 64); err == nil {
			value = parsed
		} else {
			return text
		}
	}
	n, ok := numberFrom(value)
	if !ok {
		return fmt.Sprintf("%v", value)
	}
	if multiplier != 0 {
		n *= multiplier
	}
	formatted := strconv.FormatFloat(n, 'f', 2, 64)
	formatted = strings.TrimRight(strings.TrimRight(formatted, "0"), ".")
	return formatted + "%"
}
func parseBlockHeight(value interface{}) interface{} {
	if text, ok := value.(string); ok && strings.HasPrefix(strings.ToLower(strings.TrimSpace(text)), "0x") {
		if parsed, err := strconv.ParseInt(strings.TrimPrefix(strings.ToLower(strings.TrimSpace(text)), "0x"), 16, 64); err == nil {
			return parsed
		}
	}
	return value
}
func selectGrandPool(data map[string]interface{}, pool Pool) map[string]interface{} {
	for _, item := range asSlice(data["pools"]) {
		node := asMap(item)
		info := asMap(node["info"])
		if !strings.EqualFold(fmt.Sprintf("%v", info["blockchain"]), "pearl") {
			continue
		}
		return node
	}
	return map[string]interface{}{}
}
func selectNushyPool(data map[string]interface{}, pool Pool) map[string]interface{} {
	result := asMap(data["result"])
	preferred := strings.ToUpper(strings.TrimSpace(pool.RewardMode))
	symbol := strings.TrimSpace(pool.CoinSymbol)
	if symbol == "" {
		symbol = "PRL"
	}
	for _, item := range asSlice(result["pools"]) {
		node := asMap(item)
		if !strings.EqualFold(fmt.Sprintf("%v", node["ticker"]), symbol) {
			continue
		}
		mode := strings.ToUpper(fmt.Sprintf("%v", node["payoutSystem"]))
		if preferred == "" || preferred == mode {
			return node
		}
	}
	return map[string]interface{}{}
}
func sumNumbers(values ...interface{}) interface{} {
	total := 0.0
	seen := false
	for _, value := range values {
		if n, ok := numberFrom(value); ok {
			total += n
			seen = true
		}
	}
	if !seen {
		return nil
	}
	return total
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
	units := []string{"H/s", "KH/s", "MH/s", "GH/s", "TH/s", "PH/s", "EH/s"}
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
	extra := PoolObservation{}
	if data != nil {
		switch pool.Adapter {
		case "alphapool-prl":
			poolStats := asMap(data["pool"])
			chainStats := asMap(data["chain"])
			coinStats := map[string]interface{}{}
			coins := asSlice(data["coins"])
			if len(coins) > 0 {
				coinStats = asMap(coins[0])
			}
			stratum := asMap(data["stratum"])
			values["miners"] = firstDefined(poolStats["miners24h"], poolStats["miners"], poolStats["workers"])
			values["poolHashrate"] = firstDefined(poolStats["hashrate"], poolStats["hashrate1h"])
			values["networkHashrate"] = firstDefined(coinStats["network_hash"], coinStats["networkHashrate"])
			values["blockHeight"] = firstDefined(chainStats["height"], coinStats["block_height"])
			values["estimatedReward"] = firstDefined(coinStats["ttfLabel"], poolStats["ttfLabel"])
			extra.Fee = formatPercent(firstDefined(data["feePercent"], "1"), 1)
			extra.Payout = firstDefined(pool.RewardMode, "PPLNS").(string)
			message = chooseMessage(message, fmt.Sprintf("Pool stats endpoint normalized. Stratum %v:%v.", firstDefined(stratum["host"], "us2.alphapool.tech"), firstDefined(stratum["standardPort"], "5566")))
		case "kryptex-pool":
			values["miners"] = firstDefined(data["miners"], data["workers"])
			values["poolHashrate"] = normalizeHashrate(data["hashrate"])
			values["networkHashrate"] = normalizeHashrate(firstDefined(data["net_hashrate"], data["network_hashrate"]))
			values["blockHeight"] = data["height"]
			values["estimatedReward"] = firstDefined(data["block_reward"], data["minpay"])
			extra.Fee = formatPercent(data["fee"], 100)
			extra.Payout = fmt.Sprintf("%v", firstDefined(pool.RewardMode, data["fee_type"], "PROP"))
			message = chooseMessage(message, "Kryptex public pool info endpoint normalized.")
		case "pearlhash-prl":
			values["miners"] = firstDefined(data["total_accounts"], data["accounts"], data["miners"])
			values["poolHashrate"] = normalizeHashrate(data["hashrate"])
			values["estimatedReward"] = firstDefined(data["total_workers"], data["workers"])
			message = chooseMessage(message, "Pearlhash public stats endpoint normalized.")
		case "luckypool-prl":
			config := asMap(data["config"])
			poolStats := asMap(data["pool"])
			network := asMap(data["network"])
			values["miners"] = firstDefined(poolStats["miners"], poolStats["workers"])
			values["poolHashrate"] = normalizeHashrate(poolStats["hashrate"])
			values["networkHashrate"] = normalizeHashrate(network["hashrate"])
			values["blockHeight"] = network["height"]
			values["estimatedReward"] = firstDefined(network["reward"], config["minPaymentThreshold"])
			extra.Fee = formatPercent(config["fee"], 1)
			extra.Payout = firstDefined(pool.RewardMode, "PPLNS").(string)
			message = chooseMessage(message, "LuckyPool public stats endpoint normalized.")
		case "akoyapool-prl":
			stats := asMap(data["data"])
			values["miners"] = firstDefined(stats["connected_miners"], stats["registered_miners"], stats["active_workers"])
			values["poolHashrate"] = normalizeHashrate(stats["total_hashrate"])
			values["networkHashrate"] = normalizeHashrate(stats["network_hashrate"])
			values["blockHeight"] = stats["current_block_height"]
			values["estimatedReward"] = firstDefined(stats["expected_block_time_seconds"], stats["total_paid24_h"])
			extra.Fee = formatPercent(stats["pool_fee_percent"], 1)
			extra.Payout = firstDefined(pool.RewardMode, "PPLTS").(string)
			message = chooseMessage(message, "Akoya public pool stats endpoint normalized.")
		case "baikalmine-engine":
			entity := asMap(data["entity"])
			hashrate := asMap(entity["hashrate"])
			values["miners"] = entity["miners"]
			values["poolHashrate"] = normalizeHashrate(firstDefined(hashrate["pool"], entity["hashrate"]))
			values["networkHashrate"] = normalizeHashrate(hashrate["network"])
			values["blockHeight"] = entity["networkHeight"]
			values["estimatedReward"] = firstDefined(entity["effort"], entity["blocktime"])
			extra.Payout = firstDefined(pool.RewardMode, "PPLNS").(string)
			message = chooseMessage(message, "BaikalMine engine stats endpoint normalized.")
		case "jetski-prl":
			values["miners"] = firstDefined(data["connected_addresses"], data["connected_workers"])
			values["poolHashrate"] = normalizeHashrate(data["pool_hashrate"])
			values["networkHashrate"] = normalizeHashrate(data["network_hashrate"])
			values["blockHeight"] = firstDefined(data["chain_height"], data["current_height"], data["mining_height"])
			values["estimatedReward"] = firstDefined(data["blocks_found"], data["block_reward_grains"])
			extra.Fee = formatPercent(data["pool_fee_bps"], 0.01)
			extra.Payout = firstDefined(pool.RewardMode, "PROP").(string)
			message = chooseMessage(message, "JETSKI public stats endpoint normalized.")
		case "mineprl-public":
			economics := asMap(data["economics"])
			poolStats := asMap(data["pool"])
			active := asMap(poolStats["active"])
			network := asMap(data["network"])
			values["miners"] = firstDefined(active["nodes"], economics["active_gpus"], economics["reported_gpus"])
			values["poolHashrate"] = normalizeHashrate(firstDefined(economics["reported_hashrate_hps"], economics["mean_hashrate_hps"], pick(active, "telemetry.reported_hashrate_hps"), pick(active, "telemetry.mean_hashrate_hps")))
			values["networkHashrate"] = normalizeHashrate(firstDefined(network["hashrate_hps"], network["network_hashrate_hps"], economics["network_hashrate_hps"]))
			values["blockHeight"] = firstDefined(network["height"], network["block_height"])
			values["estimatedReward"] = firstDefined(economics["expected_prl_per_day"], economics["expected_time_to_block_s"])
			extra.Fee = formatPercent(economics["pool_fee_rate"], 100)
			extra.Payout = firstDefined(pool.RewardMode, "PPLNS").(string)
			message = chooseMessage(message, "MinePRL public summary endpoint normalized.")
		case "himpool-miningcore", "miningcore-pool":
			poolNode := asMap(data["pool"])
			stats := asMap(firstDefined(data["poolStats"], poolNode["poolStats"], data["stats"], data))
			network := asMap(firstDefined(data["networkStats"], poolNode["networkStats"], data["network"], map[string]interface{}{}))
			values["miners"] = firstDefined(stats["connectedMiners"], stats["miners"], stats["workers"], stats["connectedWorkers"])
			values["poolHashrate"] = normalizeHashrate(firstDefined(stats["poolHashRate"], stats["poolHashrate"], stats["hashrate"]))
			values["networkHashrate"] = normalizeHashrate(firstDefined(network["networkHashRate"], network["networkHashrate"], network["hashrate"]))
			values["blockHeight"] = firstDefined(network["blockHeight"], stats["blockHeight"], data["blockHeight"])
			values["estimatedReward"] = firstDefined(stats["estimatedReward"], stats["reward"], data["estimatedReward"])
			extra.Fee = formatPercent(firstDefined(poolNode["poolFeePercent"], data["poolFeePercent"]), 1)
			extra.Payout = pool.RewardMode
			message = chooseMessage(message, "Miningcore-compatible pool response normalized.")
		case "herominers-node":
			config := asMap(data["config"])
			poolStats := asMap(data["pool"])
			network := asMap(data["network"])
			values["miners"] = firstDefined(poolStats["miners"], poolStats["soloMiners"], poolStats["workers"])
			values["poolHashrate"] = normalizeHashrate(firstDefined(sumNumbers(poolStats["hashrate"], poolStats["soloHashrate"]), poolStats["hashrate"]))
			values["networkHashrate"] = normalizeHashrate(network["networkHashps"])
			values["blockHeight"] = network["height"]
			values["estimatedReward"] = firstDefined(poolStats["averageReward"], poolStats["daily_earnings"])
			extra.Fee = formatPercent(config["fee"], 1)
			extra.Payout = firstDefined(pool.RewardMode, "PROP/SOLO").(string)
			message = chooseMessage(message, "HeroMiners public stats endpoint normalized.")
		case "grandpool-api":
			node := selectGrandPool(data, pool)
			info := asMap(node["info"])
			stats := asMap(node["stats"])
			fee := asMap(info["fee"])
			mode := strings.ToUpper(strings.TrimSpace(pool.RewardMode))
			if mode == "" {
				mode = strings.ToUpper(fmt.Sprintf("%v", firstDefined(info["payout_mode"], "PROP")))
			}
			values["miners"] = firstDefined(stats["miners_count"], stats["workers_count"])
			values["poolHashrate"] = normalizeHashrate(firstDefined(stats["hashrate"], stats["avg_hashrate"]))
			values["blockHeight"] = stats["last_found_block_height"]
			values["estimatedReward"] = stats["blocks_count_24h"]
			if mode == "SOLO" {
				extra.Fee = formatPercent(fee["solo_fee"], 100)
			} else {
				extra.Fee = formatPercent(fee["fee"], 100)
			}
			extra.Payout = mode
			message = chooseMessage(message, "GrandPool public pools endpoint normalized.")
		case "nushypool-v2":
			node := selectNushyPool(data, pool)
			hashrate := asMap(node["hashrate"])
			values["miners"] = firstDefined(node["activeMiners"], node["activeWorkers"])
			values["poolHashrate"] = normalizeHashrate(hashrate["total"])
			values["blockHeight"] = parseBlockHeight(node["networkBlock"])
			values["estimatedReward"] = firstDefined(node["dailyRewardPerHashrateUnit"], node["baseBlockReward"])
			extra.Fee = formatPercent(node["poolFee"], 1)
			extra.Payout = fmt.Sprintf("%v", firstDefined(node["payoutSystem"], pool.RewardMode, "POOL"))
			message = chooseMessage(message, "NushyPool V2 public pool stats endpoint normalized.")
		case "nomp-pool":
			stats := asMap(firstDefined(data["pool_stats"], data["poolStats"], data["stats"], data))
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
	obs := PoolObservation{Timestamp: timestamp, PoolID: pool.ID, PoolName: pool.Name, Adapter: pool.Adapter, Reachable: reachable && data != nil, Miners: values["miners"], PoolHashrate: values["poolHashrate"], NetworkHashrate: values["networkHashrate"], BlockHeight: values["blockHeight"], EstimatedReward: values["estimatedReward"], LatencyMs: latency, Homepage: pool.Homepage, Message: message}
	if extra.Homepage != "" {
		obs.Homepage = extra.Homepage
	}
	obs.Fee = extra.Fee
	obs.Payout = extra.Payout
	obs.Share = extra.Share
	return obs
}
func stripHTML(value string) string {
	text := regexp.MustCompile("<[^>]+>").ReplaceAllString(value, "")
	return strings.Join(strings.Fields(html.UnescapeString(text)), " ")
}

func slug(value string) string {
	value = strings.ToLower(value)
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
		} else if !lastDash {
			b.WriteRune('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func parseHashrateNOPools(pool Pool, body string, latency *int64, timestamp string) []PoolObservation {
	rowRE := regexp.MustCompile("(?s)<li><div class='w3-row estimate'.*?</li>")
	urlRE := regexp.MustCompile(`window\.open\('([^']+)'`)
	modelRE := regexp.MustCompile("(?s)<(?:div|span) class='model'>(.*?)</(?:div|span)>")
	fieldRE := regexp.MustCompile("(?s)<div class='estimatesDescription'>(.*?)</div><div class='estimates'>(.*?)</div>")
	rows := rowRE.FindAllString(body, -1)
	observations := []PoolObservation{}
	for _, row := range rows {
		modelMatch := modelRE.FindStringSubmatch(row)
		if len(modelMatch) < 2 {
			continue
		}
		name := stripHTML(modelMatch[1])
		if name == "" {
			continue
		}
		urlValue := ""
		if m := urlRE.FindStringSubmatch(row); len(m) > 1 {
			urlValue = html.UnescapeString(m[1])
		}
		fee, payout, hashrate, share := "", "", "", ""
		for _, field := range fieldRE.FindAllStringSubmatch(row, -1) {
			description := stripHTML(field[1])
			value := stripHTML(field[2])
			switch {
			case strings.HasPrefix(description, "Fee"):
				fee = value
			case strings.HasPrefix(description, "Payout"):
				payout = value
			case strings.HasPrefix(description, "Hashrate"):
				hashrate = value
				share = strings.TrimSpace(strings.TrimPrefix(description, "Hashrate"))
			}
		}
		message := "Hashrate.no PRL pool index entry."
		if fee != "" || payout != "" || share != "" {
			message = strings.TrimSpace(fmt.Sprintf("%s Fee %s. Payout %s. Share %s.", message, fee, payout, share))
		}
		poolID := pool.ID + "-" + slug(name+"-"+payout)
		observations = append(observations, PoolObservation{Timestamp: timestamp, PoolID: poolID, PoolName: name, Adapter: pool.Adapter, Reachable: true, PoolHashrate: hashrate, LatencyMs: latency, Homepage: urlValue, Fee: fee, Payout: payout, Share: share, Message: message})
	}
	return observations
}

func fetchText(endpoint string) (string, int64, error) {
	started := time.Now()
	request, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return "", 0, err
	}
	request.Header.Set("accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	request.Header.Set("user-agent", "PearlGuard-Desktop/0.3.0")
	client := appHTTPClient(9000*time.Millisecond, endpoint)
	response, err := client.Do(request)
	if err != nil {
		return "", 0, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", 0, err
	}
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return "", 0, fmt.Errorf("HTTP %d", response.StatusCode)
	}
	return string(body), time.Since(started).Milliseconds(), nil
}

func fetchJSON(endpoint string) (map[string]interface{}, int64, error) {
	return fetchPoolJSON(Pool{Endpoint: endpoint, Method: http.MethodGet})
}

func quoteNumber(node map[string]interface{}, paths ...string) *float64 {
	for _, path := range paths {
		var value interface{}
		if strings.Contains(path, ".") {
			value = pick(node, path)
		} else {
			value = node[path]
		}
		if n, ok := numberFrom(value); ok {
			return &n
		}
	}
	return nil
}

func mergeQuoteNode(parent map[string]interface{}, child map[string]interface{}) map[string]interface{} {
	merged := map[string]interface{}{}
	for key, value := range parent {
		merged[key] = value
	}
	for key, value := range child {
		merged[key] = value
	}
	return merged
}

func selectQuoteNode(raw interface{}, market string) map[string]interface{} {
	market = strings.ToLower(strings.ReplaceAll(market, "_", ""))
	matchMarket := func(node map[string]interface{}) bool {
		for _, key := range []string{"market", "marketId", "market_id", "symbol", "pair", "name"} {
			value := strings.ToLower(strings.ReplaceAll(fmt.Sprintf("%v", node[key]), "_", ""))
			if value == market || value == "prlusdt" || value == "prl/usdt" {
				return true
			}
		}
		base := strings.ToUpper(fmt.Sprintf("%v", firstDefined(node["base"], node["base_unit"], node["baseUnit"])))
		quote := strings.ToUpper(fmt.Sprintf("%v", firstDefined(node["quote"], node["quote_unit"], node["quoteUnit"])))
		return base == "PRL" && quote == "USDT"
	}
	if node, ok := raw.(map[string]interface{}); ok {
		for _, key := range []string{market, strings.ToUpper(market), "prlusdt", "PRLUSDT"} {
			if child, ok := node[key].(map[string]interface{}); ok {
				return child
			}
		}
		for _, key := range []string{"ticker", "data", "result"} {
			if child, ok := node[key].(map[string]interface{}); ok {
				return mergeQuoteNode(node, child)
			}
			if list, ok := node[key].([]interface{}); ok {
				for _, item := range list {
					child := asMap(item)
					if matchMarket(child) {
						return child
					}
				}
			}
		}
		return node
	}
	if list, ok := raw.([]interface{}); ok {
		for _, item := range list {
			node := asMap(item)
			if matchMarket(node) {
				return node
			}
		}
	}
	return map[string]interface{}{}
}

func (a *App) GetMarketQuote() MarketQuote {
	const endpoint = "https://safe.trade/api/v2/trade/public/tickers/prlusdt"
	started := time.Now()
	timestamp := nowISO()
	request, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return MarketQuote{OK: false, Source: "SafeTrade", Market: "PRL/USDT", Timestamp: timestamp, Message: err.Error()}
	}
	request.Header.Set("accept", "application/json,text/plain,*/*")
	request.Header.Set("user-agent", "PearlGuard-Desktop/0.3.0")
	client := appHTTPClient(9000*time.Millisecond, endpoint)
	response, err := client.Do(request)
	if err != nil {
		return MarketQuote{OK: false, Source: "SafeTrade", Market: "PRL/USDT", Timestamp: timestamp, Message: "SafeTrade quote is unavailable. Check the network or configured proxy."}
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 1024*1024))
	if err != nil {
		return MarketQuote{OK: false, Source: "SafeTrade", Market: "PRL/USDT", Timestamp: timestamp, Message: "SafeTrade quote response could not be read."}
	}
	latency := time.Since(started).Milliseconds()
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return MarketQuote{OK: false, Source: "SafeTrade", Market: "PRL/USDT", Timestamp: timestamp, LatencyMs: &latency, Message: fmt.Sprintf("SafeTrade quote unavailable (HTTP %d).", response.StatusCode)}
	}
	var raw interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return MarketQuote{OK: false, Source: "SafeTrade", Market: "PRL/USDT", Timestamp: timestamp, LatencyMs: &latency, Message: "SafeTrade quote endpoint did not return JSON."}
	}
	node := selectQuoteNode(raw, "prlusdt")
	quote := MarketQuote{OK: true, Source: "SafeTrade", Market: "PRL/USDT", Timestamp: timestamp, LatencyMs: &latency, Message: "SafeTrade quote refreshed."}
	quote.Last = quoteNumber(node, "last", "last_price", "lastPrice", "price", "close", "ticker.last", "data.last")
	quote.Bid = quoteNumber(node, "bid", "buy", "best_bid", "bestBid", "ticker.bid")
	quote.Ask = quoteNumber(node, "ask", "sell", "best_ask", "bestAsk", "ticker.ask")
	quote.High24h = quoteNumber(node, "high", "high24h", "high_24h", "ticker.high")
	quote.Low24h = quoteNumber(node, "low", "low24h", "low_24h", "ticker.low")
	quote.Volume24h = quoteNumber(node, "volume", "volume24h", "volume_24h", "base_volume", "baseVolume", "ticker.volume")
	quote.ChangePercent = quoteNumber(node, "change", "price_change_percent", "priceChangePercent", "changePercent")
	if quote.Last == nil {
		quote.OK = false
		quote.Message = "SafeTrade quote response did not include a last price."
	}
	return quote
}

func fetchPoolJSON(pool Pool) (map[string]interface{}, int64, error) {
	started := time.Now()
	method := strings.ToUpper(strings.TrimSpace(pool.Method))
	if method == "" {
		method = http.MethodGet
	}
	if method != http.MethodGet && method != http.MethodPost {
		return nil, 0, fmt.Errorf("unsupported HTTP method %s", method)
	}
	var body io.Reader
	if method == http.MethodPost {
		payload := pool.Body
		if payload == nil {
			payload = map[string]interface{}{}
		}
		raw, err := json.Marshal(payload)
		if err != nil {
			return nil, 0, err
		}
		body = bytes.NewReader(raw)
	}
	request, err := http.NewRequest(method, pool.Endpoint, body)
	if err != nil {
		return nil, 0, err
	}
	request.Header.Set("accept", "application/json,text/plain,*/*")
	request.Header.Set("user-agent", "PearlGuard-Desktop/0.3.0")
	if method == http.MethodPost {
		request.Header.Set("content-type", "application/json")
	}
	client := appHTTPClient(9000*time.Millisecond, pool.Endpoint)
	response, err := client.Do(request)
	if err != nil {
		return nil, 0, err
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, 0, err
	}
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return nil, 0, fmt.Errorf("HTTP %d", response.StatusCode)
	}
	var data map[string]interface{}
	if err := json.Unmarshal(responseBody, &data); err != nil {
		return nil, 0, fmt.Errorf("endpoint did not return JSON")
	}
	return data, time.Since(started).Milliseconds(), nil
}

type poolSyncOutput struct {
	Index        int
	Observations []PoolObservation
	Errors       []map[string]string
}

func syncPool(index int, pool Pool, timestamp string) poolSyncOutput {
	output := poolSyncOutput{Index: index, Observations: []PoolObservation{}, Errors: []map[string]string{}}
	if !pool.Enabled || strings.TrimSpace(pool.Endpoint) == "" {
		output.Observations = append(output.Observations, normalizePoolObservation(pool, nil, false, nil, timestamp, "Pool is saved but disabled or missing an endpoint."))
		return output
	}
	if pool.Adapter == "hashrate-no-prl-pools" {
		body, latency, err := fetchText(pool.Endpoint)
		if err != nil {
			msg := err.Error()
			output.Errors = append(output.Errors, map[string]string{"poolId": pool.ID, "message": msg})
			output.Observations = append(output.Observations, normalizePoolObservation(pool, nil, false, nil, timestamp, msg))
			return output
		}
		latencyPtr := latency
		parsed := parseHashrateNOPools(pool, body, &latencyPtr, timestamp)
		if len(parsed) == 0 {
			msg := "No PRL pool rows were parsed from Hashrate.no."
			output.Errors = append(output.Errors, map[string]string{"poolId": pool.ID, "message": msg})
			output.Observations = append(output.Observations, normalizePoolObservation(pool, nil, false, nil, timestamp, msg))
			return output
		}
		output.Observations = append(output.Observations, parsed...)
		return output
	}
	data, latency, err := fetchPoolJSON(pool)
	if err != nil {
		msg := err.Error()
		output.Errors = append(output.Errors, map[string]string{"poolId": pool.ID, "message": msg})
		output.Observations = append(output.Observations, normalizePoolObservation(pool, nil, false, nil, timestamp, msg))
		return output
	}
	output.Observations = append(output.Observations, normalizePoolObservation(pool, data, true, &latency, timestamp, ""))
	return output
}

func (a *App) SyncPools(options map[string]interface{}) PoolSyncResult {
	timestamp := nowISO()
	settings, _, _ := readWalletConfig()
	if !settings.MiningPoolEnabled {
		return PoolSyncResult{Timestamp: timestamp, Mode: "disabled", Observations: []PoolObservation{}, Errors: []map[string]string{}}
	}
	config := readPoolConfig()
	outputs := make([]poolSyncOutput, len(config.Pools))
	var wg sync.WaitGroup
	limit := make(chan struct{}, 6)
	for index, pool := range config.Pools {
		wg.Add(1)
		go func(index int, pool Pool) {
			defer wg.Done()
			limit <- struct{}{}
			defer func() { <-limit }()
			outputs[index] = syncPool(index, pool, timestamp)
		}(index, pool)
	}
	wg.Wait()
	observations := []PoolObservation{}
	errorsList := []map[string]string{}
	for _, output := range outputs {
		observations = append(observations, output.Observations...)
		errorsList = append(errorsList, output.Errors...)
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
		config.Pools[index].Endpoint = strings.TrimSpace(config.Pools[index].Endpoint)
		config.Pools[index].Method = strings.ToUpper(strings.TrimSpace(config.Pools[index].Method))
		if config.Pools[index].Method != "" && config.Pools[index].Method != http.MethodPost {
			config.Pools[index].Method = http.MethodGet
		}
		config.Pools[index].RewardMode = strings.ToUpper(strings.TrimSpace(config.Pools[index].RewardMode))
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
