package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

const (
	ModeAgent = "agent"
	ModeAPI   = "api"
)

// Config representa o arquivo JSON de configuração.
type Config struct {
	// Mode define como as métricas serão enviadas: "agent" (DogStatsD) ou "api" (HTTP).
	Mode string `json:"mode"`

	// --- Modo agent (DogStatsD) ---
	// DatadogProxyURL é o endereço do agent ou proxy DogStatsD (ex: "localhost").
	DatadogProxyURL string `json:"datadog_proxy_url,omitempty"`
	// DatadogPort é a porta UDP do DogStatsD (padrão: 8125).
	DatadogPort int `json:"datadog_port,omitempty"`

	// --- Modo api (HTTP) ---
	// DatadogAPIKey é a chave de API do Datadog para autenticação.
	DatadogAPIKey string `json:"datadog_api_key,omitempty"`
	// DatadogSite é o site do Datadog (ex: "datadoghq.com", "datadoghq.eu", "us3.datadoghq.com").
	DatadogSite string `json:"datadog_site,omitempty"`
	// HTTPProxy é o endereço do proxy HTTP corporativo (ex: "http://proxy.corp.internal:8080").
	// Quando informado, todas as chamadas à API do Datadog passarão por este proxy.
	HTTPProxy string `json:"http_proxy,omitempty"`

	Metrics []Metric `json:"metrics"`
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

	// Normaliza e valida o mode.
	cfg.Mode = strings.ToLower(strings.TrimSpace(cfg.Mode))
	if cfg.Mode == "" {
		cfg.Mode = ModeAgent // retrocompatibilidade
	}
	if cfg.Mode != ModeAgent && cfg.Mode != ModeAPI {
		return nil, fmt.Errorf("mode deve ser %q ou %q, recebido: %q", ModeAgent, ModeAPI, cfg.Mode)
	}

	// Valida campos específicos do modo.
	switch cfg.Mode {
	case ModeAgent:
		if cfg.DatadogProxyURL == "" {
			return nil, fmt.Errorf("modo agent: campo datadog_proxy_url é obrigatório")
		}
		if cfg.DatadogPort == 0 {
			return nil, fmt.Errorf("modo agent: campo datadog_port é obrigatório")
		}
	case ModeAPI:
		if cfg.DatadogAPIKey == "" {
			return nil, fmt.Errorf("modo api: campo datadog_api_key é obrigatório")
		}
		if cfg.DatadogSite == "" {
			cfg.DatadogSite = "datadoghq.com"
		}
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
