package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// writeConfigFile writes content to a temp file and returns the path.
func writeConfigFile(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "config-*.yaml")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	f.Close()
	return f.Name()
}

// setEnv sets env vars for the duration of the test and restores them on cleanup.
func setEnv(t *testing.T, pairs ...string) {
	t.Helper()
	if len(pairs)%2 != 0 {
		t.Fatal("setEnv requires an even number of arguments (key, value pairs)")
	}
	for i := 0; i < len(pairs); i += 2 {
		key, val := pairs[i], pairs[i+1]
		old, existed := os.LookupEnv(key)
		t.Cleanup(func() {
			if existed {
				os.Setenv(key, old)
			} else {
				os.Unsetenv(key)
			}
		})
		os.Setenv(key, val)
	}
}

// minimalValidConfig returns a YAML string with the minimum required fields.
func minimalValidConfig(t *testing.T) string {
	t.Helper()
	// Create a throwaway private key file so path validation passes at load time.
	keyFile, err := os.CreateTemp(t.TempDir(), "key-*.pem")
	if err != nil {
		t.Fatalf("create key file: %v", err)
	}
	keyFile.Close()
	return `
server:
  token: test-token
tenancies:
  - name: my-tenancy
    region: us-ashburn-1
    tenancy_id: ocid1.tenancy.oc1..aaaa
    user_id: ocid1.user.oc1..aaaa
    fingerprint: "aa:bb:cc"
    private_key_path: ` + keyFile.Name() + `
`
}

// ---- defaults ---------------------------------------------------------------

func TestDefaults(t *testing.T) {
	d := defaults()

	if d.Server.Port != 8080 {
		t.Errorf("default port: got %d, want 8080", d.Server.Port)
	}
	if d.Discovery.TagKey != "monitoring" {
		t.Errorf("default tag_key: got %q, want %q", d.Discovery.TagKey, "monitoring")
	}
	if d.Discovery.TagValue != "enabled" {
		t.Errorf("default tag_value: got %q, want %q", d.Discovery.TagValue, "enabled")
	}
	if d.Discovery.LinuxPort != 9100 {
		t.Errorf("default linux_port: got %d, want 9100", d.Discovery.LinuxPort)
	}
	if d.Discovery.WindowsPort != 9182 {
		t.Errorf("default windows_port: got %d, want 9182", d.Discovery.WindowsPort)
	}
	if d.Discovery.RefreshInterval != 5*time.Minute {
		t.Errorf("default refresh_interval: got %v, want 5m", d.Discovery.RefreshInterval)
	}
	if d.Discovery.RateLimitRPS != 10.0 {
		t.Errorf("default rate_limit_rps: got %f, want 10.0", d.Discovery.RateLimitRPS)
	}
	if d.Server.Token != "" {
		t.Errorf("default token should be empty, got %q", d.Server.Token)
	}
}

// ---- Load: missing file is OK -----------------------------------------------

func TestLoad_NoFileWithEnvVars(t *testing.T) {
	setEnv(t,
		"CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.yaml"),
		"SERVER_TOKEN", "mytoken",
	)
	// Also provide a minimal tenancy via the env-var path is not possible
	// (tenancies require the YAML file), so point to a valid file that has
	// tenancy config but let the token come from the env.
	cfg := minimalValidConfig(t)
	path := writeConfigFile(t, cfg)
	setEnv(t, "CONFIG_PATH", path, "SERVER_TOKEN", "env-token")

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	// Env var takes precedence over file value
	if got.Server.Token != "env-token" {
		t.Errorf("token: got %q, want %q", got.Server.Token, "env-token")
	}
}

// ---- Load: YAML parsing -----------------------------------------------------

