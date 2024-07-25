- iniciar serviço a:

cd servico-a/
go run main.go

- iniciar serviço b:

cd servico-a/
go run main.go

- testar com curl:

curl -X POST -d '{"cep": "49000173"}' -H "Content-Type: application/json" http://localhost:8080/cep

- verificar traces;

docker run -d -p 9411:9411 openzipkin/zipkin
