# ddgen — Gerador de métricas customizadas para Datadog

Aplicação CLI em Go para gerar massa de métricas customizadas no Datadog
a partir de um arquivo JSON de configuração.

Suporta dois modos de envio:

| Modo | Protocolo | Requer Agent? | Suporta proxy HTTP? |
|------|-----------|:---:|:---:|
| `agent` | UDP / DogStatsD | ✅ Sim | ❌ (UDP local) |
| `api` | HTTPS / API v2 | ❌ Não | ✅ Sim |

---

## Build

```bash
make build
# binário gerado em: dist/ddgen
```

Instalação global opcional:

```bash
sudo mv dist/ddgen /usr/local/bin/
```

---

## Uso

```bash
ddgen --file <config.json>
```

Ou via Makefile:

```bash
make run        # executa com app/sample.json     (modo agent)
make run-api    # executa com app/sample_api.json  (modo api)
```

---

## Modos de envio

### Modo `agent` (DogStatsD — UDP)

O ddgen envia as métricas via protocolo StatsD para um Datadog Agent
rodando localmente (ou acessível na rede). O Agent é responsável por
agregar e encaminhar ao Datadog via HTTPS.

```
[ddgen] --UDP/StatsD--> [Datadog Agent] --HTTPS via proxy--> [Datadog Cloud]
```

**Quando usar:** ambientes onde o Agent já está instalado (hosts, VMs,
Kubernetes) ou quando você precisar de tipos avançados como `histogram`,
`distribution` e `set`, que requerem processamento no Agent.

**Campos obrigatórios no JSON:**

```json
{
  "mode": "agent",
  "datadog_proxy_url": "localhost",
  "datadog_port": 8125
}
```

---

### Modo `api` (HTTP — API v2 direta)

O ddgen envia os pontos diretamente ao endpoint `/api/v2/series` do
Datadog, sem necessidade de Agent. Ideal para ambientes corporativos com
proxy HTTP configurado.

```
[ddgen] --HTTPS via proxy HTTP--> [Datadog Cloud]
```

**Quando usar:** ambientes onde instalar o Agent não é viável, ou quando
você precisa enviar métricas a partir de uma máquina sem Agent.

**Campos obrigatórios no JSON:**

```json
{
  "mode": "api",
  "datadog_api_key": "SUA_API_KEY",
  "datadog_site": "datadoghq.com",
  "http_proxy": "http://proxy.corp.internal:8080"
}
```

> **Atenção — tipos suportados pela API v2:**
> A API HTTP nativa aceita apenas `count`, `rate` e `gauge`.
> Tipos que dependem de processamento no Agent (`histogram`, `distribution`,
> `timing`, `set`) são automaticamente enviados como `gauge` com um aviso
> no log. Para suporte completo a esses tipos, use `mode: agent`.

---

## Formato do arquivo de configuração

### Campos raiz

| Campo | Obrigatório | Descrição |
|---|---|---|
| `mode` | Sim | `"agent"` ou `"api"` |
| `datadog_proxy_url` | Modo agent | Endereço do Agent/proxy DogStatsD |
| `datadog_port` | Modo agent | Porta UDP do DogStatsD (padrão: `8125`) |
| `datadog_api_key` | Modo api | Chave de API do Datadog |
| `datadog_site` | Modo api | Site Datadog (padrão: `"datadoghq.com"`) |
| `http_proxy` | Não | Proxy HTTP corporativo (ex: `"http://proxy:8080"`) |

### Campos por métrica

| Campo | Obrigatório | Descrição |
|---|---|---|
| `service` | Não | Adicionado como tag `service:<valor>` |
| `name` | Sim | Nome da métrica |
| `type` | Sim | Tipo (ver tabela abaixo) |
| `value` | Sim | Valor base da métrica |
| `tags` | Não | Lista de tags no formato `chave:valor` |
| `quantity` | Sim | Quantidade de emissões |
| `min_value` | Não | Limite inferior para randomização do valor |
| `max_value` | Não | Limite superior para randomização do valor |
| `delay_ms_max` | Não | Teto do jitter entre emissões (ms). Padrão: `10` |

