package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RunTenantMigrations aplica as migrações em um schema específico de tenant
func RunTenantMigrations(schema string) error {
	// Primeiro, criar o schema se não existir
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		// Construir a URL do banco a partir das variáveis individuais
		dbURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
			os.Getenv("DB_USER"),
			os.Getenv("DB_PASSWORD"),
			os.Getenv("DB_HOST"),
			os.Getenv("DB_PORT"),
			os.Getenv("DB_NAME"),
			os.Getenv("DB_SSL_MODE"),
		)
	}

	// Conectar ao banco para criar o schema
	db, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		return fmt.Errorf("erro ao conectar ao banco: %v", err)
	}
	defer db.Close()

	// Criar o schema e configurar permissões
	if _, err := db.Exec(context.Background(), fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", schema)); err != nil {
		return fmt.Errorf("erro ao criar schema: %v", err)
	}

	// Configurar o search_path para incluir o schema
	if _, err := db.Exec(context.Background(), fmt.Sprintf("SET search_path TO %s, public", schema)); err != nil {
		return fmt.Errorf("erro ao configurar search_path: %v", err)
	}

	// Configurar URL para as migrações incluindo o schema
	dbURL = fmt.Sprintf("%s&search_path=%s,public", dbURL, schema)

	// Caminho para as migrações do tenant
	migrationsPath := filepath.Join("migrations", "tenant")
	sourceURL := fmt.Sprintf("file://%s", migrationsPath)

	// Criar instância do migrate
	m, err := migrate.New(sourceURL, dbURL)
	if err != nil {
		return fmt.Errorf("erro ao criar migrate: %v", err)
	}
	defer m.Close()

	// Forçar a versão para 0 para garantir que todas as migrações sejam aplicadas
	if err := m.Force(0); err != nil {
		return fmt.Errorf("erro ao forçar versão inicial: %v", err)
	}

	// Aplicar migrações
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("erro ao aplicar migrações: %v", err)
	}

	log.Printf("Migrações aplicadas com sucesso no schema %s", schema)
	return nil
}
