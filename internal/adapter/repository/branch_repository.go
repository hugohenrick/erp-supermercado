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
	conn, err := r.db.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("falha ao obter conexão: %w", err)
	}
	defer conn.Release()

	// Verificar o tenant ID do branch e do contexto
	tenantIDFromContext := pkgtenant.GetTenantID(ctx)
	fmt.Printf("DEBUG Create Branch - TenantID do branch: '%s', TenantID do contexto: '%s'\n", b.TenantID, tenantIDFromContext)

	// Se o tenant ID do contexto for válido e diferente do tenant ID do branch, vamos usar o do contexto
	if tenantIDFromContext != "" && b.TenantID != tenantIDFromContext {
		fmt.Printf("DEBUG Create Branch - Substituindo TenantID do branch pelo do contexto\n")
		b.TenantID = tenantIDFromContext
	}

	// Primeiro, definir o search_path para public para garantir que acessamos os tenants
	_, err = conn.Exec(ctx, "SET search_path TO public")
	if err != nil {
		return fmt.Errorf("falha ao configurar search_path: %w", err)
	}

	// Verificar se o tenant existe e está ativo
	exists, err := r.tenantExists(ctx, b.TenantID)
	if err != nil {
		return fmt.Errorf("erro ao verificar tenant: %w", err)
	}
	if !exists {
		return pkgtenant.ErrTenantNotFound
	}

	// Obter o schema do tenant a partir do tenant_id
	var schema string
	err = conn.QueryRow(ctx, "SELECT schema FROM public.tenants WHERE id = $1", b.TenantID).Scan(&schema)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errors.New("tenant não encontrado")
		}
		return fmt.Errorf("falha ao obter schema do tenant: %w", err)
	}

	// Debug para verificar o schema recuperado
	fmt.Printf("DEBUG Create Branch - Tenant ID: %s, Schema: %s\n", b.TenantID, schema)

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
	err = conn.QueryRow(ctx, "SELECT max_branches FROM public.tenants WHERE id = $1", b.TenantID).Scan(&maxBranches)
	if err != nil {
		return fmt.Errorf("erro ao consultar limite de filiais: %w", err)
	}

	if count >= maxBranches {
		return ErrBranchLimitExceeded
	}

	// Inserir a filial no schema específico do tenant
	query := fmt.Sprintf(`INSERT INTO %s.branches 
		(id, tenant_id, name, code, type, document, street, number, complement, district, city, state, zip_code, country, phone, email, status, is_main, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)`, schema)

	_, err = conn.Exec(ctx, query,
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
	conn, err := r.db.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("falha ao obter conexão: %w", err)
	}
	defer conn.Release()

	// Obter tenant ID do contexto
	tenantID := pkgtenant.GetTenantID(ctx)
	if tenantID == "" {
		return nil, errors.New("tenant ID não encontrado no contexto")
	}

	// Primeiro, definir o search_path para public para garantir que acessamos os tenants
	_, err = conn.Exec(ctx, "SET search_path TO public")
	if err != nil {
		return nil, fmt.Errorf("falha ao configurar search_path: %w", err)
	}

	// Obter o schema do tenant a partir do tenant_id
	var schema string
	err = conn.QueryRow(ctx, "SELECT schema FROM public.tenants WHERE id = $1", tenantID).Scan(&schema)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("tenant não encontrado")
		}
		return nil, fmt.Errorf("falha ao obter schema do tenant: %w", err)
	}

	// Debug para verificar o schema recuperado
	fmt.Printf("DEBUG FindByID Branch - Tenant ID: %s, Schema: %s\n", tenantID, schema)

	var b branch.Branch
	var addr branch.Address

	query := fmt.Sprintf(`
		SELECT id, tenant_id, name, code, type, document, street, number, complement, district, city, state, zip_code, country, phone, email, status, is_main, created_at, updated_at
		FROM %s.branches WHERE id = $1`, schema)

	err = conn.QueryRow(ctx, query, id).Scan(
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
	conn, err := r.db.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("falha ao obter conexão: %w", err)
	}
	defer conn.Release()

	// Primeiro, definir o search_path para public para garantir que acessamos os tenants
	_, err = conn.Exec(ctx, "SET search_path TO public")
	if err != nil {
		return nil, fmt.Errorf("falha ao configurar search_path: %w", err)
	}

	// Obter o schema do tenant a partir do tenant_id
	var schema string
	err = conn.QueryRow(ctx, "SELECT schema FROM public.tenants WHERE id = $1", tenantID).Scan(&schema)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("tenant não encontrado")
		}
		return nil, fmt.Errorf("falha ao obter schema do tenant: %w", err)
	}

	// Debug para verificar o schema recuperado
	fmt.Printf("DEBUG FindByTenantAndID Branch - Tenant ID: %s, Schema: %s\n", tenantID, schema)

	var b branch.Branch
	var addr branch.Address

	query := fmt.Sprintf(`
		SELECT id, tenant_id, name, code, type, document, street, number, complement, district, city, state, zip_code, country, phone, email, status, is_main, created_at, updated_at
		FROM %s.branches WHERE id = $1 AND tenant_id = $2`, schema)

	err = conn.QueryRow(ctx, query, id, tenantID).Scan(
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
	conn, err := r.db.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("falha ao obter conexão: %w", err)
	}
	defer conn.Release()

	// Primeiro, definir o search_path para public para garantir que acessamos os tenants
	_, err = conn.Exec(ctx, "SET search_path TO public")
	if err != nil {
		return nil, fmt.Errorf("falha ao configurar search_path: %w", err)
	}

	// Obter o schema do tenant a partir do tenant_id
	var schema string
	err = conn.QueryRow(ctx, "SELECT schema FROM public.tenants WHERE id = $1", tenantID).Scan(&schema)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("tenant não encontrado")
		}
		return nil, fmt.Errorf("falha ao obter schema do tenant: %w", err)
	}

	// Debug para verificar o schema recuperado
	fmt.Printf("DEBUG FindMainBranch - Tenant ID: %s, Schema: %s\n", tenantID, schema)

	var b branch.Branch
	var addr branch.Address

	query := fmt.Sprintf(`
		SELECT id, tenant_id, name, code, type, document, street, number, complement, district, city, state, zip_code, country, phone, email, status, is_main, created_at, updated_at
		FROM %s.branches WHERE tenant_id = $1 AND is_main = true`, schema)

	err = conn.QueryRow(ctx, query, tenantID).Scan(
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
	conn, err := r.db.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("falha ao obter conexão: %w", err)
	}
	defer conn.Release()

	// Primeiro, definir o search_path para public para garantir que acessamos os tenants
	_, err = conn.Exec(ctx, "SET search_path TO public")
	if err != nil {
		return nil, fmt.Errorf("falha ao configurar search_path: %w", err)
	}

	// Obter o schema do tenant a partir do tenant_id
	var schema string
	err = conn.QueryRow(ctx, "SELECT schema FROM public.tenants WHERE id = $1", tenantID).Scan(&schema)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("tenant não encontrado")
		}
		return nil, fmt.Errorf("falha ao obter schema do tenant: %w", err)
	}

	// Debug para verificar o schema recuperado
	fmt.Printf("DEBUG ListByTenant - Tenant ID: %s, Schema: %s\n", tenantID, schema)

	query := fmt.Sprintf(`
		SELECT id, tenant_id, name, code, type, document, street, number, complement, district, city, state, zip_code, country, phone, email, status, is_main, created_at, updated_at
		FROM %s.branches WHERE tenant_id = $1 ORDER BY name LIMIT $2 OFFSET $3`, schema)

	rows, err := conn.Query(ctx, query, tenantID, limit, offset)
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
	conn, err := r.db.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("falha ao obter conexão: %w", err)
	}
	defer conn.Release()

	// Primeiro, definir o search_path para public para garantir que acessamos os tenants
	_, err = conn.Exec(ctx, "SET search_path TO public")
	if err != nil {
		return fmt.Errorf("falha ao configurar search_path: %w", err)
	}

	// Obter o schema do tenant a partir do tenant_id
	var schema string
	err = conn.QueryRow(ctx, "SELECT schema FROM public.tenants WHERE id = $1", b.TenantID).Scan(&schema)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errors.New("tenant não encontrado")
		}
		return fmt.Errorf("falha ao obter schema do tenant: %w", err)
	}

	// Debug para verificar o schema recuperado
	fmt.Printf("DEBUG Update Branch - Tenant ID: %s, Schema: %s\n", b.TenantID, schema)

	// Verificar se a filial existe
	var exists bool
	checkQuery := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s.branches WHERE id = $1 AND tenant_id = $2)", schema)
	err = conn.QueryRow(ctx, checkQuery, b.ID, b.TenantID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("erro ao verificar existência da filial: %w", err)
	}

	if !exists {
		return ErrBranchNotFound
	}

	// Verificar se a filial é principal
	var isMain bool
	mainQuery := fmt.Sprintf("SELECT is_main FROM %s.branches WHERE id = $1", schema)
	err = conn.QueryRow(ctx, mainQuery, b.ID).Scan(&isMain)
	if err != nil {
		return fmt.Errorf("erro ao verificar se filial é principal: %w", err)
	}

	// Não permitir alteração do status IsMain (deve usar método específico)
	b.IsMain = isMain

	// Atualizar a filial no schema específico do tenant
	query := fmt.Sprintf(`
		UPDATE %s.branches SET 
			name = $1, code = $2, type = $3, document = $4,
			street = $5, number = $6, complement = $7, district = $8,
			city = $9, state = $10, zip_code = $11, country = $12,
			phone = $13, email = $14, status = $15, updated_at = $16
		WHERE id = $17`, schema)

	_, err = conn.Exec(ctx, query,
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
	conn, err := r.db.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("falha ao obter conexão: %w", err)
	}
	defer conn.Release()

	// Obter tenant ID do contexto
	tenantID := pkgtenant.GetTenantID(ctx)
	if tenantID == "" {
		return errors.New("tenant ID não encontrado no contexto")
	}

	// Primeiro, definir o search_path para public para garantir que acessamos os tenants
	_, err = conn.Exec(ctx, "SET search_path TO public")
	if err != nil {
		return fmt.Errorf("falha ao configurar search_path: %w", err)
	}

	// Obter o schema do tenant a partir do tenant_id
	var schema string
	err = conn.QueryRow(ctx, "SELECT schema FROM public.tenants WHERE id = $1", tenantID).Scan(&schema)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errors.New("tenant não encontrado")
		}
		return fmt.Errorf("falha ao obter schema do tenant: %w", err)
	}

	// Debug para verificar o schema recuperado
	fmt.Printf("DEBUG Delete Branch - Tenant ID: %s, Schema: %s\n", tenantID, schema)

	// Buscar a filial para verificar se é principal
	var isMain bool
	mainQuery := fmt.Sprintf("SELECT is_main FROM %s.branches WHERE id = $1", schema)
	err = conn.QueryRow(ctx, mainQuery, id).Scan(&isMain)
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

	// Excluir a filial no schema específico do tenant
	query := fmt.Sprintf("DELETE FROM %s.branches WHERE id = $1 AND tenant_id = $2", schema)
	result, err := conn.Exec(ctx, query, id, tenantID)
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
	conn, err := r.db.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("falha ao obter conexão: %w", err)
	}
	defer conn.Release()

	// Obter tenant ID do contexto
	tenantID := pkgtenant.GetTenantID(ctx)
	if tenantID == "" {
		return errors.New("tenant ID não encontrado no contexto")
	}

	// Primeiro, definir o search_path para public para garantir que acessamos os tenants
	_, err = conn.Exec(ctx, "SET search_path TO public")
	if err != nil {
		return fmt.Errorf("falha ao configurar search_path: %w", err)
	}

	// Obter o schema do tenant a partir do tenant_id
	var schema string
	err = conn.QueryRow(ctx, "SELECT schema FROM public.tenants WHERE id = $1", tenantID).Scan(&schema)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errors.New("tenant não encontrado")
		}
		return fmt.Errorf("falha ao obter schema do tenant: %w", err)
	}

	// Debug para verificar o schema recuperado
	fmt.Printf("DEBUG UpdateStatus Branch - Tenant ID: %s, Schema: %s\n", tenantID, schema)

	// Atualizar status da filial
	query := fmt.Sprintf("UPDATE %s.branches SET status = $1 WHERE id = $2 AND tenant_id = $3", schema)
	result, err := conn.Exec(ctx, query, string(status), id, tenantID)
	if err != nil {
		return fmt.Errorf("erro ao atualizar status da filial: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrBranchNotFound
	}

	return nil
}

