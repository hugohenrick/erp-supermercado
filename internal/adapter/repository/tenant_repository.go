package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hugohenrick/erp-supermercado/internal/domain/tenant"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Erros específicos do repositório
var (
	ErrTenantNotFound          = errors.New("tenant não encontrado")
	ErrTenantDuplicateDocument = errors.New("tenant com mesmo documento já existe")
	ErrTenantDatabaseError     = errors.New("erro de banco de dados")
	// ErrDuplicateKey já definido em outro lugar do pacote
)

// TenantRepository implementa a interface tenant.Repository
type TenantRepository struct {
	db *pgxpool.Pool
}

// NewTenantRepository cria uma nova instância de TenantRepository
func NewTenantRepository(db *pgxpool.Pool) tenant.Repository {
	return &TenantRepository{
		db: db,
	}
}

// Create implementa tenant.Repository.Create
func (r *TenantRepository) Create(ctx context.Context, t *tenant.Tenant) error {
	// Verificar se já existe um tenant com o mesmo documento
	var count int
	err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM tenants WHERE document = $1", t.Document).Scan(&count)
	if err != nil {
		return fmt.Errorf("erro ao verificar existência de tenant: %w", err)
	}

	if count > 0 {
		return ErrTenantDuplicateDocument
	}

	// Inserir o tenant no banco de dados
	_, err = r.db.Exec(ctx,
		`INSERT INTO tenants (id, name, document, email, phone, status, schema, plan_type, max_branches, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		t.ID, t.Name, t.Document, t.Email, t.Phone, string(t.Status), t.Schema, t.PlanType, t.MaxBranches, t.CreatedAt, t.UpdatedAt)

	if err != nil {
		// Verificar se é um erro de chave duplicada
		if strings.Contains(err.Error(), "duplicate key") {
			return ErrDuplicateKey
		}
		return fmt.Errorf("erro ao criar tenant: %w", err)
	}

	return nil
}

// FindByID implementa tenant.Repository.FindByID
func (r *TenantRepository) FindByID(ctx context.Context, id string) (*tenant.Tenant, error) {
	var t tenant.Tenant
	var status string

	err := r.db.QueryRow(ctx, `
		SELECT id, name, document, email, phone, status, schema, plan_type, max_branches, created_at, updated_at
		FROM tenants
		WHERE id = $1`,
		id).Scan(
		&t.ID, &t.Name, &t.Document, &t.Email, &t.Phone, &status, &t.Schema, &t.PlanType, &t.MaxBranches, &t.CreatedAt, &t.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTenantNotFound
		}
		return nil, fmt.Errorf("erro ao buscar tenant por ID: %w", err)
	}

	t.Status = tenant.Status(status)
	return &t, nil
}

// FindByDocument implementa tenant.Repository.FindByDocument
func (r *TenantRepository) FindByDocument(ctx context.Context, document string) (*tenant.Tenant, error) {
	var t tenant.Tenant
	var status string

	err := r.db.QueryRow(ctx, `
		SELECT id, name, document, email, phone, status, schema, plan_type, max_branches, created_at, updated_at
		FROM tenants
		WHERE document = $1`,
		document).Scan(
		&t.ID, &t.Name, &t.Document, &t.Email, &t.Phone, &status, &t.Schema, &t.PlanType, &t.MaxBranches, &t.CreatedAt, &t.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTenantNotFound
		}
		return nil, fmt.Errorf("erro ao buscar tenant por documento: %w", err)
	}

	t.Status = tenant.Status(status)
	return &t, nil
}

// List implementa tenant.Repository.List
func (r *TenantRepository) List(ctx context.Context, limit, offset int) ([]*tenant.Tenant, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, document, email, phone, status, schema, plan_type, max_branches, created_at, updated_at
		FROM tenants
		ORDER BY name
		LIMIT $1 OFFSET $2`,
		limit, offset)

	if err != nil {
		return nil, fmt.Errorf("erro ao listar tenants: %w", err)
	}
	defer rows.Close()

	var tenants []*tenant.Tenant

	for rows.Next() {
		var t tenant.Tenant
		var status string

		err := rows.Scan(&t.ID, &t.Name, &t.Document, &t.Email, &t.Phone, &status, &t.Schema, &t.PlanType, &t.MaxBranches, &t.CreatedAt, &t.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("erro ao ler tenant: %w", err)
		}

		t.Status = tenant.Status(status)
		tenants = append(tenants, &t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erro ao percorrer resultados: %w", err)
	}

	return tenants, nil
}

// Update implementa tenant.Repository.Update
func (r *TenantRepository) Update(ctx context.Context, t *tenant.Tenant) error {
	// Verificar se o tenant existe
	exists, err := r.Exists(ctx, t.ID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrTenantNotFound
	}

	// Atualizar o tenant
	_, err = r.db.Exec(ctx, `
		UPDATE tenants
		SET name = $1, email = $2, phone = $3, plan_type = $4, max_branches = $5, updated_at = $6
		WHERE id = $7`,
		t.Name, t.Email, t.Phone, t.PlanType, t.MaxBranches, time.Now(), t.ID)

	if err != nil {
		return fmt.Errorf("erro ao atualizar tenant: %w", err)
	}

	return nil
}

// Delete implementa tenant.Repository.Delete
func (r *TenantRepository) Delete(ctx context.Context, id string) error {
	// Verificar se o tenant existe
	exists, err := r.Exists(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return ErrTenantNotFound
	}

	// Apagar o tenant
	_, err = r.db.Exec(ctx, "DELETE FROM tenants WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("erro ao excluir tenant: %w", err)
	}

	return nil
}

// UpdateStatus implementa tenant.Repository.UpdateStatus
func (r *TenantRepository) UpdateStatus(ctx context.Context, id string, status tenant.Status) error {
	// Verificar se o tenant existe
	exists, err := r.Exists(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return ErrTenantNotFound
	}

	// Atualizar o status do tenant
	_, err = r.db.Exec(ctx, `
		UPDATE tenants
		SET status = $1, updated_at = $2
		WHERE id = $3`,
		string(status), time.Now(), id)

	if err != nil {
		return fmt.Errorf("erro ao atualizar status do tenant: %w", err)
	}

	return nil
}

// Count implementa tenant.Repository.Count
func (r *TenantRepository) Count(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM tenants").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("erro ao contar tenants: %w", err)
	}

	return count, nil
}

