package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hugohenrick/erp-supermercado/internal/domain/branch"
	"github.com/hugohenrick/erp-supermercado/internal/infrastructure/database"
	"github.com/hugohenrick/erp-supermercado/pkg/tenant"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Erros específicos do repositório
var (
	ErrBranchNotFound     = errors.New("filial não encontrada")
	ErrBranchDuplicateKey = errors.New("filial com mesmo código já existe para este tenant")
)

// PostgresBranchRepository implementa a interface branch.Repository usando PostgreSQL
type PostgresBranchRepository struct {
	db *database.PostgresDB
}

// NewPostgresBranchRepository cria uma nova instância de PostgresBranchRepository
func NewPostgresBranchRepository(db *database.PostgresDB) *PostgresBranchRepository {
	return &PostgresBranchRepository{
		db: db,
	}
}

// Create implementa branch.Repository.Create
func (r *PostgresBranchRepository) Create(ctx context.Context, b *branch.Branch) error {
	conn, err := r.db.GetTenantConnection(ctx)
	if err != nil {
		return fmt.Errorf("falha ao obter conexão: %w", err)
	}
	defer conn.Release()

	// Verificar se já existe uma filial marcada como principal, se esta for marcada como principal
	if b.IsMain {
		var exists bool
		err = conn.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM branches WHERE tenant_id = $1 AND is_main = true)", b.TenantID).Scan(&exists)
		if err != nil {
			return fmt.Errorf("falha ao verificar filial principal: %w", err)
		}

		if exists {
			return errors.New("já existe uma filial principal para este tenant")
		}
	}

	// Transação para garantir atomicidade
	err = r.db.Transaction(ctx, func(tx pgx.Tx) error {
		query := `
			INSERT INTO branches (
				id, tenant_id, name, code, type, document, 
				street, number, complement, district, city, state, zip_code, country,
				phone, email, status, is_main, created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, 
				$7, $8, $9, $10, $11, $12, $13, $14, 
				$15, $16, $17, $18, $19, $20
			)
		`

		_, err := tx.Exec(ctx, query,
			b.ID,
			b.TenantID,
			b.Name,
			b.Code,
			string(b.Type),
			b.Document,
			b.Address.Street,
			b.Address.Number,
			b.Address.Complement,
			b.Address.District,
			b.Address.City,
			b.Address.State,
			b.Address.ZipCode,
			b.Address.Country,
			b.Phone,
			b.Email,
			string(b.Status),
			b.IsMain,
			b.CreatedAt,
			b.UpdatedAt,
		)

		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) {
				if pgErr.Code == "23505" { // Unique violation
					return ErrBranchDuplicateKey
				}
			}
			return fmt.Errorf("falha ao inserir filial: %w", err)
		}

		return nil
	})

	return err
}

// FindByID implementa branch.Repository.FindByID
func (r *PostgresBranchRepository) FindByID(ctx context.Context, id string) (*branch.Branch, error) {
	conn, err := r.db.GetTenantConnection(ctx)
	if err != nil {
		return nil, fmt.Errorf("falha ao obter conexão: %w", err)
	}
	defer conn.Release()

	return r.findBranchByQuery(ctx, conn, "SELECT * FROM branches WHERE id = $1", id)
}

// FindByTenantAndID implementa branch.Repository.FindByTenantAndID
func (r *PostgresBranchRepository) FindByTenantAndID(ctx context.Context, tenantID, id string) (*branch.Branch, error) {
	conn, err := r.db.GetTenantConnection(ctx)
	if err != nil {
		return nil, fmt.Errorf("falha ao obter conexão: %w", err)
	}
	defer conn.Release()

	return r.findBranchByQuery(ctx, conn, "SELECT * FROM branches WHERE tenant_id = $1 AND id = $2", tenantID, id)
}

// FindMainBranch implementa branch.Repository.FindMainBranch
func (r *PostgresBranchRepository) FindMainBranch(ctx context.Context, tenantID string) (*branch.Branch, error) {
	conn, err := r.db.GetTenantConnection(ctx)
	if err != nil {
		return nil, fmt.Errorf("falha ao obter conexão: %w", err)
	}
	defer conn.Release()

	return r.findBranchByQuery(ctx, conn, "SELECT * FROM branches WHERE tenant_id = $1 AND is_main = true", tenantID)
}

