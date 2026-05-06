# ddgen — Gerador de métricas customizadas para Datadog

Aplicação CLI em Go para gerar massa de métricas customizadas no Datadog
(via DogStatsD ou um proxy compatível) a partir de um arquivo JSON.

## Build

```bash
cd ddgen
go mod tidy
go build -o ddgen .
```

Se quiser instalar globalmente:

```bash
go install .
# ou
sudo mv ddgen /usr/local/bin/
```

## Uso

```bash
ddgen --file example.json
```

## Formato do arquivo de configuração

```json
{
  "datadog_proxy_url": "localhost",
  "datadog_port": 8135,
  "metrics": [
    {
      "service": "service_name",
      "name": "metric.name.test",
      "type": "count",
      "value": 1,
      "tags": ["env:prod", "sigla:xx2"],
      "quantity": 1000
    }
  ]
}
```

### Campos por métrica

| Campo          | Obrigatório | Descrição                                                      |
|----------------|-------------|----------------------------------------------------------------|
| `service`      | Não         | Adicionado como tag `service:<valor>`                          |
| `name`         | Sim         | Nome da métrica                                                |
| `type`         | Sim         | Tipo: `count`, `incr`, `decr`, `gauge`, `histogram`, `distribution`, `set`, `timing` |
| `value`        | Sim         | Valor base da métrica                                          |
| `tags`         | Não         | Lista de tags no formato `chave:valor`                         |
| `quantity`     | Sim         | Quantidade de emissões dessa métrica                           |
| `min_value`    | Não         | Limite inferior para randomização do valor                     |
| `max_value`    | Não         | Limite superior para randomização do valor                     |
| `delay_ms_max` | Não         | Teto do delay aleatório entre emissões (ms). Padrão: 10        |

Quando `min_value` e `max_value` estão presentes, cada emissão usa um valor
aleatório dentro do intervalo, ignorando `value`.

## Geração aleatória

- Cada especificação de métrica roda em sua própria goroutine (paralelismo).
- Entre cada emissão há um delay aleatório (jitter) configurável via
  `delay_ms_max` para distribuir as métricas no tempo e evitar burst.
- Para `gauge`, `histogram`, `distribution` e `timing`, é possível
  randomizar valores dentro de um intervalo (`min_value` / `max_value`).
- Para `set`, cada emissão envia um identificador aleatório distinto.

## Tipos de métricas suportados

- `count`     — contador, soma valores no intervalo de flush.
- `incr`      — incremento (count com valor 1).
- `decr`      — decremento (count com valor -1).
- `gauge`     — valor instantâneo (último vence).
- `histogram` — distribuição calculada no agente.
- `distribution` — distribuição global (calculada do lado do Datadog).
- `set`       — contagem de valores únicos.
- `timing`    — duração em milissegundos (tratada como histogram).

## Exemplo completo

Veja `example.json` no diretório do projeto para um exemplo com várias
métricas de tipos diferentes.

## Observações

- A porta padrão do DogStatsD é `8125`. Se você está usando um proxy
  customizado (como no exemplo, `8135`), ajuste o campo `datadog_port`.
- Certifique-se de que o agente Datadog (ou proxy) está rodando e
  escutando na porta configurada antes de executar.# generate-metric-data
