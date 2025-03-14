package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hugohenrick/erp-supermercado/internal/domain/branch"
	pkgtenant "github.com/hugohenrick/erp-supermercado/pkg/tenant"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Erros específicos do repositório de branches
var (
	ErrBranchNotFound      = errors.New("filial não encontrada")
	ErrBranchDuplicateKey  = errors.New("filial com mesmo código já existe para este tenant")
	ErrBranchNotAllowed    = errors.New("operação não permitida para esta filial")
	ErrBranchLimitExceeded = errors.New("limite de filiais excedido para este tenant")
	ErrDuplicateKey        = errors.New("registro duplicado")
)

// BranchRepository implementa a interface branch.Repository
type BranchRepository struct {
	db *pgxpool.Pool
}

// NewBranchRepository cria uma nova instância de BranchRepository
func NewBranchRepository(db *pgxpool.Pool) branch.Repository {
	return &BranchRepository{
		db: db,
	}
}

// Create implementa branch.Repository.Create
func (r *BranchRepository) Create(ctx context.Context, b *branch.Branch) error {
	// Verificar se o tenant existe e está ativo
	exists, err := r.tenantExists(ctx, b.TenantID)
	if err != nil {
		return err
	}
	if !exists {
		return pkgtenant.ErrTenantNotFound
	}

	// Verificar se já existe uma filial principal
	if b.IsMain {
		hasMain, err := r.hasMainBranch(ctx, b.TenantID)
		if err != nil {
			return err
		}
		if hasMain {
			return errors.New("já existe uma filial principal para este tenant")
		}
	}

	// Verificar limite de filiais
	count, err := r.CountByTenant(ctx, b.TenantID)
	if err != nil {
		return err
	}

	// Consultar o limite de filiais do tenant (simplificado)
	var maxBranches int
	err = r.db.QueryRow(ctx, "SELECT max_branches FROM tenants WHERE id = $1", b.TenantID).Scan(&maxBranches)
	if err != nil {
		return fmt.Errorf("erro ao consultar limite de filiais: %w", err)
	}

	if count >= maxBranches {
		return ErrBranchLimitExceeded
	}

	// Inserir a filial
	_, err = r.db.Exec(ctx,
		`INSERT INTO branches (id, tenant_id, name, code, type, document, street, number, complement, district, city, state, zip_code, country, phone, email, status, is_main, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)`,
		b.ID, b.TenantID, b.Name, b.Code, b.Type, b.Document,
		b.Address.Street, b.Address.Number, b.Address.Complement, b.Address.District,
		b.Address.City, b.Address.State, b.Address.ZipCode, b.Address.Country,
		b.Phone, b.Email, string(b.Status), b.IsMain, b.CreatedAt, b.UpdatedAt)

	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return ErrBranchDuplicateKey
		}
		return fmt.Errorf("erro ao criar filial: %w", err)
	}

	return nil
}

// FindByID implementa branch.Repository.FindByID
func (r *BranchRepository) FindByID(ctx context.Context, id string) (*branch.Branch, error) {
	var b branch.Branch
	var addr branch.Address

	err := r.db.QueryRow(ctx,
		`SELECT id, tenant_id, name, code, type, document, street, number, complement, district, city, state, zip_code, country, phone, email, status, is_main, created_at, updated_at
		FROM branches WHERE id = $1`,
		id).Scan(
		&b.ID, &b.TenantID, &b.Name, &b.Code, &b.Type, &b.Document,
		&addr.Street, &addr.Number, &addr.Complement, &addr.District,
		&addr.City, &addr.State, &addr.ZipCode, &addr.Country,
		&b.Phone, &b.Email, &b.Status, &b.IsMain, &b.CreatedAt, &b.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBranchNotFound
		}
		return nil, fmt.Errorf("erro ao buscar filial: %w", err)
	}

	b.Address = addr
	return &b, nil
}

// FindByTenantAndID implementa branch.Repository.FindByTenantAndID
func (r *BranchRepository) FindByTenantAndID(ctx context.Context, tenantID, id string) (*branch.Branch, error) {
	var b branch.Branch
	var addr branch.Address

	err := r.db.QueryRow(ctx,
		`SELECT id, tenant_id, name, code, type, document, street, number, complement, district, city, state, zip_code, country, phone, email, status, is_main, created_at, updated_at
		FROM branches WHERE id = $1 AND tenant_id = $2`,
		id, tenantID).Scan(
		&b.ID, &b.TenantID, &b.Name, &b.Code, &b.Type, &b.Document,
		&addr.Street, &addr.Number, &addr.Complement, &addr.District,
		&addr.City, &addr.State, &addr.ZipCode, &addr.Country,
		&b.Phone, &b.Email, &b.Status, &b.IsMain, &b.CreatedAt, &b.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBranchNotFound
		}
		return nil, fmt.Errorf("erro ao buscar filial: %w", err)
	}

	b.Address = addr
	return &b, nil
}

// FindMainBranch implementa branch.Repository.FindMainBranch
func (r *BranchRepository) FindMainBranch(ctx context.Context, tenantID string) (*branch.Branch, error) {
	var b branch.Branch
	var addr branch.Address

	err := r.db.QueryRow(ctx,
		`SELECT id, tenant_id, name, code, type, document, street, number, complement, district, city, state, zip_code, country, phone, email, status, is_main, created_at, updated_at
		FROM branches WHERE tenant_id = $1 AND is_main = true`,
		tenantID).Scan(
		&b.ID, &b.TenantID, &b.Name, &b.Code, &b.Type, &b.Document,
		&addr.Street, &addr.Number, &addr.Complement, &addr.District,
		&addr.City, &addr.State, &addr.ZipCode, &addr.Country,
		&b.Phone, &b.Email, &b.Status, &b.IsMain, &b.CreatedAt, &b.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBranchNotFound
		}
		return nil, fmt.Errorf("erro ao buscar filial principal: %w", err)
	}

	b.Address = addr
	return &b, nil
}

