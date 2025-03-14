# ERP para Supermercados

Sistema de gestão (ERP) especializado para pequenos e médios supermercados, com suporte multi-tenant e multi-filial.

## Funcionalidades Principais

- **Multi-tenant**: Suporte a múltiplas empresas no mesmo sistema
- **Multi-filial**: Gestão de múltiplas filiais por empresa
- **Gestão de Estoque**: Controle completo de produtos e inventário
- **PDV (Ponto de Venda)**: Interface intuitiva para operadores de caixa
- **Gestão de Compras**: Controle de fornecedores e pedidos
- **Gestão Financeira**: Contas a pagar/receber, fluxo de caixa
- **Gestão de Clientes**: Cadastro e programa de fidelidade
- **Business Intelligence**: Dashboards e relatórios gerenciais

## Requisitos

- Go 1.21 ou superior
- PostgreSQL 15+
- Redis (para cache)

## Instalação

1. Clone o repositório
```bash
git clone https://github.com/hugohenrick/erp-supermercado.git
cd erp-supermercado
```

2. Configure as variáveis de ambiente
```bash
cp .env.example .env
# Edite o arquivo .env com suas configurações
```

3. Execute as migrações do banco de dados
```bash
go run cmd/migration/main.go
```

4. Inicie o servidor
```bash
go run cmd/api/main.go
```

## Arquitetura

O projeto segue os princípios da Clean Architecture:

- `cmd/`: Pontos de entrada da aplicação
- `internal/domain/`: Entidades e regras de negócio
- `internal/usecase/`: Casos de uso da aplicação
- `internal/adapter/`: Adaptadores para repositórios e APIs
- `internal/infrastructure/`: Configuração de infraestrutura
- `pkg/`: Pacotes compartilhados

## Desenvolvimento

### Executando Testes
```bash
go test ./...
```

### Gerando Mocks
```bash
go generate ./...
```

## Licença

Este projeto é distribuído sob a licença MIT. 