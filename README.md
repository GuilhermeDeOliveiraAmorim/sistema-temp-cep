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
│ ├── main.go
│ ├── tracing.go
│ └── Dockerfile
├── service-b/
│ ├── main.go
│ ├── tracing.go
│ └── Dockerfile
├── docker-compose.yml
└── README.md

## Configuração

### 1. Clonar o Repositório

Clone o repositório do projeto para sua máquina local:

```bash
git clone <URL_DO_REPOSITORIO>
cd <DIRETORIO_DO_REPOSITORIO>

Use o Docker Compose para construir e iniciar os serviços:

docker-compose up --build

Isso iniciará três serviços:

Serviço A na porta 8080
Serviço B na porta 8081
Zipkin na porta 9411

Acesse o Zipkin no navegador para garantir que ele está rodando corretamente:

http://localhost:9411

Use curl para enviar uma requisição POST para o Serviço A com um CEP válido:

curl -X POST http://localhost:8080/cep \
     -H "Content-Type: application/json" \
     -d '{"cep": "29902555"}'
```
