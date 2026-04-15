package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadConfigFileNotFound(t *testing.T) {
	cfg, err := Load("/nonexistent/path/grctl.yaml")
	if err != nil {
		t.Fatalf("load config with missing file: %v", err)
	}

	// Should return defaults
	if cfg.NATS.Mode != NATSModeEmbedded {
		t.Fatalf("expected default nats.mode %q, got %q", NATSModeEmbedded, cfg.NATS.Mode)
	}
	if cfg.NATS.ConfigFile != "" {
		t.Fatalf("expected default nats.config_file to be empty, got %q", cfg.NATS.ConfigFile)
	}
	if cfg.NATS.Port != 4225 {
		t.Fatalf("expected default port %d, got %d", 4225, cfg.NATS.Port)
	}
	if cfg.Streams.Storage != "file" {
		t.Fatalf("expected default streams.storage %q, got %q", "file", cfg.Streams.Storage)
	}
	if cfg.Defaults.WorkerResponseTimeout != 5*time.Second {
		t.Fatalf("expected default worker_response_timeout 5s, got %v", cfg.Defaults.WorkerResponseTimeout)
	}
	if cfg.Defaults.StepTimeout != 5*time.Minute {
		t.Fatalf("expected default step_timeout 5m, got %v", cfg.Defaults.StepTimeout)
	}
}

func TestLoadConfigFile(t *testing.T) {
	configData := []byte(`nats:
  mode: external
  url: "nats://127.0.0.1:4222"
  port: 4333
streams:
  storage: memory
defaults:
  worker_response_timeout: "45s"
  step_timeout: "2m"
`)

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "grctl.yaml")

	err := os.WriteFile(configPath, configData, 0o600)
	if err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.NATS.Mode != NATSModeExternal {
		t.Fatalf("expected nats.mode %q, got %q", NATSModeExternal, cfg.NATS.Mode)
	}
	if cfg.NATS.URL != "nats://127.0.0.1:4222" {
		t.Fatalf("unexpected nats.url: %q", cfg.NATS.URL)
	}
	if cfg.NATS.Port != 4333 {
		t.Fatalf("unexpected port: %d", cfg.NATS.Port)
	}
	if cfg.Streams.Storage != "memory" {
		t.Fatalf("unexpected streams.storage: %q", cfg.Streams.Storage)
	}
	if cfg.Defaults.WorkerResponseTimeout != 45*time.Second {
		t.Fatalf("unexpected defaults.worker_response_timeout: %v", cfg.Defaults.WorkerResponseTimeout)
	}
	if cfg.Defaults.StepTimeout != 2*time.Minute {
		t.Fatalf("unexpected defaults.step_timeout: %v", cfg.Defaults.StepTimeout)
	}
}

func TestLoadConfigEnvOverrides(t *testing.T) {
	t.Setenv("grctl_NATS_PORT", "4333")
	t.Setenv("grctl_NATS_CONFIG_FILE", "/tmp/custom-nats.conf")
	t.Setenv("grctl_DEFAULTS_WORKER_RESPONSE_TIMEOUT", "7s")

	cfg, err := Load("/nonexistent/path/grctl.yaml")
	if err != nil {
		t.Fatalf("load config with env overrides: %v", err)
	}

	if cfg.NATS.Port != 4333 {
		t.Fatalf("expected env override port %d, got %d", 4333, cfg.NATS.Port)
	}
	if cfg.NATS.ConfigFile != "/tmp/custom-nats.conf" {
		t.Fatalf("expected env override nats.config_file %q, got %q", "/tmp/custom-nats.conf", cfg.NATS.ConfigFile)
	}
	if cfg.Defaults.WorkerResponseTimeout != 7*time.Second {
		t.Fatalf("expected env override defaults.worker_response_timeout 7s, got %v", cfg.Defaults.WorkerResponseTimeout)
	}
}

func TestLoadConfigEmbeddedModeInvalidPort(t *testing.T) {
	configData := []byte(`nats:
  mode: embedded
  port: 0
`)

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "grctl.yaml")
	if err := os.WriteFile(configPath, configData, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Fatal("expected validation error for invalid embedded port")
	}
}

func TestLoadConfigExternalModeIgnoresPort(t *testing.T) {
	configData := []byte(`nats:
  mode: external
  url: "nats://127.0.0.1:4222"
  port: 0
`)

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "grctl.yaml")
	if err := os.WriteFile(configPath, configData, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.NATS.Port != 0 {
		t.Fatalf("expected external mode to preserve configured port, got %d", cfg.NATS.Port)
	}
}