### Tipos de métricas suportados

| Tipo | Modo agent | Modo api | Descrição |
|---|:---:|:---:|---|
| `count` | ✅ | ✅ | Contador — soma valores no intervalo de flush |
| `incr` | ✅ | ✅ | Incremento (count +1) |
| `decr` | ✅ | ✅ | Decremento (count -1) |
| `rate` | ✅ | ✅ | Taxa por segundo |
| `gauge` | ✅ | ✅ | Valor instantâneo (último vence) |
| `histogram` | ✅ | ⚠️ gauge | Distribuição calculada no Agent |
| `distribution` | ✅ | ⚠️ gauge | Distribuição global (server-side) |
| `timing` | ✅ | ⚠️ gauge | Duração em ms (tratada como histogram) |
| `set` | ✅ | ⚠️ gauge | Contagem de valores únicos |

⚠️ = enviado como `gauge` com aviso no log quando em modo `api`.

---

## Datadog Agent com Docker

Para usar o modo `agent`, o Datadog Agent precisa estar acessível.
Use o `docker-compose.yml` incluso para subir um container rapidamente.

### Variáveis de ambiente necessárias

Crie um arquivo `.env` na raiz do projeto:

```env
# Obrigatório
DD_API_KEY=sua_chave_de_api_aqui

# Opcional — site do Datadog (padrão: datadoghq.com)
DD_SITE=datadoghq.com

# Opcional — proxy HTTP corporativo
DD_PROXY_URL=http://proxy.corp.internal:8080
```

Ou exporte direto no shell:

```bash
export DD_API_KEY=sua_chave
export DD_PROXY_URL=http://proxy.corp.internal:8080
```

### Subir o Agent

```bash
make start
```

Verifica se `DD_API_KEY` está definida, sobe o container em background
e exibe o status. O Agent fica escutando na porta `8125/UDP`.

### Parar e remover o Agent

```bash
make stop
```

### Verificar o status do Agent

```bash
docker exec datadog-agent agent status
```

### Verificar logs do Agent (diagnóstico de proxy)

```bash
docker logs datadog-agent | grep -i "proxy\|error\|failed"
```

---

## Exemplos de arquivos de configuração

**`app/sample.json`** — modo agent (DogStatsD):

```json
{
  "mode": "agent",
  "datadog_proxy_url": "localhost",
  "datadog_port": 8125,
  "metrics": [...]
}
```

**`app/sample_api.json`** — modo api (HTTP direto):

```json
{
  "mode": "api",
  "datadog_api_key": "YOUR_DD_API_KEY_HERE",
  "datadog_site": "datadoghq.com",
  "http_proxy": "http://proxy.corp.internal:8080",
  "metrics": [...]
}
```

---

## Arquitetura do projeto

```
generate-metric-data/
├── app/
│   ├── main.go          # Entrypoint — roteia para agent ou API
│   ├── config.go        # Leitura e validação do JSON de configuração
│   ├── generator.go     # Sender via DogStatsD (modo agent)
│   ├── sender_api.go    # Sender via API HTTP v2 (modo api)
│   ├── sample.json      # Exemplo — modo agent
│   └── sample_api.json  # Exemplo — modo api
├── dist/                # Binário compilado (gerado por make build)
├── docker-compose.yml   # Container do Datadog Agent
├── Makefile
└── README.md
```

---

## Referências

- [Datadog Metrics API v2](https://docs.datadoghq.com/api/latest/metrics/#submit-metrics)
- [DogStatsD](https://docs.datadoghq.com/developers/dogstatsd/)
- [Datadog Agent — configuração de proxy](https://docs.datadoghq.com/agent/configuration/proxy/)