// Exists implementa branch.Repository.Exists
func (r *BranchRepository) Exists(ctx context.Context, id string) (bool, error) {
	conn, err := r.db.Acquire(ctx)
	if err != nil {
		return false, fmt.Errorf("falha ao obter conexão: %w", err)
	}
	defer conn.Release()

	// Obter tenant ID do contexto
	tenantID := pkgtenant.GetTenantID(ctx)
	if tenantID == "" {
		return false, errors.New("tenant ID não encontrado no contexto")
	}

	// Primeiro, definir o search_path para public para garantir que acessamos os tenants
	_, err = conn.Exec(ctx, "SET search_path TO public")
	if err != nil {
		return false, fmt.Errorf("falha ao configurar search_path: %w", err)
	}

	// Obter o schema do tenant a partir do tenant_id
	var schema string
	err = conn.QueryRow(ctx, "SELECT schema FROM public.tenants WHERE id = $1", tenantID).Scan(&schema)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, errors.New("tenant não encontrado")
		}
		return false, fmt.Errorf("falha ao obter schema do tenant: %w", err)
	}

	// Debug para verificar o schema recuperado
	fmt.Printf("DEBUG Exists Branch - Tenant ID: %s, Schema: %s\n", tenantID, schema)

	var exists bool
	query := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s.branches WHERE id = $1 AND tenant_id = $2)", schema)
	err = conn.QueryRow(ctx, query, id, tenantID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("erro ao verificar existência da filial: %w", err)
	}

	return exists, nil
}

