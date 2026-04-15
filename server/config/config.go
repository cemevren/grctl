package config

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/go-viper/mapstructure/v2"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

const (
	NATSModeEmbedded = "embedded"
	NATSModeExternal = "external"
)

type Config struct {
	NATS     NATSConfig     `koanf:"nats"`
	Streams  StreamsConfig  `koanf:"streams"`
	Defaults DefaultsConfig `koanf:"defaults"`
}

type NATSConfig struct {
	Mode       string `koanf:"mode"`
	URL        string `koanf:"url"`
	ConfigFile string `koanf:"config_file"`
	Port       int    `koanf:"port"`
}

type StreamsConfig struct {
	Storage string `koanf:"storage"`
}

type DefaultsConfig struct {
	WorkerResponseTimeout time.Duration `koanf:"worker_response_timeout"`
	StepTimeout           time.Duration `koanf:"step_timeout"`
}

func Load(path string) (Config, error) {
	k := koanf.New(".")
	err := k.Load(confmap.Provider(defaultConfigMap(), "."), nil)
	if err != nil {
		return Config{}, fmt.Errorf("load default config: %w", err)
	}

	if path != "" {
		err = loadConfigFile(k, path)
		if err != nil {
			return Config{}, err
		}
	}

	err = k.Load(env.Provider("grctl_", ".", envKeyMapper), nil)
	if err != nil {
		return Config{}, fmt.Errorf("load env config: %w", err)
	}

	cfg, err := unmarshalConfig(k)
	if err != nil {
		return Config{}, err
	}

	cfg = cfg.Normalized()
	err = cfg.Validate()
	if err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) Normalized() Config {
	cfg := c
	cfg.NATS.Mode = strings.ToLower(strings.TrimSpace(cfg.NATS.Mode))
	cfg.Streams.Storage = strings.ToLower(strings.TrimSpace(cfg.Streams.Storage))
	return cfg
}

func (c Config) Validate() error {
	switch c.NATS.Mode {
	case NATSModeEmbedded:
		if c.NATS.Port < 1 || c.NATS.Port > 65535 {
			return fmt.Errorf("port must be in range 1..65535 for embedded mode")
		}
	case NATSModeExternal:
		if c.NATS.URL == "" {
			return fmt.Errorf("nats.url is required for external mode")
		}
	default:
		return fmt.Errorf("nats.mode must be %q or %q", NATSModeEmbedded, NATSModeExternal)
	}

	switch c.Streams.Storage {
	case "memory", "file":
	default:
		return fmt.Errorf("streams.storage must be \"memory\" or \"file\"")
	}

	if c.Defaults.WorkerResponseTimeout <= 0 {
		return fmt.Errorf("defaults.worker_ack_timeout must be positive")
	}
	if c.Defaults.StepTimeout <= 0 {
		return fmt.Errorf("defaults.step_timeout must be positive")
	}

	return nil
}

func (c Config) InMemoryStreams() bool {
	return c.Streams.Storage == "memory"
}

func loadConfigFile(k *koanf.Koanf, path string) error {
	_, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("stat config file %q: %w", path, err)
	}

	err = k.Load(file.Provider(path), yaml.Parser())
	if err != nil {
		return fmt.Errorf("load config file %q: %w", path, err)
	}

	return nil
}

func envKeyMapper(key string) string {
	key = strings.TrimPrefix(key, "grctl_")
	key = strings.ToLower(key)
	firstUnderscore := strings.Index(key, "_")
	if firstUnderscore == -1 {
		return key
	}

	return key[:firstUnderscore] + "." + key[firstUnderscore+1:]
}

func unmarshalConfig(k *koanf.Koanf) (Config, error) {
	cfg := Config{}
	decoderConfig := &mapstructure.DecoderConfig{
		Result:           &cfg,
		TagName:          "koanf",
		WeaklyTypedInput: true,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
		),
	}

	err := k.UnmarshalWithConf("", &cfg, koanf.UnmarshalConf{DecoderConfig: decoderConfig})
	if err != nil {
		return Config{}, fmt.Errorf("unmarshal config: %w", err)
	}

	return cfg, nil
}

func LoadConfig() (*Config, error) {
	configPath := flag.String("config", "config/grctl.yaml", "Path to grctl config file")
	cfg, err := Load(*configPath)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		return nil, err
	}
	return &cfg, nil
}
