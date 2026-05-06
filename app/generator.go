package main

import (
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/DataDog/datadog-go/v5/statsd"
)

// Generator encapsula o cliente statsd e a lógica de envio.
type Generator struct {
	client *statsd.Client
}

// NewGenerator cria um cliente statsd apontando para o endereço fornecido.
func NewGenerator(addr string) (*Generator, error) {
	client, err := statsd.New(addr,
		statsd.WithMaxMessagesPerPayload(40),
	)
	if err != nil {
		return nil, fmt.Errorf("criando cliente statsd: %w", err)
	}
	return &Generator{client: client}, nil
}

// Close fecha o cliente statsd, garantindo o flush das métricas.
func (g *Generator) Close() error {
	return g.client.Close()
}

// Flush força o envio de qualquer métrica em buffer.
func (g *Generator) Flush() error {
	return g.client.Flush()
}

// Generate envia "Quantity" métricas conforme a especificação.
func (g *Generator) Generate(m Metric) error {
	tags := buildTags(m)
	delayMax := m.DelayMsMax
	if delayMax <= 0 {
		delayMax = 10 // jitter padrão de até 10ms
	}

	log.Printf("[%s] gerando %d métricas (type=%s)", m.Name, m.Quantity, m.Type)

	for i := 0; i < m.Quantity; i++ {
		value := computeValue(m)
		if err := g.send(m, value, tags); err != nil {
			log.Printf("[%s] erro ao enviar: %v", m.Name, err)
		}

		// Delay aleatório para distribuir as emissões no tempo.
		time.Sleep(time.Duration(rand.Intn(delayMax)+1) * time.Millisecond)

		if (i+1)%500 == 0 {
			log.Printf("[%s] progresso: %d/%d", m.Name, i+1, m.Quantity)
		}
	}

	log.Printf("[%s] concluído (%d métricas)", m.Name, m.Quantity)
	return nil
}

// buildTags monta a lista final de tags adicionando "service:" automaticamente.
func buildTags(m Metric) []string {
	tags := make([]string, 0, len(m.Tags)+1)
	tags = append(tags, m.Tags...)
	if m.Service != "" {
		tags = append(tags, "service:"+m.Service)
	}
	return tags
}

// computeValue retorna o valor a ser enviado, aplicando aleatoriedade
// se min_value/max_value foram informados.
func computeValue(m Metric) float64 {
	if m.MinValue != nil && m.MaxValue != nil && *m.MaxValue > *m.MinValue {
		return *m.MinValue + rand.Float64()*(*m.MaxValue-*m.MinValue)
	}
	return m.Value
}

// send despacha a métrica conforme o tipo.
func (g *Generator) send(m Metric, value float64, tags []string) error {
	switch strings.ToLower(m.Type) {
	case "count":
		return g.client.Count(m.Name, int64(value), tags, 1)
	case "incr":
		return g.client.Incr(m.Name, tags, 1)
	case "decr":
		return g.client.Decr(m.Name, tags, 1)
	case "gauge":
		return g.client.Gauge(m.Name, value, tags, 1)
	case "histogram":
		return g.client.Histogram(m.Name, value, tags, 1)
	case "distribution":
		return g.client.Distribution(m.Name, value, tags, 1)
	case "set":
		return g.client.Set(m.Name, fmt.Sprintf("%d", rand.Int63()), tags, 1)
	case "timing":
		return g.client.Timing(m.Name, time.Duration(value)*time.Millisecond, tags, 1)
	default:
		return fmt.Errorf("tipo de métrica desconhecido: %s", m.Type)
	}
}
