package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type ValidatorIdentity struct {
	Name              string    `json:"name"`
	Address           string    `json:"address"`
	PublicKey         string    `json:"public_key"`
	PrivateKey        string    `json:"private_key"`
	Stake             uint64    `json:"stake"`
	Country           string    `json:"country"`
	Region            string    `json:"region"`
	City              string    `json:"city"`
	IPAddress         string    `json:"ip_address"`
	P2PPort           int       `json:"p2p_port"`
	APIPort           int       `json:"api_port"`
	MetricsPort       int       `json:"metrics_port"`
	UptimeSLA         string    `json:"uptime_sla"`
	Consensus         string    `json:"consensus"`
	Network           string    `json:"network"`
	TelemetryEndpoint string    `json:"telemetry_endpoint"`
	RegisteredAt       time.Time `json:"registered_at"`
	LastHeartbeatAt   time.Time `json:"last_heartbeat_at"`
}

type ValidatorManifest struct {
	Network        string             `json:"network"`
	Chain          string             `json:"chain"`
	Version        string             `json:"version"`
	GeneratedAt    time.Time          `json:"generated_at"`
	Validators     []ValidatorIdentity `json:"validators"`
	BootstrapSeeds []string           `json:"bootstrap_seeds"`
	TelemetryURL   string             `json:"telemetry_url"`
}

func main() {
	name := flag.String("name", "", "Validator name (e.g., validator-kenya-01)")
	country := flag.String("country", "", "ISO 3166-1 alpha-2 country code (e.g., KE)")
	region := flag.String("region", "", "Region or state (e.g., Nairobi)")
	city := flag.String("city", "", "City name")
	stake := flag.Uint64("stake", 0, "Stake amount in native token")
	network := flag.String("network", "tdr-testnet", "Network name")
	consensus := flag.String("consensus", "pos", "Consensus type: pos or poa")
	output := flag.String("output", "", "Output directory for validator artifacts")
	p2pPort := flag.Int("p2p-port", 3030, "P2P listen port")
	apiPort := flag.Int("api-port", 8080, "API port")
	metricsPort := flag.Int("metrics-port", 9090, "Prometheus metrics port")
	telemetryURL := flag.String("telemetry", "https://telemetry.tender.network/api/v1/register", "Telemetry endpoint")
	interactive := flag.Bool("interactive", false, "Interactive mode with prompts")
	flag.Parse()

	if *interactive {
		runInteractive()
		return
	}

	if *name == "" || *country == "" || *stake == 0 {
		fmt.Fprintln(os.Stderr, "usage: validator-onboard [--name NAME] [--country CODE] [--stake AMOUNT] [--output DIR]")
		os.Exit(1)
	}

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	publicKey := hex.EncodeToString(pub)
	privateKey := hex.EncodeToString(priv)
	addressBytes := sha256.Sum256(pub)
	address := hex.EncodeToString(addressBytes[:])

	ip, _ := detectLocalAddress()
	p2pAddr := fmt.Sprintf("%s:%d", ip, *p2pPort)
	apiAddr := fmt.Sprintf("%s:%d", ip, *apiPort)

	identity := ValidatorIdentity{
		Name:              *name,
		Address:           address,
		PublicKey:         publicKey,
		PrivateKey:        privateKey,
		Stake:             *stake,
		Country:           strings.ToUpper(*country),
		Region:            *region,
		City:              *city,
		IPAddress:         p2pAddr,
		P2PPort:           *p2pPort,
		APIPort:           *apiPort,
		MetricsPort:       *metricsPort,
		UptimeSLA:         "99.90%",
		Consensus:         *consensus,
		Network:           *network,
		TelemetryEndpoint: *telemetryURL,
		RegisteredAt:       time.Now().UTC(),
		LastHeartbeatAt:   time.Now().UTC(),
	}

	outDir := *output
	if outDir == "" {
		outDir = filepath.Join(".", "validator-"+*name)
	}
	_ = os.MkdirAll(outDir, 0o700)

	manifest := ValidatorManifest{
		Network:   *network,
		Chain:     "go-chain",
		Version:   "v1.0.0",
		GeneratedAt: time.Now().UTC(),
		Validators: []ValidatorIdentity{identity},
		BootstrapSeeds: []string{p2pAddr},
		TelemetryURL: *telemetryURL,
	}

	manifestPath := filepath.Join(outDir, "validator_manifest.json")
	writeJSON(manifestPath, manifest, 0o644)

	keyPath := filepath.Join(outDir, "validator.key.json")
	writeJSON(keyPath, identity, 0o600)

	publicIdentity := identity
	publicIdentity.PrivateKey = ""
	publicPath := filepath.Join(outDir, "validator.public.json")
	writeJSON(publicPath, publicIdentity, 0o644)

	envContent := fmt.Sprintf(`TENDER_VALIDATOR_NAME=%s
TENDER_VALIDATOR_ADDRESS=%s
TENDER_VALIDATOR_PUBLIC_KEY=%s
TENDER_P2P_PORT=%d
TENDER_API_PORT=%d
TENDER_METRICS_PORT=%d
TENDER_DATA_DIR=%s
TENDER_CONSENSUS=%s
TENDER_TELEMETRY_ENDPOINT=%s
`,
		identity.Name, identity.Address, identity.PublicKey,
		identity.P2PPort, identity.APIPort, identity.MetricsPort,
		filepath.Join(outDir, "data"), identity.Consensus, identity.TelemetryEndpoint)

	envPath := filepath.Join(outDir, ".env")
	if err := os.WriteFile(envPath, []byte(envContent), 0o600); err != nil {
		panic(err)
	}

	telemetryPayload := map[string]any{
		"validator_address": identity.Address,
		"validator_name":    identity.Name,
		"public_key":        identity.PublicKey,
		"country":           identity.Country,
		"region":            identity.Region,
		"city":              identity.City,
		"ip_address":        identity.IPAddress,
		"network":           identity.Network,
		"consensus":         identity.Consensus,
		"stake":             identity.Stake,
		"uptime_sla":        identity.UptimeSLA,
		"registered_at":     identity.RegisteredAt,
	}

	telemetryPath := filepath.Join(outDir, "telemetry_payload.json")
	writeJSON(telemetryPath, telemetryPayload, 0o644)

	fmt.Printf("validator identity generated successfully\n")
	fmt.Printf("  address:         %s\n", identity.Address)
	fmt.Printf("  public key:      %s\n", identity.PublicKey)
	fmt.Printf("  files written:   %s\n", outDir)
	fmt.Printf("  manifest:        %s\n", manifestPath)
	fmt.Printf("  telemetry:       %s\n", telemetryPath)
	fmt.Printf("  p2p address:     %s\n", p2pAddr)
	fmt.Printf("  api endpoint:    http://%s\n", apiAddr)
}

