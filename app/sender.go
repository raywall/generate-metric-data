package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// --------------------------------------------------------------------------
// Tipos da API v2 do Datadog (/api/v2/series)
// --------------------------------------------------------------------------

// ddSeriesPayload é o body enviado ao endpoint /api/v2/series.
type ddSeriesPayload struct {
	Series []ddSeries `json:"series"`
}

// ddSeries representa uma série de pontos de uma métrica.
type ddSeries struct {
	Metric    string       `json:"metric"`
	Type      int          `json:"type"` // 0=unspecified, 1=count, 2=rate, 3=gauge
	Points    []ddPoint    `json:"points"`
	Tags      []string     `json:"tags,omitempty"`
	Resources []ddResource `json:"resources,omitempty"`
}

// ddPoint é um par (timestamp, value).
type ddPoint struct {
	Timestamp int64   `json:"timestamp"`
	Value     float64 `json:"value"`
}

// ddResource permite associar a métrica a um recurso (ex: host, service).
type ddResource struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// --------------------------------------------------------------------------
// APISender
// --------------------------------------------------------------------------

// APISender envia métricas diretamente à API HTTP v2 do Datadog,
// com suporte a proxy HTTP corporativo.
type APISender struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

// NewAPISender cria um APISender configurado com a chave de API, site e
// proxy HTTP opcional (ex: "http://proxy.corp.internal:8080").
func NewAPISender(apiKey, site, httpProxy string) (*APISender, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("datadog_api_key não pode ser vazio")
	}
	if site == "" {
		site = "datadoghq.com"
	}

	transport := &http.Transport{}
	if httpProxy != "" {
		proxyURL, err := url.Parse(httpProxy)
		if err != nil {
			return nil, fmt.Errorf("http_proxy inválido %q: %w", httpProxy, err)
		}
		transport.Proxy = http.ProxyURL(proxyURL)
		log.Printf("[api] usando proxy HTTP: %s", httpProxy)
	}

	return &APISender{
		apiKey:  apiKey,
		baseURL: fmt.Sprintf("https://api.%s", site),
		client: &http.Client{
			Transport: transport,
			Timeout:   30 * time.Second,
		},
	}, nil
}

// Generate gera e envia "Quantity" métricas da especificação via API HTTP.
// Os pontos são enviados em lotes de até 500 por requisição.
func (s *APISender) Generate(m Metric) error {
	tags := buildTags(m)
	metricType := mapAPIType(m.Type)
	delayMax := m.DelayMsMax
	if delayMax <= 0 {
		delayMax = 10
	}

	log.Printf("[%s] gerando %d métricas via API (type=%s → dd_type=%d)", m.Name, m.Quantity, m.Type, metricType)

	const batchSize = 500
	batch := make([]ddPoint, 0, batchSize)

	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		series := ddSeries{
			Metric: m.Name,
			Type:   metricType,
			Points: batch,
			Tags:   tags,
		}
		if m.Service != "" {
			series.Resources = []ddResource{{Name: m.Service, Type: "service"}}
		}
		if err := s.post(ddSeriesPayload{Series: []ddSeries{series}}); err != nil {
			return err
		}
		batch = batch[:0]
		return nil
	}

	for i := 0; i < m.Quantity; i++ {
		batch = append(batch, ddPoint{
			Timestamp: time.Now().Unix(),
			Value:     computeValue(m),
		})

		if len(batch) >= batchSize {
			if err := flush(); err != nil {
				log.Printf("[%s] erro ao enviar lote: %v", m.Name, err)
			}
		}

		// Jitter entre emissões.
		time.Sleep(time.Duration(rand.Intn(delayMax)+1) * time.Millisecond)

		if (i+1)%500 == 0 {
			log.Printf("[%s] progresso: %d/%d", m.Name, i+1, m.Quantity)
		}
	}

	if err := flush(); err != nil {
		log.Printf("[%s] erro ao enviar lote final: %v", m.Name, err)
	}

	log.Printf("[%s] concluído (%d métricas)", m.Name, m.Quantity)
	return nil
}

// post serializa e envia um payload para /api/v2/series.
func (s *APISender) post(payload ddSeriesPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("serializando payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, s.baseURL+"/api/v2/series", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("criando request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("DD-API-KEY", s.apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("enviando request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("resposta inesperada do Datadog: HTTP %d", resp.StatusCode)
	}
	return nil
}

// mapAPIType converte o tipo textual para o inteiro esperado pela API v2.
//
// A API v2 só aceita count (1), rate (2) e gauge (3).
// Tipos processados pelo agent (histogram, distribution, timing, set)
// são mapeados para gauge pois não podem ser agregados server-side.
func mapAPIType(t string) int {
	switch strings.ToLower(t) {
	case "count", "incr", "decr":
		return 1 // COUNT
	case "rate":
		return 2 // RATE
	case "gauge":
		return 3 // GAUGE
	default:
		// histogram, distribution, timing, set → gauge (melhor aproximação via API)
		log.Printf("[aviso] tipo %q não é suportado nativamente pela API v2; enviando como gauge. "+
			"Para suporte completo a este tipo, use mode=agent.", t)
		return 3
	}
}