// findBranchByQuery é um método auxiliar para executar queries de busca de filial
func (r *PostgresBranchRepository) findBranchByQuery(ctx context.Context, conn *pgxpool.Conn, query string, args ...interface{}) (*branch.Branch, error) {
	b := &branch.Branch{
		Address: branch.Address{},
	}

	var branchType, status string

	err := conn.QueryRow(ctx, query, args...).Scan(
		&b.ID,
		&b.TenantID,
		&b.Name,
		&b.Code,
		&branchType,
		&b.Document,
		&b.Address.Street,
		&b.Address.Number,
		&b.Address.Complement,
		&b.Address.District,
		&b.Address.City,
		&b.Address.State,
		&b.Address.ZipCode,
		&b.Address.Country,
		&b.Phone,
		&b.Email,
		&status,
		&b.IsMain,
		&b.CreatedAt,
		&b.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBranchNotFound
		}
		return nil, fmt.Errorf("falha ao buscar filial: %w", err)
	}

	b.Type = branch.BranchType(branchType)
	b.Status = branch.Status(status)

	return b, nil
}

// Update implementa branch.Repository.Update
func (r *PostgresBranchRepository) Update(ctx context.Context, b *branch.Branch) error {
	conn, err := r.db.GetTenantConnection(ctx)
	if err != nil {
		return fmt.Errorf("falha ao obter conexão: %w", err)
	}
	defer conn.Release()

	// Verificar se a filial já existe
	exists, err := r.Exists(ctx, b.ID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrBranchNotFound
	}

	// Se estiver definindo como filial principal, verificar se já existe outra
	if b.IsMain {
		var mainExists bool
		err = conn.QueryRow(ctx,
			"SELECT EXISTS(SELECT 1 FROM branches WHERE tenant_id = $1 AND is_main = true AND id != $2)",
			b.TenantID, b.ID).Scan(&mainExists)
		if err != nil {
			return fmt.Errorf("falha ao verificar filial principal: %w", err)
		}

		if mainExists {
			return errors.New("já existe uma filial principal para este tenant")
		}
	}

	// Atualizar a filial
	query := `
		UPDATE branches
		SET 
			name = $1,
			code = $2,
			type = $3,
			document = $4,
			street = $5,
			number = $6,
			complement = $7,
			district = $8,
			city = $9,
			state = $10,
			zip_code = $11,
			country = $12,
			phone = $13,
			email = $14,
			status = $15,
			is_main = $16,
			updated_at = $17
		WHERE 
			id = $18 AND tenant_id = $19
	`

	result, err := conn.Exec(ctx, query,
		b.Name,
		b.Code,
		string(b.Type),
		b.Document,
		b.Address.Street,
		b.Address.Number,
		b.Address.Complement,
		b.Address.District,
		b.Address.City,
		b.Address.State,
		b.Address.ZipCode,
		b.Address.Country,
		b.Phone,
		b.Email,
		string(b.Status),
		b.IsMain,
		time.Now(),
		b.ID,
		b.TenantID,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // Unique violation
			return ErrBranchDuplicateKey
		}
		return fmt.Errorf("falha ao atualizar filial: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrBranchNotFound
	}

	return nil
}

// Delete implementa branch.Repository.Delete
func (r *PostgresBranchRepository) Delete(ctx context.Context, id string) error {
	// Primeiro obtemos o tenant ID do contexto
	tenantID := tenant.GetTenantID(ctx)
	if tenantID == "" {
		return errors.New("tenant ID não encontrado no contexto")
	}

	conn, err := r.db.GetTenantConnection(ctx)
	if err != nil {
		return fmt.Errorf("falha ao obter conexão: %w", err)
	}
	defer conn.Release()

	// Verificar se é a filial principal
	var isMain bool
	err = conn.QueryRow(ctx, "SELECT is_main FROM branches WHERE id = $1 AND tenant_id = $2", id, tenantID).Scan(&isMain)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrBranchNotFound
		}
		return fmt.Errorf("falha ao verificar se é filial principal: %w", err)
	}

	if isMain {
		return errors.New("não é possível excluir a filial principal")
	}

	// Excluir a filial
	result, err := conn.Exec(ctx, "DELETE FROM branches WHERE id = $1 AND tenant_id = $2", id, tenantID)
	if err != nil {
		return fmt.Errorf("falha ao excluir filial: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrBranchNotFound
	}

	return nil
}