func TestLoad_ParsesYAML(t *testing.T) {
	tmp := t.TempDir()
	keyPath := filepath.Join(tmp, "key.pem")
	if err := os.WriteFile(keyPath, []byte("dummy"), 0600); err != nil {
		t.Fatalf("write key file: %v", err)
	}

	yaml := `
server:
  port: 9090
  token: file-token
discovery:
  tag_key: env
  tag_value: true
  linux_port: 8888
  windows_port: 7777
  refresh_interval: 2m
  rate_limit_rps: 5.0
tenancies:
  - name: prod
    region: eu-frankfurt-1
    tenancy_id: ocid1.tenancy.oc1..prod
    user_id: ocid1.user.oc1..prod
    fingerprint: "11:22:33"
    private_key_path: ` + keyPath + `
    compartments:
      - ocid1.compartment.oc1..aaa
`
	path := writeConfigFile(t, yaml)
	setEnv(t, "CONFIG_PATH", path)

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if got.Server.Port != 9090 {
		t.Errorf("port: got %d, want 9090", got.Server.Port)
	}
	if got.Server.Token != "file-token" {
		t.Errorf("token: got %q, want %q", got.Server.Token, "file-token")
	}
	if got.Discovery.TagKey != "env" {
		t.Errorf("tag_key: got %q, want %q", got.Discovery.TagKey, "env")
	}
	if got.Discovery.LinuxPort != 8888 {
		t.Errorf("linux_port: got %d, want 8888", got.Discovery.LinuxPort)
	}
	if got.Discovery.WindowsPort != 7777 {
		t.Errorf("windows_port: got %d, want 7777", got.Discovery.WindowsPort)
	}
	if got.Discovery.RefreshInterval != 2*time.Minute {
		t.Errorf("refresh_interval: got %v, want 2m", got.Discovery.RefreshInterval)
	}
	if got.Discovery.RateLimitRPS != 5.0 {
		t.Errorf("rate_limit_rps: got %f, want 5.0", got.Discovery.RateLimitRPS)
	}
	if len(got.Tenancies) != 1 {
		t.Fatalf("tenancies count: got %d, want 1", len(got.Tenancies))
	}
	if got.Tenancies[0].Name != "prod" {
		t.Errorf("tenancy name: got %q, want %q", got.Tenancies[0].Name, "prod")
	}
	if len(got.Tenancies[0].Compartments) != 1 {
		t.Errorf("compartments count: got %d, want 1", len(got.Tenancies[0].Compartments))
	}
}

// ---- Load: env var overrides ------------------------------------------------

func TestLoad_EnvVarOverrides(t *testing.T) {
	path := writeConfigFile(t, minimalValidConfig(t))

	setEnv(t,
		"CONFIG_PATH", path,
		"SERVER_PORT", "9999",
		"DISCOVERY_TAG_KEY", "my-tag",
		"DISCOVERY_TAG_VALUE", "yes",
		"DISCOVERY_LINUX_PORT", "1234",
		"DISCOVERY_WINDOWS_PORT", "5678",
		"DISCOVERY_REFRESH_INTERVAL", "10m",
		"DISCOVERY_RATE_LIMIT_RPS", "20.5",
	)

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if got.Server.Port != 9999 {
		t.Errorf("port: got %d, want 9999", got.Server.Port)
	}
	if got.Discovery.TagKey != "my-tag" {
		t.Errorf("tag_key: got %q, want %q", got.Discovery.TagKey, "my-tag")
	}
	if got.Discovery.TagValue != "yes" {
		t.Errorf("tag_value: got %q, want %q", got.Discovery.TagValue, "yes")
	}
	if got.Discovery.LinuxPort != 1234 {
		t.Errorf("linux_port: got %d, want 1234", got.Discovery.LinuxPort)
	}
	if got.Discovery.WindowsPort != 5678 {
		t.Errorf("windows_port: got %d, want 5678", got.Discovery.WindowsPort)
	}
	if got.Discovery.RefreshInterval != 10*time.Minute {
		t.Errorf("refresh_interval: got %v, want 10m", got.Discovery.RefreshInterval)
	}
	if got.Discovery.RateLimitRPS != 20.5 {
		t.Errorf("rate_limit_rps: got %f, want 20.5", got.Discovery.RateLimitRPS)
	}
}

// ---- Load: invalid env var values -------------------------------------------

func TestLoad_InvalidEnvVars(t *testing.T) {
	path := writeConfigFile(t, minimalValidConfig(t))

	cases := []struct {
		name   string
		envKey string
		val    string
	}{
		{"invalid SERVER_PORT", "SERVER_PORT", "not-a-number"},
		{"invalid DISCOVERY_LINUX_PORT", "DISCOVERY_LINUX_PORT", "abc"},
		{"invalid DISCOVERY_WINDOWS_PORT", "DISCOVERY_WINDOWS_PORT", "xyz"},
		{"invalid DISCOVERY_REFRESH_INTERVAL", "DISCOVERY_REFRESH_INTERVAL", "bad-duration"},
		{"invalid DISCOVERY_RATE_LIMIT_RPS", "DISCOVERY_RATE_LIMIT_RPS", "nope"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setEnv(t, "CONFIG_PATH", path, tc.envKey, tc.val)
			_, err := Load()
			if err == nil {
				t.Errorf("expected error for %s=%q, got nil", tc.envKey, tc.val)
			}
		})
	}
}

// ---- Load: invalid YAML -----------------------------------------------------

func TestLoad_InvalidYAML(t *testing.T) {
	path := writeConfigFile(t, ":::not valid yaml:::")
	setEnv(t, "CONFIG_PATH", path)

	_, err := Load()
	if err == nil {
		t.Error("expected error for invalid YAML, got nil")
	}
}

// ---- validate ---------------------------------------------------------------

