# Serviço A

## Descrição

O Serviço A recebe um CEP via POST e valida o input. Se o CEP for válido, ele encaminha a requisição para o Serviço B.

## Como Rodar

1. **Construir a Imagem Docker**:
   ```bash
   docker build -t service-a .
   ```