// FindByNameLike implementa tenant.Repository.FindByNameLike
func (r *TenantRepository) FindByNameLike(ctx context.Context, name string, limit, offset int) ([]*tenant.Tenant, error) {
	// Utilizar ILIKE para busca case-insensitive
	rows, err := r.db.Query(ctx, `
		SELECT id, name, document, email, phone, status, schema, plan_type, max_branches, created_at, updated_at
		FROM tenants
		WHERE name ILIKE $1
		ORDER BY name
		LIMIT $2 OFFSET $3`,
		"%"+name+"%", limit, offset)

	if err != nil {
		return nil, fmt.Errorf("erro ao buscar tenants por nome: %w", err)
	}
	defer rows.Close()

	var tenants []*tenant.Tenant

	for rows.Next() {
		var t tenant.Tenant
		var status string

		err := rows.Scan(&t.ID, &t.Name, &t.Document, &t.Email, &t.Phone, &status, &t.Schema, &t.PlanType, &t.MaxBranches, &t.CreatedAt, &t.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("erro ao ler tenant: %w", err)
		}

		t.Status = tenant.Status(status)
		tenants = append(tenants, &t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erro ao percorrer resultados: %w", err)
	}

	return tenants, nil
}

// Exists implementa tenant.Repository.Exists
func (r *TenantRepository) Exists(ctx context.Context, id string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM tenants WHERE id = $1)", id).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("erro ao verificar existência de tenant: %w", err)
	}

	return exists, nil
}

// ExistsByDocument implementa tenant.Repository.ExistsByDocument
func (r *TenantRepository) ExistsByDocument(ctx context.Context, document string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM tenants WHERE document = $1)", document).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("erro ao verificar existência de tenant por documento: %w", err)
	}

	return exists, nil
}