// CountByTenant implementa branch.Repository.CountByTenant
func (r *BranchRepository) CountByTenant(ctx context.Context, tenantID string) (int, error) {
	conn, err := r.db.Acquire(ctx)
	if err != nil {
		return 0, fmt.Errorf("falha ao obter conexão: %w", err)
	}
	defer conn.Release()

	// Primeiro, definir o search_path para public para garantir que acessamos os tenants
	_, err = conn.Exec(ctx, "SET search_path TO public")
	if err != nil {
		return 0, fmt.Errorf("falha ao configurar search_path: %w", err)
	}

	// Obter o schema do tenant a partir do tenant_id
	var schema string
	err = conn.QueryRow(ctx, "SELECT schema FROM public.tenants WHERE id = $1", tenantID).Scan(&schema)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, errors.New("tenant não encontrado")
		}
		return 0, fmt.Errorf("falha ao obter schema do tenant: %w", err)
	}

	// Debug para verificar o schema recuperado
	fmt.Printf("DEBUG CountByTenant Branch - Tenant ID: %s, Schema: %s\n", tenantID, schema)

	var count int
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s.branches WHERE tenant_id = $1", schema)
	err = conn.QueryRow(ctx, query, tenantID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("erro ao contar filiais: %w", err)
	}

	return count, nil
}

