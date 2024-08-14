# Sistema de Temperatura por CEP

Este projeto é um sistema em Go que permite a consulta de temperatura baseada em um CEP. O sistema é composto por dois serviços:

- **Serviço A**: Recebe um CEP, valida e redireciona a requisição para o Serviço B.
- **Serviço B**: Recebe um CEP válido, pesquisa a localização e retorna a temperatura atual em Celsius, Fahrenheit e Kelvin.

Além disso, o projeto utiliza OpenTelemetry e Zipkin para rastreamento distribuído.

## Pré-requisitos

- Docker
- Docker Compose

## Estrutura do Projeto

/project-root
├── service-a/
│ ├── go.mod
│ ├── go.sum
│ ├── main.go
│ ├── tracing.go
│ └── Dockerfile
├── service-b/
│ ├── go.mod
│ ├── go.sum
│ ├── main.go
│ ├── tracing.go
│ └── Dockerfile
├── docker-compose.yml
├── otel-collector-config.yaml
└── README.md

## Configuração

### 1. Clonar o Repositório

Clone o repositório do projeto para sua máquina local:

```bash
git clone <URL_DO_REPOSITORIO>
cd <DIRETORIO_DO_REPOSITORIO>

Use o Docker Compose para construir e iniciar os serviços:

docker-compose up --build

## Serviços Incluídos

- **Serviço A** na porta 8080
- **Serviço B** na porta 8081
- **Zipkin** na porta 9411
- **Otel Collector** na porta 4317 (gRPC) e 4318 (HTTP)

## Configuração do OpenTelemetry e Zipkin

O OpenTelemetry está configurado para enviar spans ao Otel Collector, que por sua vez, exporta os dados para o Zipkin.

### Executar Otel Collector

O Otel Collector será iniciado automaticamente com o Docker Compose. Você pode verificar os spans coletados acessando o Zipkin em:

Use curl para enviar uma requisição POST para o Serviço A com um CEP válido:

curl -X POST http://localhost:8080/cep \
     -H "Content-Type: application/json" \
     -d '{"cep": "29902555"}'
```