// ListByTenant implementa branch.Repository.ListByTenant
func (r *BranchRepository) ListByTenant(ctx context.Context, tenantID string, limit, offset int) ([]*branch.Branch, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, tenant_id, name, code, type, document, street, number, complement, district, city, state, zip_code, country, phone, email, status, is_main, created_at, updated_at
		FROM branches WHERE tenant_id = $1 ORDER BY name LIMIT $2 OFFSET $3`,
		tenantID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar filiais: %w", err)
	}
	defer rows.Close()

	branches := make([]*branch.Branch, 0)
	for rows.Next() {
		var b branch.Branch
		var addr branch.Address

		err = rows.Scan(
			&b.ID, &b.TenantID, &b.Name, &b.Code, &b.Type, &b.Document,
			&addr.Street, &addr.Number, &addr.Complement, &addr.District,
			&addr.City, &addr.State, &addr.ZipCode, &addr.Country,
			&b.Phone, &b.Email, &b.Status, &b.IsMain, &b.CreatedAt, &b.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("erro ao mapear filial: %w", err)
		}

		b.Address = addr
		branches = append(branches, &b)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("erro ao ler resultados: %w", err)
	}

	return branches, nil
}

// Update implementa branch.Repository.Update
func (r *BranchRepository) Update(ctx context.Context, b *branch.Branch) error {
	// Verificar se a filial existe
	existing, err := r.FindByID(ctx, b.ID)
	if err != nil {
		return err
	}

	// Não permitir alteração do status IsMain (deve usar método específico)
	b.IsMain = existing.IsMain

	// Atualizar a filial
	_, err = r.db.Exec(ctx,
		`UPDATE branches SET 
			name = $1, code = $2, type = $3, document = $4,
			street = $5, number = $6, complement = $7, district = $8,
			city = $9, state = $10, zip_code = $11, country = $12,
			phone = $13, email = $14, status = $15, updated_at = $16
		WHERE id = $17`,
		b.Name, b.Code, b.Type, b.Document,
		b.Address.Street, b.Address.Number, b.Address.Complement, b.Address.District,
		b.Address.City, b.Address.State, b.Address.ZipCode, b.Address.Country,
		b.Phone, b.Email, string(b.Status), b.UpdatedAt, b.ID)

	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return ErrBranchDuplicateKey
		}
		return fmt.Errorf("erro ao atualizar filial: %w", err)
	}

	return nil
}

// Delete implementa branch.Repository.Delete
func (r *BranchRepository) Delete(ctx context.Context, id string) error {
	// Buscar a filial para verificar se é principal
	var isMain bool
	err := r.db.QueryRow(ctx, "SELECT is_main FROM branches WHERE id = $1", id).Scan(&isMain)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrBranchNotFound
		}
		return fmt.Errorf("erro ao buscar filial: %w", err)
	}

	// Não permitir exclusão da filial principal
	if isMain {
		return errors.New("não é permitido excluir a filial principal")
	}

	// Excluir a filial
	result, err := r.db.Exec(ctx, "DELETE FROM branches WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("erro ao excluir filial: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrBranchNotFound
	}

	return nil
}

// UpdateStatus implementa branch.Repository.UpdateStatus
func (r *BranchRepository) UpdateStatus(ctx context.Context, id string, status branch.Status) error {
	// Verificar se a filial existe
	b, err := r.FindByID(ctx, id)
	if err != nil {
		return err
	}

	// Não permitir inativar a filial principal
	if b.IsMain && status != branch.StatusActive {
		return errors.New("não é permitido desativar a filial principal")
	}

	// Atualizar o status da filial
	_, err = r.db.Exec(ctx,
		"UPDATE branches SET status = $1, updated_at = NOW() WHERE id = $2",
		string(status), id)

	if err != nil {
		return fmt.Errorf("erro ao atualizar status da filial: %w", err)
	}

	return nil
}

// Exists implementa branch.Repository.Exists
func (r *BranchRepository) Exists(ctx context.Context, id string) (bool, error) {
	var count int
	err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM branches WHERE id = $1", id).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("erro ao verificar existência de filial: %w", err)
	}

	return count > 0, nil
}

// CountByTenant implementa branch.Repository.CountByTenant
func (r *BranchRepository) CountByTenant(ctx context.Context, tenantID string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM branches WHERE tenant_id = $1", tenantID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("erro ao contar filiais do tenant: %w", err)
	}
	return count, nil
}

// Métodos auxiliares

func (r *BranchRepository) tenantExists(ctx context.Context, tenantID string) (bool, error) {
	var count int
	err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM tenants WHERE id = $1", tenantID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("erro ao verificar existência de tenant: %w", err)
	}
	return count > 0, nil
}

func (r *BranchRepository) hasMainBranch(ctx context.Context, tenantID string) (bool, error) {
	var count int
	err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM branches WHERE tenant_id = $1 AND is_main = true", tenantID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("erro ao verificar existência de filial principal: %w", err)
	}
	return count > 0, nil
}