// Métodos auxiliares

func (r *BranchRepository) tenantExists(ctx context.Context, tenantID string) (bool, error) {
	conn, err := r.db.Acquire(ctx)
	if err != nil {
		return false, fmt.Errorf("falha ao obter conexão: %w", err)
	}
	defer conn.Release()

	// Verificar se o tenantID é um UUID válido
	fmt.Printf("DEBUG tenantExists - TenantID recebido: '%s'\n", tenantID)

	// Configurar search_path para public
	_, err = conn.Exec(ctx, "SET search_path TO public")
	if err != nil {
		return false, fmt.Errorf("falha ao configurar search_path: %w", err)
	}

	// Primeiro verificar se existe o tenant com esse ID
	var count int
	err = conn.QueryRow(ctx, "SELECT COUNT(*) FROM public.tenants WHERE id = $1", tenantID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("erro ao contar tenant: %w", err)
	}

	fmt.Printf("DEBUG tenantExists - Contagem de registros: %d\n", count)

	if count == 0 {
		// Tenant não existe
		// Verificamos todos os tenants para debug
		rows, err := conn.Query(ctx, "SELECT id, name, schema FROM public.tenants LIMIT 10")
		if err != nil {
			fmt.Printf("DEBUG tenantExists - Erro ao listar tenants: %s\n", err.Error())
		} else {
			defer rows.Close()
			fmt.Println("DEBUG tenantExists - Tenants disponíveis:")
			for rows.Next() {
				var id, name, schema string
				if err := rows.Scan(&id, &name, &schema); err == nil {
					fmt.Printf("   ID: %s, Nome: %s, Schema: %s\n", id, name, schema)
				}
			}
		}
		return false, nil
	}

	// Verificar qual é o status atual do tenant
	var status string
	err = conn.QueryRow(ctx, "SELECT status FROM public.tenants WHERE id = $1", tenantID).Scan(&status)
	if err != nil {
		return false, fmt.Errorf("erro ao obter status do tenant: %w", err)
	}

	fmt.Printf("DEBUG tenantExists - Status atual do tenant: '%s'\n", status)

	// Verificar se o tenant está ativo (considerando tanto 'ACTIVE' quanto 'active')
	var exists bool
	statusQuery := `
		SELECT EXISTS(
			SELECT 1 FROM public.tenants 
			WHERE id = $1 
			AND (
				LOWER(status) = LOWER('ACTIVE') OR
				status = 'active'
			)
		)`
	err = conn.QueryRow(ctx, statusQuery, tenantID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("erro ao verificar status do tenant: %w", err)
	}

	fmt.Printf("DEBUG tenantExists - Tenant existe e está ativo: %v\n", exists)

	// Se o tenant existe e a verificação de status passou, retornamos true
	return exists, nil
}