func TestValidate_MissingToken(t *testing.T) {
	c := defaults()
	c.Tenancies = []TenancyConfig{{
		Name: "t", Region: "r", TenancyID: "tid", UserID: "uid",
		Fingerprint: "fp", PrivateKeyPath: "/k",
	}}
	if err := c.validate(); err == nil {
		t.Error("expected error for missing token")
	}
}

func TestValidate_NoTenancies(t *testing.T) {
	c := defaults()
	c.Server.Token = "tok"
	if err := c.validate(); err == nil {
		t.Error("expected error for no tenancies")
	}
}

func TestValidate_TenancyRequiredFields(t *testing.T) {
	base := TenancyConfig{
		Name: "t", Region: "r", TenancyID: "tid", UserID: "uid",
		Fingerprint: "fp", PrivateKeyPath: "/k",
	}

	cases := []struct {
		name   string
		mutate func(*TenancyConfig)
	}{
		{"missing name", func(c *TenancyConfig) { c.Name = "" }},
		{"missing tenancy_id", func(c *TenancyConfig) { c.TenancyID = "" }},
		{"missing user_id", func(c *TenancyConfig) { c.UserID = "" }},
		{"missing region", func(c *TenancyConfig) { c.Region = "" }},
		{"missing fingerprint", func(c *TenancyConfig) { c.Fingerprint = "" }},
		{"missing private_key_path", func(c *TenancyConfig) { c.PrivateKeyPath = "" }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc := tc
			tenancy := base
			tc.mutate(&tenancy)
			cfg := defaults()
			cfg.Server.Token = "tok"
			cfg.Tenancies = []TenancyConfig{tenancy}
			if err := cfg.validate(); err == nil {
				t.Errorf("expected validation error for %s", tc.name)
			}
		})
	}
}

func TestValidate_InstancePrincipalAuth(t *testing.T) {
	// instance_principal does not require user_id, fingerprint, or private_key_path
	cfg := defaults()
	cfg.Server.Token = "tok"
	cfg.Tenancies = []TenancyConfig{{
		Name: "t", Region: "r", TenancyID: "tid",
		AuthType: "instance_principal",
	}}
	if err := cfg.validate(); err != nil {
		t.Errorf("unexpected error for instance_principal tenancy: %v", err)
	}
}

func TestValidate_InstancePrincipalStillRequiresRegionAndTenancyID(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*TenancyConfig)
	}{
		{"missing name", func(c *TenancyConfig) { c.Name = "" }},
		{"missing tenancy_id", func(c *TenancyConfig) { c.TenancyID = "" }},
		{"missing region", func(c *TenancyConfig) { c.Region = "" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tenancy := TenancyConfig{
				Name: "t", Region: "r", TenancyID: "tid",
				AuthType: "instance_principal",
			}
			tc.mutate(&tenancy)
			cfg := defaults()
			cfg.Server.Token = "tok"
			cfg.Tenancies = []TenancyConfig{tenancy}
			if err := cfg.validate(); err == nil {
				t.Errorf("expected validation error for %s", tc.name)
			}
		})
	}
}

func TestValidate_UnknownAuthType(t *testing.T) {
	cfg := defaults()
	cfg.Server.Token = "tok"
	cfg.Tenancies = []TenancyConfig{{
		Name: "t", Region: "r", TenancyID: "tid",
		AuthType: "magic_beans",
	}}
	if err := cfg.validate(); err == nil {
		t.Error("expected validation error for unknown auth_type")
	}
}

func TestValidate_ValidConfig(t *testing.T) {
	c := defaults()
	c.Server.Token = "tok"
	c.Tenancies = []TenancyConfig{{
		Name: "t", Region: "r", TenancyID: "tid", UserID: "uid",
		Fingerprint: "fp", PrivateKeyPath: "/k",
	}}
	if err := c.validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidate_EmptyCompartmentsIsValid(t *testing.T) {
	c := defaults()
	c.Server.Token = "tok"
	c.Tenancies = []TenancyConfig{{
		Name: "t", Region: "r", TenancyID: "tid", UserID: "uid",
		Fingerprint: "fp", PrivateKeyPath: "/k",
		Compartments: []string{}, // explicitly empty - should be valid
	}}
	if err := c.validate(); err != nil {
		t.Errorf("empty compartments should be valid: %v", err)
	}
}

// ---- envStr -----------------------------------------------------------------

func TestEnvStr(t *testing.T) {
	setEnv(t, "TEST_ENVSTR_KEY", "hello")
	if got := envStr("TEST_ENVSTR_KEY", "fallback"); got != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}
	os.Unsetenv("TEST_ENVSTR_KEY")
	if got := envStr("TEST_ENVSTR_KEY", "fallback"); got != "fallback" {
		t.Errorf("got %q, want %q", got, "fallback")
	}
}
