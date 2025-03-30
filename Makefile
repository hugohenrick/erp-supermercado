# SuperERP - Sistema de Gestão para Supermercados
# Makefile

# Cores para output
GREEN  := $(shell tput -Txterm setaf 2)
YELLOW := $(shell tput -Txterm setaf 3)
RED    := $(shell tput -Txterm setaf 1)
NC     := $(shell tput -Txterm sgr0)

# Variáveis do projeto
APP_NAME     := erp-supermercado
APP_PATH     := ./cmd/api
MIGRATION_PATH := ./cmd/migration
MIGRATIONS_DIR := ./migrations
GO_FILES     := $(shell find . -name "*.go" -not -path "./vendor/*")
GOPATH       := $(shell go env GOPATH)
DOCKER_COMPOSE=docker-compose

# Alvos .PHONY
.PHONY: build run dev clean test test-verbose coverage lint fmt swag help migrate migrate-up migrate-down migrate-create migrate-force migrate-version docker-up docker-down docker-logs deps migrate-tenant-up migrate-tenant-down migrate-tenant-force migrate-all-tenants

# Dependências
deps: ## Instala as dependências do projeto
	@echo "${YELLOW}Instalando dependências...${NC}"
	@go mod tidy
	@echo "${GREEN}Dependências instaladas com sucesso${NC}"

# Compilação e execução
build: ## Compila a aplicação
	@echo "${YELLOW}Compilando a aplicação...${NC}"
	@go build -o bin/$(APP_NAME) ./$(APP_PATH)
	@echo "${GREEN}Aplicação compilada com sucesso em bin/$(APP_NAME)${NC}"

run: build ## Compila e executa a aplicação
	@echo "${YELLOW}Executando a aplicação...${NC}"
	@./bin/$(APP_NAME)

dev: ## Executa a aplicação com hot-reload usando Air (precisa estar instalado)
	@command -v air > /dev/null || go install github.com/cosmtrek/air@latest
	@echo "${YELLOW}Executando em modo de desenvolvimento com hot-reload...${NC}"
	@air -c .air.toml

clean: ## Remove binários compilados e arquivos temporários
	@echo "${YELLOW}Removendo binários e arquivos temporários...${NC}"
	@rm -rf bin
	@rm -rf tmp
	@echo "${GREEN}Limpeza concluída${NC}"

# Testes
test: ## Executa os testes
	@echo "${YELLOW}Executando testes...${NC}"
	@go test -race ./...

test-verbose: ## Executa os testes com saída detalhada
	@echo "${YELLOW}Executando testes com saída detalhada...${NC}"
	@go test -v -race ./...

coverage: ## Gera relatório de cobertura de testes
	@echo "${YELLOW}Gerando relatório de cobertura de testes...${NC}"
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "${GREEN}Relatório de cobertura gerado em coverage.html${NC}"

# Qualidade de código
lint: ## Executa o linter (golangci-lint precisa estar instalado)
	@command -v golangci-lint > /dev/null || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "${YELLOW}Executando linter...${NC}"
	@golangci-lint run

fmt: ## Formata o código-fonte
	@echo "${YELLOW}Formatando código...${NC}"
	@gofmt -s -w $(GO_FILES)
	@echo "${GREEN}Código formatado${NC}"

# Documentação
swag: ## Gera documentação Swagger
	@command -v $(GOPATH)/bin/swag > /dev/null || go install github.com/swaggo/swag/cmd/swag@latest
	@echo "${YELLOW}Gerando documentação Swagger...${NC}"
	@$(GOPATH)/bin/swag init -g $(APP_PATH)/main.go -o ./docs
	@echo "${GREEN}Documentação Swagger gerada em ./docs${NC}"

# Migrações
migrate: ## Executa as migrações internas usando o código Go
	@echo "${YELLOW}Executando migrações...${NC}"
	@go run $(MIGRATION_PATH)/main.go
	@echo "${GREEN}Migrações executadas com sucesso${NC}"

migrate-up: ## Executa migrações para cima usando golang-migrate (precisa estar instalado)
	@command -v migrate > /dev/null || (echo "${RED}golang-migrate não está instalado. Instale-o primeiro.${NC}" && exit 1)
	@echo "${YELLOW}Executando migrações para cima...${NC}"
	@migrate -path $(MIGRATIONS_DIR) -database $$(grep DB_URL .env | cut -d '=' -f2) up
	@echo "${GREEN}Migrações executadas com sucesso${NC}"

migrate-down: ## Reverte última migração usando golang-migrate
	@command -v migrate > /dev/null || (echo "${RED}golang-migrate não está instalado. Instale-o primeiro.${NC}" && exit 1)
	@echo "${YELLOW}Revertendo última migração...${NC}"
	@migrate -path $(MIGRATIONS_DIR) -database $$(grep DB_URL .env | cut -d '=' -f2) down 1
	@echo "${GREEN}Migração revertida com sucesso${NC}"

migrate-create: ## Cria nova migração (ex: make migrate-create name=create_users_table)
	@command -v migrate > /dev/null || (echo "${RED}golang-migrate não está instalado. Instale-o primeiro.${NC}" && exit 1)
	@echo "${YELLOW}Criando novo arquivo de migração: $(name)...${NC}"
	@migrate create -ext sql -dir $(MIGRATIONS_DIR) -seq $(name)
	@echo "${GREEN}Arquivos de migração criados${NC}"

