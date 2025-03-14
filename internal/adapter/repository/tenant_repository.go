package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hugohenrick/erp-supermercado/internal/domain/tenant"
	"github.com/hugohenrick/erp-supermercado/internal/infrastructure/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Erros específicos do repositório
var (
	ErrTenantNotFound = errors.New("tenant não encontrado")
	ErrDuplicateKey   = errors.New("tenant com mesmo documento já existe")
	ErrDatabaseError  = errors.New("erro de banco de dados")
)

// PostgresTenantRepository implementa a interface tenant.Repository usando PostgreSQL
type PostgresTenantRepository struct {
	db *database.PostgresDB
}

// NewPostgresTenantRepository cria uma nova instância de PostgresTenantRepository
func NewPostgresTenantRepository(db *database.PostgresDB) *PostgresTenantRepository {
	return &PostgresTenantRepository{
		db: db,
	}
}

// Create implementa tenant.Repository.Create
func (r *PostgresTenantRepository) Create(ctx context.Context, t *tenant.Tenant) error {
	// Obter conexão
	conn, err := r.db.GetConnection(ctx) // Usamos conexão normal aqui pois este tenant ainda não existe
	if err != nil {
		return fmt.Errorf("falha ao obter conexão: %w", err)
	}
	defer conn.Release()

	// Criar schema para o tenant
	err = r.db.CreateTenantSchema(ctx, t.ID, t.Schema)
	if err != nil {
		return fmt.Errorf("falha ao criar schema: %w", err)
	}

	// Inserir o tenant
	query := `
		INSERT INTO tenants (
			id, name, document, email, phone, status, schema, plan_type, max_branches, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)
	`

	_, err = conn.Exec(ctx, query,
		t.ID,
		t.Name,
		t.Document,
		t.Email,
		t.Phone,
		string(t.Status),
		t.Schema,
		t.PlanType,
		t.MaxBranches,
		t.CreatedAt,
		t.UpdatedAt,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // Unique violation
			return ErrDuplicateKey
		}
		return fmt.Errorf("falha ao inserir tenant: %w", err)
	}

	return nil
}

// FindByID implementa tenant.Repository.FindByID
func (r *PostgresTenantRepository) FindByID(ctx context.Context, id string) (*tenant.Tenant, error) {
	// Este método é usado para validar um tenant, então usamos a conexão direta
	conn, err := r.db.GetConnection(ctx)
	if err != nil {
		return nil, fmt.Errorf("falha ao obter conexão: %w", err)
	}
	defer conn.Release()

	t := &tenant.Tenant{}

	query := `
		SELECT 
			id, name, document, email, phone, status, schema, plan_type, max_branches, created_at, updated_at
		FROM 
			tenants
		WHERE 
			id = $1
	`

	var status string
	err = conn.QueryRow(ctx, query, id).Scan(
		&t.ID,
		&t.Name,
		&t.Document,
		&t.Email,
		&t.Phone,
		&status,
		&t.Schema,
		&t.PlanType,
		&t.MaxBranches,
		&t.CreatedAt,
		&t.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTenantNotFound
		}
		return nil, fmt.Errorf("falha ao buscar tenant: %w", err)
	}

	t.Status = tenant.Status(status)

	return t, nil
}

// FindByDocument implementa tenant.Repository.FindByDocument
func (r *PostgresTenantRepository) FindByDocument(ctx context.Context, document string) (*tenant.Tenant, error) {
	conn, err := r.db.GetConnection(ctx)
	if err != nil {
		return nil, fmt.Errorf("falha ao obter conexão: %w", err)
	}
	defer conn.Release()

	t := &tenant.Tenant{}

	query := `
		SELECT 
			id, name, document, email, phone, status, schema, plan_type, max_branches, created_at, updated_at
		FROM 
			tenants
		WHERE 
			document = $1
	`

	var status string
	err = conn.QueryRow(ctx, query, document).Scan(
		&t.ID,
		&t.Name,
		&t.Document,
		&t.Email,
		&t.Phone,
		&status,
		&t.Schema,
		&t.PlanType,
		&t.MaxBranches,
		&t.CreatedAt,
		&t.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTenantNotFound
		}
		return nil, fmt.Errorf("falha ao buscar tenant por documento: %w", err)
	}

	t.Status = tenant.Status(status)

	return t, nil
}

