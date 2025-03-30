package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/hugohenrick/erp-supermercado/internal/domain/certificate"
	"github.com/hugohenrick/erp-supermercado/pkg/branch"
	"github.com/hugohenrick/erp-supermercado/pkg/tenant"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CertificateRepository implementa a interface certificate.Repository
type CertificateRepository struct {
	db *pgxpool.Pool
}

// NewCertificateRepository cria uma nova instância de CertificateRepository
func NewCertificateRepository(db *pgxpool.Pool) certificate.Repository {
	return &CertificateRepository{
		db: db,
	}
}

// Create implementa o método Create da interface certificate.Repository
func (r *CertificateRepository) Create(ctx context.Context, cert *certificate.Certificate) error {
	conn, err := r.db.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("falha ao adquirir conexão: %w", err)
	}
	defer conn.Release()

	// Obter tenant_id do contexto
	tenantID := tenant.GetTenantIDFromContext(ctx)
	if tenantID == "" {
		return fmt.Errorf("tenant ID não encontrado no contexto")
	}

	// Configurar search_path para public para acessar a tabela de tenants
	_, err = conn.Exec(ctx, "SET search_path TO public")
	if err != nil {
		return fmt.Errorf("falha ao configurar search_path: %w", err)
	}

	// Obter o schema do tenant
	var schema string
	err = conn.QueryRow(ctx, "SELECT schema FROM tenants WHERE id = $1", tenantID).Scan(&schema)
	if err != nil {
		return fmt.Errorf("falha ao obter schema do tenant: %w", err)
	}

	// Verificar se a filial existe
	var exists bool
	err = conn.QueryRow(ctx, fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s.branches WHERE id = $1)", schema), cert.BranchID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("falha ao verificar se a filial existe: %w", err)
	}
	if !exists {
		return fmt.Errorf("filial com ID %s não encontrada", cert.BranchID)
	}

	// Verificar se o certificado já está ativo e desativar outros certificados ativos
	if cert.IsActive {
		_, err = conn.Exec(ctx, fmt.Sprintf("UPDATE %s.branch_certificates SET is_active = false WHERE branch_id = $1 AND is_active = true", schema), cert.BranchID)
		if err != nil {
			return fmt.Errorf("falha ao desativar certificados existentes: %w", err)
		}
	}

	// Inserir o novo certificado
	query := fmt.Sprintf(`
		INSERT INTO %s.branch_certificates (
			id, tenant_id, branch_id, name, certificate_data, certificate_path, 
			password, expiration_date, is_active, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, schema)

	_, err = conn.Exec(ctx, query,
		cert.ID, cert.TenantID, cert.BranchID, cert.Name, cert.CertificateData,
		cert.CertificatePath, cert.Password, cert.ExpirationDate, cert.IsActive,
		cert.CreatedAt, cert.UpdatedAt)

	if err != nil {
		return fmt.Errorf("falha ao inserir certificado: %w", err)
	}

	return nil
}

// FindByID implementa o método FindByID da interface certificate.Repository
func (r *CertificateRepository) FindByID(ctx context.Context, id string) (*certificate.Certificate, error) {
	conn, err := r.db.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("falha ao adquirir conexão: %w", err)
	}
	defer conn.Release()

	// Obter tenant_id do contexto
	tenantID := tenant.GetTenantIDFromContext(ctx)
	if tenantID == "" {
		return nil, fmt.Errorf("tenant ID não encontrado no contexto")
	}

	// Configurar search_path para public para acessar a tabela de tenants
	_, err = conn.Exec(ctx, "SET search_path TO public")
	if err != nil {
		return nil, fmt.Errorf("falha ao configurar search_path: %w", err)
	}

	// Obter o schema do tenant
	var schema string
	err = conn.QueryRow(ctx, "SELECT schema FROM tenants WHERE id = $1", tenantID).Scan(&schema)
	if err != nil {
		return nil, fmt.Errorf("falha ao obter schema do tenant: %w", err)
	}

	// Buscar o certificado
	query := fmt.Sprintf(`
		SELECT 
			id, tenant_id, branch_id, name, certificate_data, certificate_path, 
			password, expiration_date, is_active, created_at, updated_at
		FROM %s.branch_certificates
		WHERE id = $1 AND tenant_id = $2
	`, schema)

	var cert certificate.Certificate
	err = conn.QueryRow(ctx, query, id, tenantID).Scan(
		&cert.ID, &cert.TenantID, &cert.BranchID, &cert.Name, &cert.CertificateData,
		&cert.CertificatePath, &cert.Password, &cert.ExpirationDate, &cert.IsActive,
		&cert.CreatedAt, &cert.UpdatedAt)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("certificado com ID %s não encontrado", id)
		}
		return nil, fmt.Errorf("falha ao buscar certificado: %w", err)
	}

	return &cert, nil
}

// FindByBranch implementa o método FindByBranch da interface certificate.Repository
func (r *CertificateRepository) FindByBranch(ctx context.Context, branchID string) ([]*certificate.Certificate, error) {
	conn, err := r.db.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("falha ao adquirir conexão: %w", err)
	}
	defer conn.Release()

	// Obter tenant_id do contexto
	tenantID := tenant.GetTenantIDFromContext(ctx)
	if tenantID == "" {
		return nil, fmt.Errorf("tenant ID não encontrado no contexto")
	}

	// Configurar search_path para public para acessar a tabela de tenants
	_, err = conn.Exec(ctx, "SET search_path TO public")
	if err != nil {
		return nil, fmt.Errorf("falha ao configurar search_path: %w", err)
	}

	// Obter o schema do tenant
	var schema string
	err = conn.QueryRow(ctx, "SELECT schema FROM tenants WHERE id = $1", tenantID).Scan(&schema)
	if err != nil {
		return nil, fmt.Errorf("falha ao obter schema do tenant: %w", err)
	}

	// Obter branch_id do contexto se não fornecido
	if branchID == "" {
		branchID = branch.GetBranchID(ctx)
	}

	// Buscar os certificados
	query := fmt.Sprintf(`
		SELECT 
			id, tenant_id, branch_id, name, certificate_data, certificate_path, 
			password, expiration_date, is_active, created_at, updated_at
		FROM %s.branch_certificates
		WHERE branch_id = $1 AND tenant_id = $2
		ORDER BY is_active DESC, expiration_date DESC
	`, schema)

	rows, err := conn.Query(ctx, query, branchID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("falha ao buscar certificados: %w", err)
	}
	defer rows.Close()

	// Processar os resultados
	certificates := []*certificate.Certificate{}
	for rows.Next() {
		var cert certificate.Certificate
		err = rows.Scan(
			&cert.ID, &cert.TenantID, &cert.BranchID, &cert.Name, &cert.CertificateData,
			&cert.CertificatePath, &cert.Password, &cert.ExpirationDate, &cert.IsActive,
			&cert.CreatedAt, &cert.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("falha ao ler certificado: %w", err)
		}
		certificates = append(certificates, &cert)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("erro ao iterar certificados: %w", err)
	}

	return certificates, nil
}

// FindActiveCertificate implementa o método FindActiveCertificate da interface certificate.Repository
func (r *CertificateRepository) FindActiveCertificate(ctx context.Context, branchID string) (*certificate.Certificate, error) {
	conn, err := r.db.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("falha ao adquirir conexão: %w", err)
	}
	defer conn.Release()

	// Obter tenant_id do contexto
	tenantID := tenant.GetTenantIDFromContext(ctx)
	if tenantID == "" {
		return nil, fmt.Errorf("tenant ID não encontrado no contexto")
	}

	// Configurar search_path para public para acessar a tabela de tenants
	_, err = conn.Exec(ctx, "SET search_path TO public")
	if err != nil {
		return nil, fmt.Errorf("falha ao configurar search_path: %w", err)
	}

	// Obter o schema do tenant
	var schema string
	err = conn.QueryRow(ctx, "SELECT schema FROM tenants WHERE id = $1", tenantID).Scan(&schema)
	if err != nil {
		return nil, fmt.Errorf("falha ao obter schema do tenant: %w", err)
	}

	// Obter branch_id do contexto se não fornecido
	if branchID == "" {
		branchID = branch.GetBranchID(ctx)
	}

	// Buscar o certificado ativo
	query := fmt.Sprintf(`
		SELECT 
			id, tenant_id, branch_id, name, certificate_data, certificate_path, 
			password, expiration_date, is_active, created_at, updated_at
		FROM %s.branch_certificates
		WHERE branch_id = $1 AND tenant_id = $2 AND is_active = true
		LIMIT 1
	`, schema)

	var cert certificate.Certificate
	err = conn.QueryRow(ctx, query, branchID, tenantID).Scan(
		&cert.ID, &cert.TenantID, &cert.BranchID, &cert.Name, &cert.CertificateData,
		&cert.CertificatePath, &cert.Password, &cert.ExpirationDate, &cert.IsActive,
		&cert.CreatedAt, &cert.UpdatedAt)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("nenhum certificado ativo encontrado para a filial %s", branchID)
		}
		return nil, fmt.Errorf("falha ao buscar certificado ativo: %w", err)
	}

	return &cert, nil
}

// List implementa o método List da interface certificate.Repository
func (r *CertificateRepository) List(ctx context.Context, tenantID string, limit, offset int) ([]*certificate.Certificate, error) {
	conn, err := r.db.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("falha ao adquirir conexão: %w", err)
	}
	defer conn.Release()

	// Obter tenant_id do contexto se não fornecido
	if tenantID == "" {
		tenantID = tenant.GetTenantIDFromContext(ctx)
		if tenantID == "" {
			return nil, fmt.Errorf("tenant ID não encontrado no contexto")
		}
	}

	// Configurar search_path para public para acessar a tabela de tenants
	_, err = conn.Exec(ctx, "SET search_path TO public")
	if err != nil {
		return nil, fmt.Errorf("falha ao configurar search_path: %w", err)
	}

	// Obter o schema do tenant
	var schema string
	err = conn.QueryRow(ctx, "SELECT schema FROM tenants WHERE id = $1", tenantID).Scan(&schema)
	if err != nil {
		return nil, fmt.Errorf("falha ao obter schema do tenant: %w", err)
	}

	// Validar parâmetros de paginação
	if limit <= 0 {
		limit = 10 // Valor padrão
	}
	if offset < 0 {
		offset = 0
	}

	// Buscar os certificados com paginação
	query := fmt.Sprintf(`
		SELECT 
			id, tenant_id, branch_id, name, certificate_data, certificate_path, 
			password, expiration_date, is_active, created_at, updated_at
		FROM %s.branch_certificates
		WHERE tenant_id = $1
		ORDER BY branch_id, is_active DESC, expiration_date DESC
		LIMIT $2 OFFSET $3
	`, schema)

	rows, err := conn.Query(ctx, query, tenantID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("falha ao buscar certificados: %w", err)
	}
	defer rows.Close()

	// Processar os resultados
	certificates := []*certificate.Certificate{}
	for rows.Next() {
		var cert certificate.Certificate
		err = rows.Scan(
			&cert.ID, &cert.TenantID, &cert.BranchID, &cert.Name, &cert.CertificateData,
			&cert.CertificatePath, &cert.Password, &cert.ExpirationDate, &cert.IsActive,
			&cert.CreatedAt, &cert.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("falha ao ler certificado: %w", err)
		}
		certificates = append(certificates, &cert)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("erro ao iterar certificados: %w", err)
	}

	return certificates, nil
}

// Update implementa o método Update da interface certificate.Repository
func (r *CertificateRepository) Update(ctx context.Context, cert *certificate.Certificate) error {
	conn, err := r.db.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("falha ao adquirir conexão: %w", err)
	}
	defer conn.Release()

	// Obter tenant_id do contexto
	tenantID := tenant.GetTenantIDFromContext(ctx)
	if tenantID == "" {
		return fmt.Errorf("tenant ID não encontrado no contexto")
	}

	// Configurar search_path para public para acessar a tabela de tenants
	_, err = conn.Exec(ctx, "SET search_path TO public")
	if err != nil {
		return fmt.Errorf("falha ao configurar search_path: %w", err)
	}

	// Obter o schema do tenant
	var schema string
	err = conn.QueryRow(ctx, "SELECT schema FROM tenants WHERE id = $1", tenantID).Scan(&schema)
	if err != nil {
		return fmt.Errorf("falha ao obter schema do tenant: %w", err)
	}

	// Verificar se o certificado existe
	var exists bool
	err = conn.QueryRow(ctx, fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s.branch_certificates WHERE id = $1 AND tenant_id = $2)", schema), cert.ID, tenantID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("falha ao verificar se o certificado existe: %w", err)
	}
	if !exists {
		return fmt.Errorf("certificado com ID %s não encontrado", cert.ID)
	}

	// Verificar se o certificado já está ativo e desativar outros certificados ativos
	if cert.IsActive {
		_, err = conn.Exec(ctx, fmt.Sprintf("UPDATE %s.branch_certificates SET is_active = false WHERE branch_id = $1 AND id != $2 AND is_active = true", schema), cert.BranchID, cert.ID)
		if err != nil {
			return fmt.Errorf("falha ao desativar outros certificados: %w", err)
		}
	}

	// Atualizar o certificado
	query := fmt.Sprintf(`
		UPDATE %s.branch_certificates SET
			name = $1, certificate_data = $2, certificate_path = $3,
			password = $4, expiration_date = $5, is_active = $6, updated_at = $7
		WHERE id = $8 AND tenant_id = $9
	`, schema)

	_, err = conn.Exec(ctx, query,
		cert.Name, cert.CertificateData, cert.CertificatePath,
		cert.Password, cert.ExpirationDate, cert.IsActive, time.Now(),
		cert.ID, tenantID)

	if err != nil {
		return fmt.Errorf("falha ao atualizar certificado: %w", err)
	}

	return nil
}

// Delete implementa o método Delete da interface certificate.Repository
func (r *CertificateRepository) Delete(ctx context.Context, id string) error {
	conn, err := r.db.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("falha ao adquirir conexão: %w", err)
	}
	defer conn.Release()

	// Obter tenant_id do contexto
	tenantID := tenant.GetTenantIDFromContext(ctx)
	if tenantID == "" {
		return fmt.Errorf("tenant ID não encontrado no contexto")
	}

	// Configurar search_path para public para acessar a tabela de tenants
	_, err = conn.Exec(ctx, "SET search_path TO public")
	if err != nil {
		return fmt.Errorf("falha ao configurar search_path: %w", err)
	}

	// Obter o schema do tenant
	var schema string
	err = conn.QueryRow(ctx, "SELECT schema FROM tenants WHERE id = $1", tenantID).Scan(&schema)
	if err != nil {
		return fmt.Errorf("falha ao obter schema do tenant: %w", err)
	}

	// Verificar se o certificado está sendo usado em alguma configuração fiscal
	var usedByConfig bool
	err = conn.QueryRow(ctx, fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s.fiscal_configurations WHERE certificate_id = $1)", schema), id).Scan(&usedByConfig)
	if err != nil {
		return fmt.Errorf("falha ao verificar se o certificado está em uso: %w", err)
	}
	if usedByConfig {
		return fmt.Errorf("não é possível excluir o certificado pois está em uso em configurações fiscais")
	}

	// Excluir o certificado
	_, err = conn.Exec(ctx, fmt.Sprintf("DELETE FROM %s.branch_certificates WHERE id = $1 AND tenant_id = $2", schema), id, tenantID)
	if err != nil {
		return fmt.Errorf("falha ao excluir certificado: %w", err)
	}

	return nil
}

// Activate implementa o método Activate da interface certificate.Repository
func (r *CertificateRepository) Activate(ctx context.Context, id string) error {
	conn, err := r.db.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("falha ao adquirir conexão: %w", err)
	}
	defer conn.Release()

	// Obter tenant_id do contexto
	tenantID := tenant.GetTenantIDFromContext(ctx)
	if tenantID == "" {
		return fmt.Errorf("tenant ID não encontrado no contexto")
	}

	// Configurar search_path para public para acessar a tabela de tenants
	_, err = conn.Exec(ctx, "SET search_path TO public")
	if err != nil {
		return fmt.Errorf("falha ao configurar search_path: %w", err)
	}

	// Obter o schema do tenant
	var schema string
	err = conn.QueryRow(ctx, "SELECT schema FROM tenants WHERE id = $1", tenantID).Scan(&schema)
	if err != nil {
		return fmt.Errorf("falha ao obter schema do tenant: %w", err)
	}

	// Obter a filial do certificado
	var branchID string
	err = conn.QueryRow(ctx, fmt.Sprintf("SELECT branch_id FROM %s.branch_certificates WHERE id = $1 AND tenant_id = $2", schema), id, tenantID).Scan(&branchID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("certificado com ID %s não encontrado", id)
		}
		return fmt.Errorf("falha ao obter dados do certificado: %w", err)
	}

	// Iniciar transação
	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("falha ao iniciar transação: %w", err)
	}
	defer tx.Rollback(ctx)

	// Desativar todos os certificados da filial
	_, err = tx.Exec(ctx, fmt.Sprintf("UPDATE %s.branch_certificates SET is_active = false WHERE branch_id = $1", schema), branchID)
	if err != nil {
		return fmt.Errorf("falha ao desativar certificados: %w", err)
	}

	// Ativar o certificado especificado
	_, err = tx.Exec(ctx, fmt.Sprintf("UPDATE %s.branch_certificates SET is_active = true, updated_at = $1 WHERE id = $2", schema), time.Now(), id)
	if err != nil {
		return fmt.Errorf("falha ao ativar certificado: %w", err)
	}

	// Confirmar transação
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("falha ao confirmar transação: %w", err)
	}

	return nil
}

// Deactivate implementa o método Deactivate da interface certificate.Repository
func (r *CertificateRepository) Deactivate(ctx context.Context, id string) error {
	conn, err := r.db.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("falha ao adquirir conexão: %w", err)
	}
	defer conn.Release()

	// Obter tenant_id do contexto
	tenantID := tenant.GetTenantIDFromContext(ctx)
	if tenantID == "" {
		return fmt.Errorf("tenant ID não encontrado no contexto")
	}

	// Configurar search_path para public para acessar a tabela de tenants
	_, err = conn.Exec(ctx, "SET search_path TO public")
	if err != nil {
		return fmt.Errorf("falha ao configurar search_path: %w", err)
	}

	// Obter o schema do tenant
	var schema string
	err = conn.QueryRow(ctx, "SELECT schema FROM tenants WHERE id = $1", tenantID).Scan(&schema)
	if err != nil {
		return fmt.Errorf("falha ao obter schema do tenant: %w", err)
	}

	// Verificar se o certificado está sendo usado em alguma configuração fiscal
	var isUsedByConfig bool
	err = conn.QueryRow(ctx, fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s.fiscal_configurations WHERE certificate_id = $1)", schema), id).Scan(&isUsedByConfig)
	if err != nil {
		return fmt.Errorf("falha ao verificar se o certificado está em uso: %w", err)
	}
	if isUsedByConfig {
		return fmt.Errorf("não é possível desativar o certificado pois está em uso em configurações fiscais")
	}

	// Desativar o certificado
	_, err = conn.Exec(ctx, fmt.Sprintf("UPDATE %s.branch_certificates SET is_active = false, updated_at = $1 WHERE id = $2 AND tenant_id = $3", schema), time.Now(), id, tenantID)
	if err != nil {
		return fmt.Errorf("falha ao desativar certificado: %w", err)
	}

	return nil
}

// CountByTenant implementa o método CountByTenant da interface certificate.Repository
func (r *CertificateRepository) CountByTenant(ctx context.Context, tenantID string) (int, error) {
	conn, err := r.db.Acquire(ctx)
	if err != nil {
		return 0, fmt.Errorf("falha ao adquirir conexão: %w", err)
	}
	defer conn.Release()

	// Obter tenant_id do contexto se não fornecido
	if tenantID == "" {
		tenantID = tenant.GetTenantIDFromContext(ctx)
		if tenantID == "" {
			return 0, fmt.Errorf("tenant ID não encontrado no contexto")
		}
	}

	// Configurar search_path para public para acessar a tabela de tenants
	_, err = conn.Exec(ctx, "SET search_path TO public")
	if err != nil {
		return 0, fmt.Errorf("falha ao configurar search_path: %w", err)
	}

	// Obter o schema do tenant
	var schema string
	err = conn.QueryRow(ctx, "SELECT schema FROM tenants WHERE id = $1", tenantID).Scan(&schema)
	if err != nil {
		return 0, fmt.Errorf("falha ao obter schema do tenant: %w", err)
	}

	// Contar certificados
	var count int
	err = conn.QueryRow(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s.branch_certificates WHERE tenant_id = $1", schema), tenantID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("falha ao contar certificados: %w", err)
	}

	return count, nil
}

// CountByBranch implementa o método CountByBranch da interface certificate.Repository
func (r *CertificateRepository) CountByBranch(ctx context.Context, branchID string) (int, error) {
	conn, err := r.db.Acquire(ctx)
	if err != nil {
		return 0, fmt.Errorf("falha ao adquirir conexão: %w", err)
	}
	defer conn.Release()

	// Obter tenant_id do contexto
	tenantID := tenant.GetTenantIDFromContext(ctx)
	if tenantID == "" {
		return 0, fmt.Errorf("tenant ID não encontrado no contexto")
	}

	// Configurar search_path para public para acessar a tabela de tenants
	_, err = conn.Exec(ctx, "SET search_path TO public")
	if err != nil {
		return 0, fmt.Errorf("falha ao configurar search_path: %w", err)
	}

	// Obter o schema do tenant
	var schema string
	err = conn.QueryRow(ctx, "SELECT schema FROM tenants WHERE id = $1", tenantID).Scan(&schema)
	if err != nil {
		return 0, fmt.Errorf("falha ao obter schema do tenant: %w", err)
	}

	// Obter branch_id do contexto se não fornecido
	if branchID == "" {
		branchID = branch.GetBranchID(ctx)
	}

	// Contar certificados da filial
	var count int
	err = conn.QueryRow(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s.branch_certificates WHERE branch_id = $1 AND tenant_id = $2", schema), branchID, tenantID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("falha ao contar certificados: %w", err)
	}

	return count, nil
}

// Exists implementa o método Exists da interface certificate.Repository
func (r *CertificateRepository) Exists(ctx context.Context, id string) (bool, error) {
	conn, err := r.db.Acquire(ctx)
	if err != nil {
		return false, fmt.Errorf("falha ao adquirir conexão: %w", err)
	}
	defer conn.Release()

	// Obter tenant_id do contexto
	tenantID := tenant.GetTenantIDFromContext(ctx)
	if tenantID == "" {
		return false, fmt.Errorf("tenant ID não encontrado no contexto")
	}

	// Configurar search_path para public para acessar a tabela de tenants
	_, err = conn.Exec(ctx, "SET search_path TO public")
	if err != nil {
		return false, fmt.Errorf("falha ao configurar search_path: %w", err)
	}

	// Obter o schema do tenant
	var schema string
	err = conn.QueryRow(ctx, "SELECT schema FROM tenants WHERE id = $1", tenantID).Scan(&schema)
	if err != nil {
		return false, fmt.Errorf("falha ao obter schema do tenant: %w", err)
	}

	// Verificar se o certificado existe
	var exists bool
	err = conn.QueryRow(ctx, fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s.branch_certificates WHERE id = $1 AND tenant_id = $2)", schema), id, tenantID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("falha ao verificar se o certificado existe: %w", err)
	}

	return exists, nil
}

// FindExpiring implementa o método FindExpiring da interface certificate.Repository
func (r *CertificateRepository) FindExpiring(ctx context.Context, daysToExpire int) ([]*certificate.Certificate, error) {
	conn, err := r.db.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("falha ao adquirir conexão: %w", err)
	}
	defer conn.Release()

	// Obter tenant_id do contexto
	tenantID := tenant.GetTenantIDFromContext(ctx)
	if tenantID == "" {
		return nil, fmt.Errorf("tenant ID não encontrado no contexto")
	}

	// Configurar search_path para public para acessar a tabela de tenants
	_, err = conn.Exec(ctx, "SET search_path TO public")
	if err != nil {
		return nil, fmt.Errorf("falha ao configurar search_path: %w", err)
	}

	// Obter o schema do tenant
	var schema string
	err = conn.QueryRow(ctx, "SELECT schema FROM tenants WHERE id = $1", tenantID).Scan(&schema)
	if err != nil {
		return nil, fmt.Errorf("falha ao obter schema do tenant: %w", err)
	}

	// Calcular a data limite
	expirationLimit := time.Now().AddDate(0, 0, daysToExpire)

	// Buscar certificados que expirarão em X dias
	query := fmt.Sprintf(`
		SELECT 
			id, tenant_id, branch_id, name, certificate_data, certificate_path, 
			password, expiration_date, is_active, created_at, updated_at
		FROM %s.branch_certificates
		WHERE tenant_id = $1 AND expiration_date <= $2
		ORDER BY expiration_date
	`, schema)

	rows, err := conn.Query(ctx, query, tenantID, expirationLimit)
	if err != nil {
		return nil, fmt.Errorf("falha ao buscar certificados: %w", err)
	}
	defer rows.Close()

	// Processar os resultados
	certificates := []*certificate.Certificate{}
	for rows.Next() {
		var cert certificate.Certificate
		err = rows.Scan(
			&cert.ID, &cert.TenantID, &cert.BranchID, &cert.Name, &cert.CertificateData,
			&cert.CertificatePath, &cert.Password, &cert.ExpirationDate, &cert.IsActive,
			&cert.CreatedAt, &cert.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("falha ao ler certificado: %w", err)
		}
		certificates = append(certificates, &cert)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("erro ao iterar certificados: %w", err)
	}

	return certificates, nil
}
