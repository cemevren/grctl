package config

func defaultConfigMap() map[string]any {
	return map[string]any{
		"nats.mode":                        NATSModeEmbedded,
		"nats.port":                        4225,
		"streams.storage":                  "file",
		"defaults.worker_response_timeout": "5s",
		"defaults.step_timeout":            "5m",
	}
}
