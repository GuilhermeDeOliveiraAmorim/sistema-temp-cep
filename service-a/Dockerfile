# Use uma imagem base do Go
FROM golang:1.21.5

# Crie e defina o diretório de trabalho
WORKDIR /app

# Copie o arquivo go.mod e go.sum e baixe as dependências
COPY go.mod go.sum ./
RUN go mod download

# Copie o código-fonte para o contêiner
COPY . .

# Compile o aplicativo
RUN go build -o service-a

# Defina o comando padrão
CMD ["./service-a"]

# Exponha a porta que o aplicativo vai usar
EXPOSE 8080
