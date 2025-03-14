package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hugohenrick/erp-supermercado/internal/infrastructure/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	// Carregar variáveis de ambiente
	if err := godotenv.Load(); err != nil {
		log.Printf("Aviso: Arquivo .env não encontrado: %v", err)
	}

	// Criar conexão com o banco
	db, err := database.NewPostgresDB()
	if err != nil {
		log.Fatalf("Erro ao conectar com o banco de dados: %v", err)
	}

	// Executar as migrações
	if err := runMigrations(db); err != nil {
		log.Fatalf("Erro ao executar migrações: %v", err)
	}

	log.Println("Migrações executadas com sucesso!")
}

func runMigrations(db *pgxpool.Pool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	conn, err := db.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("erro ao obter conexão: %w", err)
	}
	defer conn.Release()

	// Verificar se a tabela de migrações existe
	if err := createMigrationsTable(ctx, conn); err != nil {
		return fmt.Errorf("erro ao criar tabela de migrações: %w", err)
	}

	// Verificar última migração executada
	lastMigration, err := getLastMigration(ctx, conn)
	if err != nil {
		return fmt.Errorf("erro ao verificar última migração: %w", err)
	}

	log.Printf("Última migração executada: %s", lastMigration)

	// Lista de migrações
	migrations := []migration{
		{
			version: "001_init_schema",
			up: `
				-- Tabela de tenants (empresas)
				CREATE TABLE IF NOT EXISTS tenants (
					id UUID PRIMARY KEY,
					name VARCHAR(255) NOT NULL,
					document VARCHAR(20) UNIQUE NOT NULL,
					email VARCHAR(255),
					phone VARCHAR(20),
					status VARCHAR(20) NOT NULL,
					schema VARCHAR(50) NOT NULL,
					plan_type VARCHAR(50) NOT NULL,
					max_branches INTEGER NOT NULL DEFAULT 1,
					created_at TIMESTAMP NOT NULL,
					updated_at TIMESTAMP NOT NULL
				);
				
				-- Índices
				CREATE INDEX IF NOT EXISTS idx_tenants_status ON tenants(status);
				CREATE INDEX IF NOT EXISTS idx_tenants_document ON tenants(document);
			`,
		},
		{
			version: "002_create_branches",
			up: `
				-- Tabela de filiais
				CREATE TABLE IF NOT EXISTS branches (
					id UUID PRIMARY KEY,
					tenant_id UUID NOT NULL REFERENCES tenants(id),
					name VARCHAR(255) NOT NULL,
					code VARCHAR(50) NOT NULL,
					type VARCHAR(20) NOT NULL,
					document VARCHAR(20),
					street VARCHAR(255),
					number VARCHAR(20),
					complement VARCHAR(255),
					district VARCHAR(255),
					city VARCHAR(255),
					state VARCHAR(50),
					zip_code VARCHAR(20),
					country VARCHAR(50) DEFAULT 'Brasil',
					phone VARCHAR(20),
					email VARCHAR(255),
					status VARCHAR(20) NOT NULL,
					is_main BOOLEAN NOT NULL DEFAULT false,
					created_at TIMESTAMP NOT NULL,
					updated_at TIMESTAMP NOT NULL,
					UNIQUE(tenant_id, code)
				);
				
				-- Índices
				CREATE INDEX IF NOT EXISTS idx_branches_tenant_id ON branches(tenant_id);
				CREATE INDEX IF NOT EXISTS idx_branches_status ON branches(status);
				CREATE INDEX IF NOT EXISTS idx_branches_is_main ON branches(is_main);
			`,
		},
		{
			version: "003_create_users",
			up: `
				-- Tabela de usuários
				CREATE TABLE IF NOT EXISTS users (
					id UUID PRIMARY KEY,
					tenant_id UUID NOT NULL REFERENCES tenants(id),
					branch_id UUID REFERENCES branches(id),
					name VARCHAR(255) NOT NULL,
					email VARCHAR(255) NOT NULL,
					password VARCHAR(255) NOT NULL,
					role VARCHAR(50) NOT NULL,
					status VARCHAR(20) NOT NULL,
					last_login_at TIMESTAMP,
					created_at TIMESTAMP NOT NULL,
					updated_at TIMESTAMP NOT NULL,
					UNIQUE(tenant_id, email)
				);
				
				-- Índices
				CREATE INDEX IF NOT EXISTS idx_users_tenant_id ON users(tenant_id);
				CREATE INDEX IF NOT EXISTS idx_users_branch_id ON users(branch_id);
				CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
				CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);
			`,
		},
		{
			version: "004_create_product_categories",
			up: `
				-- Tabela de categorias de produtos
				CREATE TABLE IF NOT EXISTS product_categories (
					id UUID PRIMARY KEY,
					tenant_id UUID NOT NULL REFERENCES tenants(id),
					name VARCHAR(255) NOT NULL,
					code VARCHAR(50),
					parent_id UUID REFERENCES product_categories(id),
					description TEXT,
					active BOOLEAN NOT NULL DEFAULT true,
					created_at TIMESTAMP NOT NULL,
					updated_at TIMESTAMP NOT NULL,
					UNIQUE(tenant_id, name)
				);
				
				-- Índices
				CREATE INDEX IF NOT EXISTS idx_product_categories_tenant_id ON product_categories(tenant_id);
				CREATE INDEX IF NOT EXISTS idx_product_categories_parent_id ON product_categories(parent_id);
			`,
		},
		{
			version: "005_create_products",
			up: `
				-- Tabela de produtos
				CREATE TABLE IF NOT EXISTS products (
					id UUID PRIMARY KEY,
					tenant_id UUID NOT NULL REFERENCES tenants(id),
					sku VARCHAR(50) NOT NULL,
					barcode VARCHAR(50),
					name VARCHAR(255) NOT NULL,
					description TEXT,
					category_id UUID REFERENCES product_categories(id),
					unit VARCHAR(10) NOT NULL,
					cost_price DECIMAL(15,2) NOT NULL,
					sell_price DECIMAL(15,2) NOT NULL,
					tax_rate DECIMAL(5,2) DEFAULT 0,
					min_stock DECIMAL(15,3) DEFAULT 0,
					max_stock DECIMAL(15,3) DEFAULT 0,
					weight DECIMAL(10,3),
					width DECIMAL(10,3),
					height DECIMAL(10,3),
					depth DECIMAL(10,3),
					perishable BOOLEAN DEFAULT false,
					active BOOLEAN NOT NULL DEFAULT true,
					created_at TIMESTAMP NOT NULL,
					updated_at TIMESTAMP NOT NULL,
					UNIQUE(tenant_id, sku)
				);
				
				-- Índices
				CREATE INDEX IF NOT EXISTS idx_products_tenant_id ON products(tenant_id);
				CREATE INDEX IF NOT EXISTS idx_products_category_id ON products(category_id);
				CREATE INDEX IF NOT EXISTS idx_products_barcode ON products(barcode);
				CREATE INDEX IF NOT EXISTS idx_products_active ON products(active);
			`,
		},
		{
			version: "006_create_inventory",
			up: `
				-- Tabela de estoque
				CREATE TABLE IF NOT EXISTS inventory (
					id UUID PRIMARY KEY,
					tenant_id UUID NOT NULL REFERENCES tenants(id),
					branch_id UUID NOT NULL REFERENCES branches(id),
					product_id UUID NOT NULL REFERENCES products(id),
					quantity DECIMAL(15,3) NOT NULL DEFAULT 0,
					last_counted_at TIMESTAMP,
					created_at TIMESTAMP NOT NULL,
					updated_at TIMESTAMP NOT NULL,
					UNIQUE(tenant_id, branch_id, product_id)
				);
				
				-- Índices
				CREATE INDEX IF NOT EXISTS idx_inventory_tenant_id ON inventory(tenant_id);
				CREATE INDEX IF NOT EXISTS idx_inventory_branch_id ON inventory(branch_id);
				CREATE INDEX IF NOT EXISTS idx_inventory_product_id ON inventory(product_id);
				
				-- Tabela de movimentações de estoque
				CREATE TABLE IF NOT EXISTS inventory_movements (
					id UUID PRIMARY KEY,
					tenant_id UUID NOT NULL REFERENCES tenants(id),
					branch_id UUID NOT NULL REFERENCES branches(id),
					product_id UUID NOT NULL REFERENCES products(id),
					type VARCHAR(20) NOT NULL,
					quantity DECIMAL(15,3) NOT NULL,
					previous_quantity DECIMAL(15,3) NOT NULL,
					reference_id UUID,
					reference_type VARCHAR(50),
					notes TEXT,
					created_by UUID REFERENCES users(id),
					created_at TIMESTAMP NOT NULL
				);
				
				-- Índices
				CREATE INDEX IF NOT EXISTS idx_inventory_movements_tenant_id ON inventory_movements(tenant_id);
				CREATE INDEX IF NOT EXISTS idx_inventory_movements_branch_id ON inventory_movements(branch_id);
				CREATE INDEX IF NOT EXISTS idx_inventory_movements_product_id ON inventory_movements(product_id);
				CREATE INDEX IF NOT EXISTS idx_inventory_movements_type ON inventory_movements(type);
				CREATE INDEX IF NOT EXISTS idx_inventory_movements_reference_id ON inventory_movements(reference_id);
				CREATE INDEX IF NOT EXISTS idx_inventory_movements_created_at ON inventory_movements(created_at);
			`,
		},
		{
			version: "007_create_customers",
			up: `
				-- Tabela de clientes
				CREATE TABLE IF NOT EXISTS customers (
					id UUID PRIMARY KEY,
					tenant_id UUID NOT NULL REFERENCES tenants(id),
					branch_id UUID REFERENCES branches(id),
					name VARCHAR(255) NOT NULL,
					document VARCHAR(20),
					email VARCHAR(255),
					phone VARCHAR(20),
					street VARCHAR(255),
					number VARCHAR(20),
					complement VARCHAR(255),
					district VARCHAR(255),
					city VARCHAR(255),
					state VARCHAR(50),
					zip_code VARCHAR(20),
					country VARCHAR(50) DEFAULT 'Brasil',
					birthday DATE,
					notes TEXT,
					active BOOLEAN NOT NULL DEFAULT true,
					created_at TIMESTAMP NOT NULL,
					updated_at TIMESTAMP NOT NULL
				);
				
				-- Índices
				CREATE INDEX IF NOT EXISTS idx_customers_tenant_id ON customers(tenant_id);
				CREATE INDEX IF NOT EXISTS idx_customers_branch_id ON customers(branch_id);
				CREATE INDEX IF NOT EXISTS idx_customers_document ON customers(document);
				CREATE INDEX IF NOT EXISTS idx_customers_email ON customers(email);
				CREATE INDEX IF NOT EXISTS idx_customers_active ON customers(active);
			`,
		},
		// Adicione outras migrações conforme necessário
	}

	// Executar migrações pendentes
	for _, m := range migrations {
		if m.version <= lastMigration {
			log.Printf("Pulando migração %s (já executada)", m.version)
			continue
		}

		log.Printf("Executando migração %s", m.version)

		// Iniciar transação
		tx, err := conn.Begin(ctx)
		if err != nil {
			return fmt.Errorf("erro ao iniciar transação: %w", err)
		}

		// Executar migração
		_, err = tx.Exec(ctx, m.up)
		if err != nil {
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				log.Printf("Erro ao fazer rollback: %v", rbErr)
			}
			return fmt.Errorf("erro ao executar migração %s: %w", m.version, err)
		}

		// Registrar migração executada
		_, err = tx.Exec(ctx,
			"INSERT INTO migrations (version, executed_at) VALUES ($1, $2)",
			m.version, time.Now())
		if err != nil {
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				log.Printf("Erro ao fazer rollback: %v", rbErr)
			}
			return fmt.Errorf("erro ao registrar migração %s: %w", m.version, err)
		}

		// Commit
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("erro ao fazer commit da migração %s: %w", m.version, err)
		}

		log.Printf("Migração %s executada com sucesso", m.version)
	}

	return nil
}

func createMigrationsTable(ctx context.Context, conn *pgxpool.Conn) error {
	query := `
		CREATE TABLE IF NOT EXISTS migrations (
			version VARCHAR(100) PRIMARY KEY,
			executed_at TIMESTAMP NOT NULL
		)
	`
	_, err := conn.Exec(ctx, query)
	return err
}

func getLastMigration(ctx context.Context, conn *pgxpool.Conn) (string, error) {
	var version string
	err := conn.QueryRow(ctx,
		"SELECT version FROM migrations ORDER BY executed_at DESC LIMIT 1").Scan(&version)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return version, nil
}

type migration struct {
	version string
	up      string
}