migrate-force: ## Força versão de migração (ex: make migrate-force v=1)
	@command -v migrate > /dev/null || (echo "${RED}golang-migrate não está instalado. Instale-o primeiro.${NC}" && exit 1)
	@echo "${YELLOW}Forçando versão de migração para $(v)...${NC}"
	@migrate -path $(MIGRATIONS_DIR) -database $$(grep DB_URL .env | cut -d '=' -f2) force $(v)
	@echo "${GREEN}Versão de migração definida como $(v)${NC}"

migrate-version: ## Mostra versão atual da migração
	@command -v migrate > /dev/null || (echo "${RED}golang-migrate não está instalado. Instale-o primeiro.${NC}" && exit 1)
	@echo "${YELLOW}Verificando versão atual da migração...${NC}"
	@migrate -path $(MIGRATIONS_DIR) -database $$(grep DB_URL .env | cut -d '=' -f2) version

# Docker
docker-up: ## Inicia os serviços com Docker Compose
	@echo "${YELLOW}Iniciando serviços com Docker Compose...${NC}"
	@$(DOCKER_COMPOSE) up -d
	@echo "${GREEN}Serviços iniciados${NC}"

docker-down: ## Para os serviços do Docker Compose
	@echo "${YELLOW}Parando serviços do Docker Compose...${NC}"
	@$(DOCKER_COMPOSE) down
	@echo "${GREEN}Serviços parados${NC}"

docker-logs: ## Mostra logs dos containers
	@echo "${YELLOW}Exibindo logs dos containers...${NC}"
	@$(DOCKER_COMPOSE) logs -f

# Utilitários
setup: ## Configura o ambiente de desenvolvimento
	@echo "${YELLOW}Configurando ambiente de desenvolvimento...${NC}"
	@go mod download
	@go mod tidy
	@if [ ! -f .env ]; then cp .env.example .env; fi
	@echo "${GREEN}Ambiente configurado com sucesso${NC}"

# Ajuda
help: ## Mostra esta ajuda
	@echo "${BLUE}SuperERP - Sistema de Gestão para Supermercados${NC}"
	@echo "${BLUE}----------------------------------------${NC}"
	@echo "Utilização: make ${YELLOW}<comando>${NC}"
	@echo ""
	@echo "Comandos:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  ${YELLOW}%-20s${NC} %s\n", $$1, $$2}'

# Default
.DEFAULT_GOAL := help 

# Tenant Migration Commands
migrate-tenant-up: ## Executa migrações para cima em um schema específico usando golang-migrate
	@command -v migrate > /dev/null || (echo "${RED}golang-migrate não está instalado. Instale-o primeiro.${NC}" && exit 1)
	@echo "${YELLOW}Executando migrações para cima no schema $(schema)...${NC}"
	@PGPASSWORD=$$(grep DB_PASSWORD .env | cut -d '=' -f2) psql -h $$(grep DB_HOST .env | cut -d '=' -f2) -p $$(grep DB_PORT .env | cut -d '=' -f2) -U $$(grep DB_USER .env | cut -d '=' -f2) -d $$(grep DB_NAME .env | cut -d '=' -f2) -c "CREATE SCHEMA IF NOT EXISTS $(schema);"
	@PGPASSWORD=$$(grep DB_PASSWORD .env | cut -d '=' -f2) psql -h $$(grep DB_HOST .env | cut -d '=' -f2) -p $$(grep DB_PORT .env | cut -d '=' -f2) -U $$(grep DB_USER .env | cut -d '=' -f2) -d $$(grep DB_NAME .env | cut -d '=' -f2) -c "CREATE TABLE IF NOT EXISTS $(schema).schema_migrations (version bigint not null primary key, dirty boolean not null);"
	@migrate -path migrations/tenant -database "postgres://$$(grep DB_USER .env | cut -d '=' -f2):$$(grep DB_PASSWORD .env | cut -d '=' -f2)@$$(grep DB_HOST .env | cut -d '=' -f2):$$(grep DB_PORT .env | cut -d '=' -f2)/$$(grep DB_NAME .env | cut -d '=' -f2)?sslmode=disable&search_path=$(schema)" up
	@echo "${GREEN}Migrações executadas com sucesso${NC}"

migrate-tenant-down:
	@if [ -z "$(schema)" ]; then \
		echo "Error: schema parameter is required. Usage: make migrate-tenant-down schema=<tenant_schema>"; \
		exit 1; \
	fi
	migrate -database "$(DATABASE_URL)?search_path=$(schema)" -path migrations/tenant down

migrate-tenant-force:
	@if [ -z "$(schema)" ]; then \
		echo "Error: schema and version parameters are required. Usage: make migrate-tenant-force schema=<tenant_schema> version=<version>"; \
		exit 1; \
	fi
	@if [ -z "$(version)" ]; then \
		echo "Error: version parameter is required. Usage: make migrate-tenant-force schema=<tenant_schema> version=<version>"; \
		exit 1; \
	fi
	migrate -database "$(DATABASE_URL)?search_path=$(schema)" -path migrations/tenant force $(version)

migrate-all-tenants:
	@psql "$(DATABASE_URL)" -t -A -c "SELECT schema FROM tenants WHERE status = 'active'" | while read schema; do \
		echo "Migrating tenant schema: $$schema"; \
		migrate -database "$(DATABASE_URL)?search_path=$$schema" -path migrations/tenant up; \
	done 