// ListByTenant implementa branch.Repository.ListByTenant
func (r *PostgresBranchRepository) ListByTenant(ctx context.Context, tenantID string, limit, offset int) ([]*branch.Branch, error) {
	conn, err := r.db.GetTenantConnection(ctx)
	if err != nil {
		return nil, fmt.Errorf("falha ao obter conexão: %w", err)
	}
	defer conn.Release()

	query := `
		SELECT 
			id, tenant_id, name, code, type, document, 
			street, number, complement, district, city, state, zip_code, country,
			phone, email, status, is_main, created_at, updated_at
		FROM 
			branches
		WHERE 
			tenant_id = $1
		ORDER BY 
			is_main DESC, name ASC
		LIMIT $2 OFFSET $3
	`

	rows, err := conn.Query(ctx, query, tenantID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("falha ao listar filiais: %w", err)
	}
	defer rows.Close()

	var branches []*branch.Branch

	for rows.Next() {
		b := &branch.Branch{
			Address: branch.Address{},
		}
		var branchType, status string

		err := rows.Scan(
			&b.ID,
			&b.TenantID,
			&b.Name,
			&b.Code,
			&branchType,
			&b.Document,
			&b.Address.Street,
			&b.Address.Number,
			&b.Address.Complement,
			&b.Address.District,
			&b.Address.City,
			&b.Address.State,
			&b.Address.ZipCode,
			&b.Address.Country,
			&b.Phone,
			&b.Email,
			&status,
			&b.IsMain,
			&b.CreatedAt,
			&b.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("falha ao ler filial: %w", err)
		}

		b.Type = branch.BranchType(branchType)
		b.Status = branch.Status(status)
		branches = append(branches, b)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erro ao iterar resultados: %w", err)
	}

	return branches, nil
}

// CountByTenant implementa branch.Repository.CountByTenant
func (r *PostgresBranchRepository) CountByTenant(ctx context.Context, tenantID string) (int, error) {
	conn, err := r.db.GetTenantConnection(ctx)
	if err != nil {
		return 0, fmt.Errorf("falha ao obter conexão: %w", err)
	}
	defer conn.Release()

	var count int
	err = conn.QueryRow(ctx, "SELECT COUNT(*) FROM branches WHERE tenant_id = $1", tenantID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("falha ao contar filiais: %w", err)
	}

	return count, nil
}

// UpdateStatus implementa branch.Repository.UpdateStatus
func (r *PostgresBranchRepository) UpdateStatus(ctx context.Context, id string, status branch.Status) error {
	tenantID := tenant.GetTenantID(ctx)
	if tenantID == "" {
		return errors.New("tenant ID não encontrado no contexto")
	}

	conn, err := r.db.GetTenantConnection(ctx)
	if err != nil {
		return fmt.Errorf("falha ao obter conexão: %w", err)
	}
	defer conn.Release()

	// Verificar se é a filial principal
	var isMain bool
	err = conn.QueryRow(ctx, "SELECT is_main FROM branches WHERE id = $1 AND tenant_id = $2", id, tenantID).Scan(&isMain)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrBranchNotFound
		}
		return fmt.Errorf("falha ao verificar se é filial principal: %w", err)
	}

	// Se for a filial principal e estiver tentando desativar, retornar erro
	if isMain && status != branch.StatusActive {
		return errors.New("não é possível desativar a filial principal")
	}

	// Atualizar o status
	query := `
		UPDATE branches
		SET 
			status = $1,
			updated_at = $2
		WHERE 
			id = $3 AND tenant_id = $4
	`

	result, err := conn.Exec(ctx, query, string(status), time.Now(), id, tenantID)
	if err != nil {
		return fmt.Errorf("falha ao atualizar status da filial: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrBranchNotFound
	}

	return nil
}

// Exists implementa branch.Repository.Exists
func (r *PostgresBranchRepository) Exists(ctx context.Context, id string) (bool, error) {
	tenantID := tenant.GetTenantID(ctx)
	if tenantID == "" {
		return false, errors.New("tenant ID não encontrado no contexto")
	}

	conn, err := r.db.GetTenantConnection(ctx)
	if err != nil {
		return false, fmt.Errorf("falha ao obter conexão: %w", err)
	}
	defer conn.Release()

	var exists bool
	err = conn.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM branches WHERE id = $1 AND tenant_id = $2)", id, tenantID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("falha ao verificar existência da filial: %w", err)
	}

	return exists, nil
}
