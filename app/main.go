package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

func main() {
	var (
		configPath string
		showHelp   bool
	)

	flag.StringVar(&configPath, "file", "", "Caminho do arquivo JSON de configuração (obrigatório)")
	flag.BoolVar(&showHelp, "help", false, "Exibe ajuda")
	flag.Parse()

	if showHelp {
		printUsage()
		return
	}

	if configPath == "" {
		printUsage()
		os.Exit(1)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Erro ao carregar configuração: %v", err)
	}

	totalMetrics := 0
	for _, m := range cfg.Metrics {
		totalMetrics += m.Quantity
	}

	log.Printf("Modo de envio: %s", cfg.Mode)
	log.Printf("Iniciando geração: %d especificações, %d métricas no total", len(cfg.Metrics), totalMetrics)
	start := time.Now()

	switch cfg.Mode {
	case ModeAgent:
		runAgent(cfg)
	case ModeAPI:
		runAPI(cfg)
	}

	log.Printf("Concluído em %s. %d métricas enviadas.", time.Since(start).Round(time.Millisecond), totalMetrics)
}

// runAgent envia as métricas via DogStatsD (requer Datadog Agent rodando).
func runAgent(cfg *Config) {
	addr := fmt.Sprintf("%s:%d", cfg.DatadogProxyURL, cfg.DatadogPort)
	log.Printf("Conectando ao Datadog Agent em %s (UDP/StatsD)", addr)

	gen, err := NewGenerator(addr)
	if err != nil {
		log.Fatalf("Erro ao criar gerador StatsD: %v", err)
	}
	defer gen.Close()

	var wg sync.WaitGroup
	for _, m := range cfg.Metrics {
		wg.Add(1)
		go func(m Metric) {
			defer wg.Done()
			if err := gen.Generate(m); err != nil {
				log.Printf("[%s] erro: %v", m.Name, err)
			}
		}(m)
	}
	wg.Wait()

	if err := gen.Flush(); err != nil {
		log.Printf("Erro no flush StatsD: %v", err)
	}
}

// runAPI envia as métricas diretamente à API HTTP v2 do Datadog.
func runAPI(cfg *Config) {
	log.Printf("Conectando à API Datadog: https://api.%s (proxy: %q)", cfg.DatadogSite, cfg.HTTPProxy)

	sender, err := NewAPISender(cfg.DatadogAPIKey, cfg.DatadogSite, cfg.HTTPProxy)
	if err != nil {
		log.Fatalf("Erro ao criar API sender: %v", err)
	}

	var wg sync.WaitGroup
	for _, m := range cfg.Metrics {
		wg.Add(1)
		go func(m Metric) {
			defer wg.Done()
			if err := sender.Generate(m); err != nil {
				log.Printf("[%s] erro: %v", m.Name, err)
			}
		}(m)
	}
	wg.Wait()
}

func printUsage() {
	fmt.Println("ddgen - Gerador de métricas customizadas para Datadog")
	fmt.Println()
	fmt.Println("Uso:")
	fmt.Println("  ddgen --file <config.json>")
	fmt.Println()
	fmt.Println("Modos disponíveis (campo \"mode\" no JSON):")
	fmt.Println("  agent  — envia via DogStatsD para um Datadog Agent local (UDP)")
	fmt.Println("  api    — envia diretamente à API HTTP v2 do Datadog (com suporte a proxy)")
	fmt.Println()
	fmt.Println("Flags:")
	flag.PrintDefaults()
}