func (r *BranchRepository) hasMainBranch(ctx context.Context, tenantID string) (bool, error) {
	conn, err := r.db.Acquire(ctx)
	if err != nil {
		return false, fmt.Errorf("falha ao obter conexão: %w", err)
	}
	defer conn.Release()

	// Primeiro, definir o search_path para public para garantir que acessamos os tenants
	_, err = conn.Exec(ctx, "SET search_path TO public")
	if err != nil {
		return false, fmt.Errorf("falha ao configurar search_path: %w", err)
	}

	// Obter o schema do tenant a partir do tenant_id
	var schema string
	err = conn.QueryRow(ctx, "SELECT schema FROM public.tenants WHERE id = $1", tenantID).Scan(&schema)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, errors.New("tenant não encontrado")
		}
		return false, fmt.Errorf("falha ao obter schema do tenant: %w", err)
	}

	// Debug para verificar o schema recuperado
	fmt.Printf("DEBUG hasMainBranch - Tenant ID: %s, Schema: %s\n", tenantID, schema)

	var exists bool
	query := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s.branches WHERE tenant_id = $1 AND is_main = true)", schema)
	err = conn.QueryRow(ctx, query, tenantID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("erro ao verificar existência de filial principal: %w", err)
	}

	return exists, nil
}
