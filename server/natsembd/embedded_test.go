package natsembd

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunEmbeddedServerWithConfig_OverridesConfigPort(t *testing.T) {
	effectivePort := getFreePort(t)
	configPort := getFreePort(t)
	storeDir := t.TempDir()

	configData := fmt.Appendf(nil, "server_name: \"grctl-test\"\nport: %d\njetstream {\n  store_dir: %q\n}", configPort, storeDir)

	configPath := filepath.Join(t.TempDir(), "nats.conf")
	if err := os.WriteFile(configPath, configData, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	nc, js, ns, err := RunEmbeddedServerWithConfig(configPath, effectivePort)
	if err != nil {
		t.Fatalf("run embedded server: %v", err)
	}
	t.Cleanup(func() {
		nc.Close()
		ns.Shutdown()
		ns.WaitForShutdown()
	})

	if js == nil {
		t.Fatal("expected jetstream context")
	}

	addr, ok := ns.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("expected TCP address, got %T", ns.Addr())
	}
	if addr.Port != effectivePort {
		t.Fatalf("expected effective port %d, got %d", effectivePort, addr.Port)
	}
}

func TestRunEmbeddedServerWithConfig_InvalidEffectivePort(t *testing.T) {
	_, _, _, err := RunEmbeddedServerWithConfig("unused.conf", 0)
	if err == nil {
		t.Fatal("expected error for invalid effective port")
	}
}

func TestResolveEmbeddedOptions_EmptyConfigUsesDefaults(t *testing.T) {
	opts, err := resolveEmbeddedOptions("")
	if err != nil {
		t.Fatalf("resolve options: %v", err)
	}
	if !opts.JetStream {
		t.Fatal("expected JetStream to be enabled by default")
	}
	if opts.StoreDir != "./data" {
		t.Fatalf("expected default store dir ./data, got %q", opts.StoreDir)
	}
	if opts.Host != "127.0.0.1" {
		t.Fatalf("expected default host 127.0.0.1, got %q", opts.Host)
	}
}

func TestResolveEmbeddedOptions_RequiresJetStreamFromConfig(t *testing.T) {
	configData := []byte(`port: 4222`)
	configPath := filepath.Join(t.TempDir(), "nats.conf")
	if err := os.WriteFile(configPath, configData, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := resolveEmbeddedOptions(configPath)
	if err == nil {
		t.Fatal("expected error when JetStream is disabled")
	}
	if !strings.Contains(err.Error(), "JetStream must be enabled") {
		t.Fatalf("expected jetstream error, got %v", err)
	}
}

func getFreePort(t *testing.T) int {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("get free port: %v", err)
	}
	defer func() { _ = l.Close() }()

	addr, ok := l.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("expected TCP addr, got %T", l.Addr())
	}
	return addr.Port
}
