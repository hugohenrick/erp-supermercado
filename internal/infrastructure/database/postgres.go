package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/hugohenrick/erp-supermercado/pkg/tenant"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresConfig contém as configurações para conexão com o PostgreSQL
type PostgresConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	Database        string
	SSLMode         string
	MaxConnections  int32
	MinConnections  int32
	MaxConnLifetime time.Duration
}

// NewPostgresConfigFromEnv cria uma nova configuração a partir de variáveis de ambiente
func NewPostgresConfigFromEnv() *PostgresConfig {
	port, _ := strconv.Atoi(getEnv("DB_PORT", "5432"))
	maxConns, _ := strconv.Atoi(getEnv("DB_MAX_CONNECTIONS", "10"))
	minConns, _ := strconv.Atoi(getEnv("DB_MIN_CONNECTIONS", "2"))
	maxLifetime, _ := strconv.Atoi(getEnv("DB_MAX_LIFETIME", "300"))

	return &PostgresConfig{
		Host:            getEnv("DB_HOST", "localhost"),
		Port:            port,
		User:            getEnv("DB_USER", "postgres"),
		Password:        getEnv("DB_PASSWORD", "postgres"),
		Database:        getEnv("DB_NAME", "erp_supermercado"),
		SSLMode:         getEnv("DB_SSL_MODE", "disable"),
		MaxConnections:  int32(maxConns),
		MinConnections:  int32(minConns),
		MaxConnLifetime: time.Duration(maxLifetime) * time.Second,
	}
}

// ConnectionString retorna a string de conexão para o PostgreSQL
func (c *PostgresConfig) ConnectionString() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Database, c.SSLMode,
	)
}

// PostgresDB gerencia a conexão com o PostgreSQL
type PostgresDB struct {
	pool   *pgxpool.Pool
	config *PostgresConfig
}

// NewPostgresDB cria uma nova instância de PostgresDB
func NewPostgresDB(config *PostgresConfig) (*PostgresDB, error) {
	connString := config.ConnectionString()

	// Configuração do pool de conexões
	poolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("erro ao parsear configuração do pool: %w", err)
	}

	// Configuração de limites de conexão
	poolConfig.MaxConns = config.MaxConnections
	poolConfig.MinConns = config.MinConnections
	poolConfig.MaxConnLifetime = config.MaxConnLifetime

	// Criação do pool
	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar pool de conexões: %w", err)
	}

	// Teste de conexão
	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("erro ao testar conexão com banco: %w", err)
	}

	return &PostgresDB{
		pool:   pool,
		config: config,
	}, nil
}

// GetConnection retorna uma conexão do pool para uso
func (db *PostgresDB) GetConnection(ctx context.Context) (*pgxpool.Conn, error) {
	return db.pool.Acquire(ctx)
}

// GetTenantConnection retorna uma conexão configurada para o tenant específico
func (db *PostgresDB) GetTenantConnection(ctx context.Context) (*pgxpool.Conn, error) {
	// Adquirir conexão do pool
	conn, err := db.pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("erro ao adquirir conexão do pool: %w", err)
	}

	// Verificar se há um tenant no contexto
	tenantID := tenant.GetTenantID(ctx)
	if tenantID == "" {
		// Se não houver tenant, usa o schema public
		_, err = conn.Exec(ctx, "SET search_path TO public")
		if err != nil {
			conn.Release()
			return nil, fmt.Errorf("erro ao definir schema public: %w", err)
		}
		return conn, nil
	}

	// Buscar informações do tenant para obter o schema
	var schema string
	err = conn.QueryRow(ctx,
		"SELECT schema FROM tenants WHERE id = $1",
		tenantID).Scan(&schema)

	if err != nil {
		conn.Release()
		return nil, fmt.Errorf("erro ao buscar schema do tenant: %w", err)
	}

	// Configurar a conexão para usar o schema do tenant
	_, err = conn.Exec(ctx, fmt.Sprintf("SET search_path TO %s, public", schema))
	if err != nil {
		conn.Release()
		return nil, fmt.Errorf("erro ao definir schema do tenant: %w", err)
	}

	return conn, nil
}

// CreateTenantSchema cria um novo schema para o tenant
func (db *PostgresDB) CreateTenantSchema(ctx context.Context, tenantID, schema string) error {
	conn, err := db.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("erro ao adquirir conexão do pool: %w", err)
	}
	defer conn.Release()

	// Criar schema
	_, err = conn.Exec(ctx, fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", schema))
	if err != nil {
		return fmt.Errorf("erro ao criar schema: %w", err)
	}

	// Configurar permissões
	_, err = conn.Exec(ctx, fmt.Sprintf("GRANT ALL ON SCHEMA %s TO %s", schema, db.config.User))
	if err != nil {
		return fmt.Errorf("erro ao configurar permissões do schema: %w", err)
	}

	return nil
}

// Close fecha o pool de conexões
func (db *PostgresDB) Close() {
	if db.pool != nil {
		db.pool.Close()
	}
}

// Transaction executa uma função dentro de uma transação
func (db *PostgresDB) Transaction(ctx context.Context, txFunc func(tx pgx.Tx) error) error {
	// Adquirir conexão do pool
	conn, err := db.GetTenantConnection(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	// Iniciar transação
	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("erro ao iniciar transação: %w", err)
	}

	// Executar função dentro da transação
	if err := txFunc(tx); err != nil {
		// Rollback em caso de erro
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			log.Printf("erro ao fazer rollback: %v", rbErr)
		}
		return err
	}

	// Commit se tudo ocorreu bem
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("erro ao fazer commit: %w", err)
	}

	return nil
}

// getEnv retorna o valor de uma variável de ambiente ou um valor padrão
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
