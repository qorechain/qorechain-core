package orchestrator

// Config holds sidecar orchestrator configuration.
type Config struct {
	Enabled           bool                   `toml:"enabled"`
	RegistryURL       string                 `toml:"registry_url"`
	CredentialAPI     string                 `toml:"credential_api"`
	DataDir           string                 `toml:"data_dir"`
	GRPCSidecarAddr   string                 `toml:"grpc_sidecar_addr"`
	GRPCAdminAddr     string                 `toml:"grpc_admin_addr"`
	ReconcileInterval string                 `toml:"reconcile_interval"`
	ChainConfigs      map[string]ChainConfig `toml:"chains"`
}

// ChainConfig holds per-chain sidecar configuration.
type ChainConfig struct {
	Image      string            `toml:"image"`
	Version    string            `toml:"version"`
	ExtraEnv   map[string]string `toml:"extra_env"`
	ExtraPorts []string          `toml:"extra_ports"`
}

// DefaultConfig returns default orchestrator config.
func DefaultConfig() Config {
	return Config{
		Enabled:           false,
		RegistryURL:       "ghcr.io/qorechain",
		CredentialAPI:     "https://api.qorechain.io/v1/credentials",
		GRPCSidecarAddr:   ":9900",
		GRPCAdminAddr:     "127.0.0.1:9901",
		ReconcileInterval: "60s",
		ChainConfigs:      make(map[string]ChainConfig),
	}
}

// ImageForChain returns the full Docker image reference for a chain.
func (c Config) ImageForChain(chain string) string {
	if cc, ok := c.ChainConfigs[chain]; ok && cc.Image != "" {
		return cc.Image
	}
	return c.RegistryURL + "/sidecar-" + chain
}

// VersionForChain returns the image version tag for a chain.
func (c Config) VersionForChain(chain string) string {
	if cc, ok := c.ChainConfigs[chain]; ok && cc.Version != "" {
		return cc.Version
	}
	return "latest"
}