// Update implementa tenant.Repository.Update
func (r *PostgresTenantRepository) Update(ctx context.Context, t *tenant.Tenant) error {
	conn, err := r.db.GetConnection(ctx)
	if err != nil {
		return fmt.Errorf("falha ao obter conexão: %w", err)
	}
	defer conn.Release()

	query := `
		UPDATE tenants
		SET 
			name = $1,
			email = $2,
			phone = $3,
			status = $4,
			plan_type = $5,
			max_branches = $6,
			updated_at = $7
		WHERE 
			id = $8
	`

	result, err := conn.Exec(ctx, query,
		t.Name,
		t.Email,
		t.Phone,
		string(t.Status),
		t.PlanType,
		t.MaxBranches,
		time.Now(),
		t.ID,
	)

	if err != nil {
		return fmt.Errorf("falha ao atualizar tenant: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrTenantNotFound
	}

	return nil
}

// Delete implementa tenant.Repository.Delete
func (r *PostgresTenantRepository) Delete(ctx context.Context, id string) error {
	conn, err := r.db.GetConnection(ctx)
	if err != nil {
		return fmt.Errorf("falha ao obter conexão: %w", err)
	}
	defer conn.Release()

	// Primeiro verificamos se o tenant existe
	var schema string
	err = conn.QueryRow(ctx, "SELECT schema FROM tenants WHERE id = $1", id).Scan(&schema)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrTenantNotFound
		}
		return fmt.Errorf("falha ao verificar tenant: %w", err)
	}

	// Usamos uma transação para garantir a atomicidade
	err = r.db.Transaction(ctx, func(tx pgx.Tx) error {
		// Excluir o tenant
		_, err := tx.Exec(ctx, "DELETE FROM tenants WHERE id = $1", id)
		if err != nil {
			return fmt.Errorf("falha ao excluir tenant: %w", err)
		}

		// Opcionalmente, podemos excluir o schema
		// Este é um passo perigoso e talvez seja melhor apenas marcar o tenant como inativo
		// _, err = tx.Exec(ctx, fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", schema))
		// if err != nil {
		//     return fmt.Errorf("falha ao excluir schema: %w", err)
		// }

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

// List implementa tenant.Repository.List
func (r *PostgresTenantRepository) List(ctx context.Context, limit, offset int) ([]*tenant.Tenant, error) {
	conn, err := r.db.GetConnection(ctx)
	if err != nil {
		return nil, fmt.Errorf("falha ao obter conexão: %w", err)
	}
	defer conn.Release()

	query := `
		SELECT 
			id, name, document, email, phone, status, schema, plan_type, max_branches, created_at, updated_at
		FROM 
			tenants
		ORDER BY 
			name
		LIMIT $1 OFFSET $2
	`

	rows, err := conn.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("falha ao listar tenants: %w", err)
	}
	defer rows.Close()

	var tenants []*tenant.Tenant

	for rows.Next() {
		t := &tenant.Tenant{}
		var status string

		err := rows.Scan(
			&t.ID,
			&t.Name,
			&t.Document,
			&t.Email,
			&t.Phone,
			&status,
			&t.Schema,
			&t.PlanType,
			&t.MaxBranches,
			&t.CreatedAt,
			&t.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("falha ao ler tenant: %w", err)
		}

		t.Status = tenant.Status(status)
		tenants = append(tenants, t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erro ao iterar resultados: %w", err)
	}

	return tenants, nil
}

// Count implementa tenant.Repository.Count
func (r *PostgresTenantRepository) Count(ctx context.Context) (int, error) {
	conn, err := r.db.GetConnection(ctx)
	if err != nil {
		return 0, fmt.Errorf("falha ao obter conexão: %w", err)
	}
	defer conn.Release()

	var count int
	err = conn.QueryRow(ctx, "SELECT COUNT(*) FROM tenants").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("falha ao contar tenants: %w", err)
	}

	return count, nil
}

// UpdateStatus implementa tenant.Repository.UpdateStatus
func (r *PostgresTenantRepository) UpdateStatus(ctx context.Context, id string, status tenant.Status) error {
	conn, err := r.db.GetConnection(ctx)
	if err != nil {
		return fmt.Errorf("falha ao obter conexão: %w", err)
	}
	defer conn.Release()

	query := `
		UPDATE tenants
		SET 
			status = $1,
			updated_at = $2
		WHERE 
			id = $3
	`

	result, err := conn.Exec(ctx, query, string(status), time.Now(), id)
	if err != nil {
		return fmt.Errorf("falha ao atualizar status do tenant: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrTenantNotFound
	}

	return nil
}

// Exists implementa tenant.Repository.Exists
func (r *PostgresTenantRepository) Exists(ctx context.Context, id string) (bool, error) {
	conn, err := r.db.GetConnection(ctx)
	if err != nil {
		return false, fmt.Errorf("falha ao obter conexão: %w", err)
	}
	defer conn.Release()

	var exists bool
	err = conn.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM tenants WHERE id = $1)", id).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("falha ao verificar existência do tenant: %w", err)
	}

	return exists, nil
}