func detectLocalAddress() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1", nil
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String(), nil
}

func writeJSON(path string, v any, perm os.FileMode) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(path, data, perm); err != nil {
		panic(err)
	}
}

func runInteractive() {
	fmt.Println("TDR Validator Onboarding - Interactive Mode")
	fmt.Println(strings.Repeat("=", 50))

	var name, country, region, city, consensus, network, telemetry string
	var stake uint64
	var p2pPort, apiPort, metricsPort int

	fmt.Print("Validator name: ")
	fmt.Scanln(&name)
	fmt.Print("Country code (ISO 3166-1 alpha-2): ")
	fmt.Scanln(&country)
	fmt.Print("Region: ")
	fmt.Scanln(&region)
	fmt.Print("City: ")
	fmt.Scanln(&city)
	fmt.Print("Stake amount: ")
	fmt.Scanf("%d", &stake)
	fmt.Print("Network (default tdr-testnet): ")
	fmt.Scanln(&network)
	if network == "" {
		network = "tdr-testnet"
	}
	fmt.Print("Consensus (default pos): ")
	fmt.Scanln(&consensus)
	if consensus == "" {
		consensus = "pos"
	}
	fmt.Print("P2P port (default 3030): ")
	fmt.Scanf("%d", &p2pPort)
	if p2pPort == 0 {
		p2pPort = 3030
	}
	fmt.Print("API port (default 8080): ")
	fmt.Scanf("%d", &apiPort)
	if apiPort == 0 {
		apiPort = 8080
	}
	fmt.Print("Metrics port (default 9090): ")
	fmt.Scanf("%d", &metricsPort)
	if metricsPort == 0 {
		metricsPort = 9090
	}
	fmt.Print("Telemetry endpoint (default https://telemetry.tender.network/api/v1/register): ")
	fmt.Scanln(&telemetry)
	if telemetry == "" {
		telemetry = "https://telemetry.tender.network/api/v1/register"
	}

	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	publicKey := hex.EncodeToString(pub)
	privateKey := hex.EncodeToString(priv)
	addressBytes := sha256.Sum256(pub)
	address := hex.EncodeToString(addressBytes[:])

	identity := ValidatorIdentity{
		Name:              name,
		Address:           address,
		PublicKey:         publicKey,
		PrivateKey:        privateKey,
		Stake:             stake,
		Country:           strings.ToUpper(country),
		Region:            region,
		City:              city,
		P2PPort:           p2pPort,
		APIPort:           apiPort,
		MetricsPort:       metricsPort,
		UptimeSLA:         "99.90%",
		Consensus:         consensus,
		Network:           network,
		TelemetryEndpoint: telemetry,
		RegisteredAt:       time.Now().UTC(),
		LastHeartbeatAt:   time.Now().UTC(),
	}

	outDir := filepath.Join(".", "validator-"+name)
	_ = os.MkdirAll(outDir, 0o700)

	manifest := ValidatorManifest{
		Network:   network,
		Chain:     "go-chain",
		Version:   "v1.0.0",
		GeneratedAt: time.Now().UTC(),
		Validators: []ValidatorIdentity{identity},
		TelemetryURL: telemetry,
	}

	writeJSON(filepath.Join(outDir, "validator_manifest.json"), manifest, 0o644)
	writeJSON(filepath.Join(outDir, "validator.key.json"), identity, 0o600)

	fmt.Printf("\nValidator registered at %s\n", outDir)
	fmt.Printf("Address: %s\n", address)
}
