package natsembd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

const embeddedServerReadyTimeout = 15 * time.Second

func RunEmbeddedServer(storeDir string) (*nats.Conn, jetstream.JetStream, *server.Server, error) {
	opts := &server.Options{
		ServerName: "grctl-embedded",
		Port:       server.RANDOM_PORT,
		JetStream:  true,
		StoreDir:   storeDir,
		NoSigs:     true,
		NoLog:      true,
	}
	return runEmbeddedServer(opts)
}

func RunEmbeddedServerWithConfig(configFile string, effectivePort int) (*nats.Conn, jetstream.JetStream, *server.Server, error) {
	if effectivePort < 1 || effectivePort > 65535 {
		return nil, nil, nil, fmt.Errorf("effective embedded port must be in range 1..65535, got %d", effectivePort)
	}
	opts, err := resolveEmbeddedOptions(configFile)
	if err != nil {
		return nil, nil, nil, err
	}
	// grctl port config is authoritative for embedded mode.
	opts.Port = effectivePort
	opts.NoSigs = true
	return runEmbeddedServer(opts)
}

func resolveEmbeddedOptions(configFile string) (*server.Options, error) {
	if strings.TrimSpace(configFile) == "" {
		return &server.Options{
			ServerName: "grctl-embedded",
			Host:       "127.0.0.1",
			JetStream:  true,
			StoreDir:   "./data",
			NoLog:      true,
		}, nil
	}

	opts, err := server.ProcessConfigFile(configFile)
	if err != nil {
		return nil, err
	}
	if !opts.JetStream {
		return nil, errors.New("JetStream must be enabled in NATS config")
	}
	return opts, nil
}

func NewJetStreamContext(nc *nats.Conn) (jetstream.JetStream, error) {
	js, err := jetstream.New(nc)
	if err != nil {
		return nil, err
	}

	maxRetries := 10
	retryDelay := 500 * time.Millisecond
	for i := range maxRetries {
		_, err = js.AccountInfo(context.Background())
		if err == nil {
			return js, nil
		}
		if i == maxRetries-1 {
			return nil, fmt.Errorf("jetstream not ready after %d retries: %w", maxRetries, err)
		}
		time.Sleep(retryDelay)
	}

	return js, nil
}

func runEmbeddedServer(opts *server.Options) (*nats.Conn, jetstream.JetStream, *server.Server, error) {
	slog.Info("Embedded NATS starting", "port", opts.Port, "server_name", opts.ServerName)
	ns, err := server.NewServer(opts)
	if err != nil {
		return nil, nil, nil, err
	}

	go ns.Start()

	if !ns.ReadyForConnections(embeddedServerReadyTimeout) {
		running := ns.Running()
		clientURL := ns.ClientURL()
		ns.Shutdown()
		return nil, nil, nil, fmt.Errorf("NATS server timeout after %s (running=%t client_url=%q)", embeddedServerReadyTimeout, running, clientURL)
	}

	clientOpts := []nats.Option{nats.InProcessServer(ns)}
	nc, err := nats.Connect(ns.ClientURL(), clientOpts...)
	if err != nil {
		ns.Shutdown()
		return nil, nil, nil, err
	}

	js, err := NewJetStreamContext(nc)
	if err != nil {
		nc.Close()
		ns.Shutdown()
		return nil, nil, nil, err
	}

	return nc, js, ns, nil
}
