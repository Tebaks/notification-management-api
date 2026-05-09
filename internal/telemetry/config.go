package telemetry

import "github.com/kelseyhightower/envconfig"

type TracerConfig struct {
	OTLPEndpoint string `envconfig:"OTEL_EXPORTER_OTLP_ENDPOINT" default:"localhost:4318"`
	ServiceName  string `envconfig:"OTEL_SERVICE_NAME" default:"notification-api"`
	Environment  string `envconfig:"OTEL_ENVIRONMENT" default:"development"`
}

func NewTracerConfig() (*TracerConfig, error) {
	var cfg TracerConfig
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
