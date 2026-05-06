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

	addr := fmt.Sprintf("%s:%d", cfg.DatadogProxyURL, cfg.DatadogPort)
	log.Printf("Conectando ao Datadog em %s", addr)

	gen, err := NewGenerator(addr)
	if err != nil {
		log.Fatalf("Erro ao criar gerador: %v", err)
	}
	defer gen.Close()

	totalMetrics := 0
	for _, m := range cfg.Metrics {
		totalMetrics += m.Quantity
	}

	log.Printf("Iniciando geração: %d especificações, %d métricas no total", len(cfg.Metrics), totalMetrics)
	start := time.Now()

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

	// Garante que tudo foi enviado pelo cliente statsd antes de fechar.
	if err := gen.Flush(); err != nil {
		log.Printf("Erro no flush: %v", err)
	}

	log.Printf("Concluído em %s. %d métricas enviadas.", time.Since(start).Round(time.Millisecond), totalMetrics)
}

func printUsage() {
	fmt.Println("ddgen - Gerador de métricas customizadas para Datadog")
	fmt.Println()
	fmt.Println("Uso:")
	fmt.Println("  ddgen --file <config.json>")
	fmt.Println()
	fmt.Println("Flags:")
	flag.PrintDefaults()
}
