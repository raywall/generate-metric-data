.PHONY: build test run run-api start stop

# -----------------------------------------------------------------
# Build
# -----------------------------------------------------------------

## build: compila o binário em dist/ddgen
build:
	@cd app; \
	 go mod tidy; \
	 mkdir -p ../dist; \
	 go build -o ../dist/ddgen .
	@echo "Binário gerado em dist/ddgen"

# -----------------------------------------------------------------
# Testes
# -----------------------------------------------------------------

## test: executa os testes da aplicação
test:
	@cd app; \
	 go mod tidy; \
	 go test ./...

# -----------------------------------------------------------------
# Execução — modo agent (DogStatsD)
# -----------------------------------------------------------------

## run: executa a aplicação com o sample.json (modo agent)
run:
	@cd app; \
	 go run . --file sample.json

# -----------------------------------------------------------------
# Execução — modo API (HTTP direto ao Datadog)
# -----------------------------------------------------------------

## run-api: executa a aplicação com o sample_api.json (modo api)
run-api:
	@cd app; \
	 go run . --file sample_api.json

# -----------------------------------------------------------------
# Agent Docker
# -----------------------------------------------------------------

## start: sobe o container do Datadog Agent em background
##        Requer: DD_API_KEY e (opcional) DD_PROXY_URL no ambiente ou .env
start:
	@if [ -z "$$DD_API_KEY" ] && [ ! -f .env ]; then \
	  echo "ERRO: variável DD_API_KEY não definida."; \
	  echo "  Exporte-a no shell:  export DD_API_KEY=<sua_chave>"; \
	  echo "  Ou crie um arquivo .env com:  DD_API_KEY=<sua_chave>"; \
	  exit 1; \
	fi
	docker compose up -d
	@echo "Aguardando o agent ficar saudável..."
	@sleep 5
	@docker compose ps

## stop: para e remove o container do Datadog Agent
stop:
	docker compose down
	@echo "Container do Datadog Agent removido."