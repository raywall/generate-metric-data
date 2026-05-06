package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config representa o arquivo JSON de configuração.
type Config struct {
	DatadogProxyURL string   `json:"datadog_proxy_url"`
	DatadogPort     int      `json:"datadog_port"`
	Metrics         []Metric `json:"metrics"`
}

// Metric representa uma especificação de métrica a ser gerada.
//
// Campos opcionais (min_value/max_value) permitem variar o valor
// dentro de um intervalo. Se não informados, usa-se "value" como
// valor base. delay_ms_max define o teto do delay aleatório entre
// emissões (em milissegundos); padrão: 10ms.
type Metric struct {
	Service  string   `json:"service"`
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Value    float64  `json:"value"`
	Tags     []string `json:"tags"`
	Quantity int      `json:"quantity"`

	// Opcionais — permitem dados aleatórios mais ricos.
	MinValue   *float64 `json:"min_value,omitempty"`
	MaxValue   *float64 `json:"max_value,omitempty"`
	DelayMsMax int      `json:"delay_ms_max,omitempty"`
}

// LoadConfig lê e valida o arquivo de configuração.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("lendo arquivo: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse do JSON: %w", err)
	}

	if cfg.DatadogProxyURL == "" {
		return nil, fmt.Errorf("campo datadog_proxy_url é obrigatório")
	}
	if cfg.DatadogPort == 0 {
		return nil, fmt.Errorf("campo datadog_port é obrigatório")
	}
	if len(cfg.Metrics) == 0 {
		return nil, fmt.Errorf("é necessário ao menos uma métrica em metrics")
	}

	for i, m := range cfg.Metrics {
		if m.Name == "" {
			return nil, fmt.Errorf("metrics[%d]: campo name é obrigatório", i)
		}
		if m.Type == "" {
			return nil, fmt.Errorf("metrics[%d]: campo type é obrigatório", i)
		}
		if m.Quantity <= 0 {
			return nil, fmt.Errorf("metrics[%d]: quantity deve ser > 0", i)
		}
	}

	return &cfg, nil
}
