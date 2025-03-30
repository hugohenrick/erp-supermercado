package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/hugohenrick/erp-supermercado/internal/domain/fiscal"
	"github.com/hugohenrick/erp-supermercado/pkg/branch"
	"github.com/hugohenrick/erp-supermercado/pkg/tenant"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// FiscalRepository implementa a interface fiscal.Repository
type FiscalRepository struct {
	db *pgxpool.Pool
}

// NewFiscalRepository cria uma nova instância de FiscalRepository
func NewFiscalRepository(db *pgxpool.Pool) fiscal.Repository {
	return &FiscalRepository{
		db: db,
	}
}

// Create implementa o método Create da interface fiscal.Repository
func (r *FiscalRepository) Create(ctx context.Context, config *fiscal.Configuration) error {
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
	err = conn.QueryRow(ctx, fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s.branches WHERE id = $1)", schema), config.BranchID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("falha ao verificar se a filial existe: %w", err)
	}
	if !exists {
		return fmt.Errorf("filial com ID %s não encontrada", config.BranchID)
	}

	// Verificar se já existe configuração para esta filial
	err = conn.QueryRow(ctx, fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s.fiscal_configurations WHERE branch_id = $1)", schema), config.BranchID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("falha ao verificar configurações existentes: %w", err)
	}
	if exists {
		return fmt.Errorf("já existe uma configuração fiscal para a filial %s", config.BranchID)
	}

	// Verificar se o certificado existe, se fornecido
	if config.CertificateID != "" {
		err = conn.QueryRow(ctx, fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s.branch_certificates WHERE id = $1)", schema), config.CertificateID).Scan(&exists)
		if err != nil {
			return fmt.Errorf("falha ao verificar se o certificado existe: %w", err)
		}
		if !exists {
			return fmt.Errorf("certificado com ID %s não encontrado", config.CertificateID)
		}
	}

	// Inserir a nova configuração fiscal
	query := fmt.Sprintf(`
		INSERT INTO %s.fiscal_configurations (
			id, tenant_id, branch_id, certificate_id, 
			nfe_series, nfe_next_number, nfe_environment, nfe_csc_id, nfe_csc_token,
			nfce_series, nfce_next_number, nfce_environment, nfce_csc_id, nfce_csc_token,
			fiscal_csc, fiscal_csc_id, contingency_enabled,
			smtp_host, smtp_port, smtp_username, smtp_password,
			print_danfe_mode, printer_name, printer_paper_size,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14,
			$15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26
		)
	`, schema)

	_, err = conn.Exec(ctx, query,
		config.ID, config.TenantID, config.BranchID, config.CertificateID,
		config.NFeSeries, config.NFeNextNumber, config.NFeEnvironment, config.NFeCSCID, config.NFeCSCToken,
		config.NFCeSeries, config.NFCeNextNumber, config.NFCeEnvironment, config.NFCeCSCID, config.NFCeCSCToken,
		config.FiscalCSC, config.FiscalCSCID, config.ContingencyEnabled,
		config.SMTPHost, config.SMTPPort, config.SMTPUsername, config.SMTPPassword,
		config.PrintDANFEMode, config.PrinterName, config.PrinterPaperSize,
		config.CreatedAt, config.UpdatedAt)

	if err != nil {
		return fmt.Errorf("falha ao inserir configuração fiscal: %w", err)
	}

	return nil
}

// FindByID implementa o método FindByID da interface fiscal.Repository
func (r *FiscalRepository) FindByID(ctx context.Context, id string) (*fiscal.Configuration, error) {
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

	// Buscar a configuração
	query := fmt.Sprintf(`
		SELECT 
			id, tenant_id, branch_id, certificate_id, 
			nfe_series, nfe_next_number, nfe_environment, nfe_csc_id, nfe_csc_token,
			nfce_series, nfce_next_number, nfce_environment, nfce_csc_id, nfce_csc_token,
			fiscal_csc, fiscal_csc_id, contingency_enabled,
			smtp_host, smtp_port, smtp_username, smtp_password,
			print_danfe_mode, printer_name, printer_paper_size,
			created_at, updated_at
		FROM %s.fiscal_configurations
		WHERE id = $1 AND tenant_id = $2
	`, schema)

	var config fiscal.Configuration
	err = conn.QueryRow(ctx, query, id, tenantID).Scan(
		&config.ID, &config.TenantID, &config.BranchID, &config.CertificateID,
		&config.NFeSeries, &config.NFeNextNumber, &config.NFeEnvironment, &config.NFeCSCID, &config.NFeCSCToken,
		&config.NFCeSeries, &config.NFCeNextNumber, &config.NFCeEnvironment, &config.NFCeCSCID, &config.NFCeCSCToken,
		&config.FiscalCSC, &config.FiscalCSCID, &config.ContingencyEnabled,
		&config.SMTPHost, &config.SMTPPort, &config.SMTPUsername, &config.SMTPPassword,
		&config.PrintDANFEMode, &config.PrinterName, &config.PrinterPaperSize,
		&config.CreatedAt, &config.UpdatedAt)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("configuração fiscal com ID %s não encontrada", id)
		}
		return nil, fmt.Errorf("falha ao buscar configuração fiscal: %w", err)
	}

	return &config, nil
}

// FindByBranch implementa o método FindByBranch da interface fiscal.Repository
func (r *FiscalRepository) FindByBranch(ctx context.Context, branchID string) (*fiscal.Configuration, error) {
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

	// Buscar a configuração
	query := fmt.Sprintf(`
		SELECT 
			id, tenant_id, branch_id, certificate_id, 
			nfe_series, nfe_next_number, nfe_environment, nfe_csc_id, nfe_csc_token,
			nfce_series, nfce_next_number, nfce_environment, nfce_csc_id, nfce_csc_token,
			fiscal_csc, fiscal_csc_id, contingency_enabled,
			smtp_host, smtp_port, smtp_username, smtp_password,
			print_danfe_mode, printer_name, printer_paper_size,
			created_at, updated_at
		FROM %s.fiscal_configurations
		WHERE branch_id = $1 AND tenant_id = $2
	`, schema)

	var config fiscal.Configuration
	err = conn.QueryRow(ctx, query, branchID, tenantID).Scan(
		&config.ID, &config.TenantID, &config.BranchID, &config.CertificateID,
		&config.NFeSeries, &config.NFeNextNumber, &config.NFeEnvironment, &config.NFeCSCID, &config.NFeCSCToken,
		&config.NFCeSeries, &config.NFCeNextNumber, &config.NFCeEnvironment, &config.NFCeCSCID, &config.NFCeCSCToken,
		&config.FiscalCSC, &config.FiscalCSCID, &config.ContingencyEnabled,
		&config.SMTPHost, &config.SMTPPort, &config.SMTPUsername, &config.SMTPPassword,
		&config.PrintDANFEMode, &config.PrinterName, &config.PrinterPaperSize,
		&config.CreatedAt, &config.UpdatedAt)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("configuração fiscal não encontrada para a filial %s", branchID)
		}
		return nil, fmt.Errorf("falha ao buscar configuração fiscal: %w", err)
	}

	return &config, nil
}

// List implementa o método List da interface fiscal.Repository
func (r *FiscalRepository) List(ctx context.Context, tenantID string, limit, offset int) ([]*fiscal.Configuration, error) {
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

	// Buscar as configurações fiscais com paginação
	query := fmt.Sprintf(`
		SELECT 
			id, tenant_id, branch_id, certificate_id, 
			nfe_series, nfe_next_number, nfe_environment, nfe_csc_id, nfe_csc_token,
			nfce_series, nfce_next_number, nfce_environment, nfce_csc_id, nfce_csc_token,
			fiscal_csc, fiscal_csc_id, contingency_enabled,
			smtp_host, smtp_port, smtp_username, smtp_password,
			print_danfe_mode, printer_name, printer_paper_size,
			created_at, updated_at
		FROM %s.fiscal_configurations
		WHERE tenant_id = $1
		ORDER BY branch_id
		LIMIT $2 OFFSET $3
	`, schema)

	rows, err := conn.Query(ctx, query, tenantID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("falha ao buscar configurações fiscais: %w", err)
	}
	defer rows.Close()

	// Processar os resultados
	configs := []*fiscal.Configuration{}
	for rows.Next() {
		var config fiscal.Configuration
		err = rows.Scan(
			&config.ID, &config.TenantID, &config.BranchID, &config.CertificateID,
			&config.NFeSeries, &config.NFeNextNumber, &config.NFeEnvironment, &config.NFeCSCID, &config.NFeCSCToken,
			&config.NFCeSeries, &config.NFCeNextNumber, &config.NFCeEnvironment, &config.NFCeCSCID, &config.NFCeCSCToken,
			&config.FiscalCSC, &config.FiscalCSCID, &config.ContingencyEnabled,
			&config.SMTPHost, &config.SMTPPort, &config.SMTPUsername, &config.SMTPPassword,
			&config.PrintDANFEMode, &config.PrinterName, &config.PrinterPaperSize,
			&config.CreatedAt, &config.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("falha ao ler configuração fiscal: %w", err)
		}
		configs = append(configs, &config)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("erro ao iterar configurações fiscais: %w", err)
	}

	return configs, nil
}

// Update implementa o método Update da interface fiscal.Repository
func (r *FiscalRepository) Update(ctx context.Context, config *fiscal.Configuration) error {
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

	// Verificar se a configuração existe
	var exists bool
	err = conn.QueryRow(ctx, fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s.fiscal_configurations WHERE id = $1 AND tenant_id = $2)", schema), config.ID, tenantID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("falha ao verificar se a configuração existe: %w", err)
	}
	if !exists {
		return fmt.Errorf("configuração fiscal com ID %s não encontrada", config.ID)
	}

	// Verificar se o certificado existe, se fornecido
	if config.CertificateID != "" {
		err = conn.QueryRow(ctx, fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s.branch_certificates WHERE id = $1)", schema), config.CertificateID).Scan(&exists)
		if err != nil {
			return fmt.Errorf("falha ao verificar se o certificado existe: %w", err)
		}
		if !exists {
			return fmt.Errorf("certificado com ID %s não encontrado", config.CertificateID)
		}
	}

	// Atualizar a configuração
	query := fmt.Sprintf(`
		UPDATE %s.fiscal_configurations SET
			certificate_id = $1, 
			nfe_series = $2, nfe_next_number = $3, nfe_environment = $4, nfe_csc_id = $5, nfe_csc_token = $6,
			nfce_series = $7, nfce_next_number = $8, nfce_environment = $9, nfce_csc_id = $10, nfce_csc_token = $11,
			fiscal_csc = $12, fiscal_csc_id = $13, contingency_enabled = $14,
			smtp_host = $15, smtp_port = $16, smtp_username = $17, smtp_password = $18,
			print_danfe_mode = $19, printer_name = $20, printer_paper_size = $21,
			updated_at = $22
		WHERE id = $23 AND tenant_id = $24
	`, schema)

	_, err = conn.Exec(ctx, query,
		config.CertificateID,
		config.NFeSeries, config.NFeNextNumber, config.NFeEnvironment, config.NFeCSCID, config.NFeCSCToken,
		config.NFCeSeries, config.NFCeNextNumber, config.NFCeEnvironment, config.NFCeCSCID, config.NFCeCSCToken,
		config.FiscalCSC, config.FiscalCSCID, config.ContingencyEnabled,
		config.SMTPHost, config.SMTPPort, config.SMTPUsername, config.SMTPPassword,
		config.PrintDANFEMode, config.PrinterName, config.PrinterPaperSize,
		time.Now(),
		config.ID, tenantID)

	if err != nil {
		return fmt.Errorf("falha ao atualizar configuração fiscal: %w", err)
	}

	return nil
}

// Delete implementa o método Delete da interface fiscal.Repository
func (r *FiscalRepository) Delete(ctx context.Context, id string) error {
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

	// Excluir a configuração fiscal
	_, err = conn.Exec(ctx, fmt.Sprintf("DELETE FROM %s.fiscal_configurations WHERE id = $1 AND tenant_id = $2", schema), id, tenantID)
	if err != nil {
		return fmt.Errorf("falha ao excluir configuração fiscal: %w", err)
	}

	return nil
}

// UpdateNFeNextNumber implementa o método UpdateNFeNextNumber da interface fiscal.Repository
func (r *FiscalRepository) UpdateNFeNextNumber(ctx context.Context, id string, nextNumber int) error {
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

	// Atualizar o próximo número de NFe
	query := fmt.Sprintf(`
		UPDATE %s.fiscal_configurations SET
			nfe_next_number = $1,
			updated_at = $2
		WHERE id = $3 AND tenant_id = $4
	`, schema)

	_, err = conn.Exec(ctx, query, nextNumber, time.Now(), id, tenantID)
	if err != nil {
		return fmt.Errorf("falha ao atualizar próximo número de NFe: %w", err)
	}

	return nil
}

// UpdateNFCeNextNumber implementa o método UpdateNFCeNextNumber da interface fiscal.Repository
func (r *FiscalRepository) UpdateNFCeNextNumber(ctx context.Context, id string, nextNumber int) error {
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

	// Atualizar o próximo número de NFCe
	query := fmt.Sprintf(`
		UPDATE %s.fiscal_configurations SET
			nfce_next_number = $1,
			updated_at = $2
		WHERE id = $3 AND tenant_id = $4
	`, schema)

	_, err = conn.Exec(ctx, query, nextNumber, time.Now(), id, tenantID)
	if err != nil {
		return fmt.Errorf("falha ao atualizar próximo número de NFCe: %w", err)
	}

	return nil
}

// GetAndIncrementNFeNumber implementa o método GetAndIncrementNFeNumber da interface fiscal.Repository
func (r *FiscalRepository) GetAndIncrementNFeNumber(ctx context.Context, branchID string) (int, error) {
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

	// Iniciar uma transação
	tx, err := conn.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("falha ao iniciar transação: %w", err)
	}
	defer tx.Rollback(ctx)

	// Obter e incrementar o número de NFe em uma única operação
	var currentNumber int
	query := fmt.Sprintf(`
		UPDATE %s.fiscal_configurations SET
			nfe_next_number = nfe_next_number + 1,
			updated_at = $1
		WHERE branch_id = $2 AND tenant_id = $3
		RETURNING nfe_next_number - 1
	`, schema)

	err = tx.QueryRow(ctx, query, time.Now(), branchID, tenantID).Scan(&currentNumber)
	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, fmt.Errorf("configuração fiscal não encontrada para a filial %s", branchID)
		}
		return 0, fmt.Errorf("falha ao obter e incrementar número de NFe: %w", err)
	}

	// Confirmar a transação
	if err = tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("falha ao confirmar transação: %w", err)
	}

	return currentNumber, nil
}

// GetAndIncrementNFCeNumber implementa o método GetAndIncrementNFCeNumber da interface fiscal.Repository
func (r *FiscalRepository) GetAndIncrementNFCeNumber(ctx context.Context, branchID string) (int, error) {
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

	// Iniciar uma transação
	tx, err := conn.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("falha ao iniciar transação: %w", err)
	}
	defer tx.Rollback(ctx)

	// Obter e incrementar o número de NFCe em uma única operação
	var currentNumber int
	query := fmt.Sprintf(`
		UPDATE %s.fiscal_configurations SET
			nfce_next_number = nfce_next_number + 1,
			updated_at = $1
		WHERE branch_id = $2 AND tenant_id = $3
		RETURNING nfce_next_number - 1
	`, schema)

	err = tx.QueryRow(ctx, query, time.Now(), branchID, tenantID).Scan(&currentNumber)
	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, fmt.Errorf("configuração fiscal não encontrada para a filial %s", branchID)
		}
		return 0, fmt.Errorf("falha ao obter e incrementar número de NFCe: %w", err)
	}

	// Confirmar a transação
	if err = tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("falha ao confirmar transação: %w", err)
	}

	return currentNumber, nil
}

// Exists implementa o método Exists da interface fiscal.Repository
func (r *FiscalRepository) Exists(ctx context.Context, id string) (bool, error) {
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

	// Verificar se a configuração existe
	var exists bool
	err = conn.QueryRow(ctx, fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s.fiscal_configurations WHERE id = $1 AND tenant_id = $2)", schema), id, tenantID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("falha ao verificar se a configuração existe: %w", err)
	}

	return exists, nil
}

// ExistsByBranch implementa o método ExistsByBranch da interface fiscal.Repository
func (r *FiscalRepository) ExistsByBranch(ctx context.Context, branchID string) (bool, error) {
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

	// Obter branch_id do contexto se não fornecido
	if branchID == "" {
		branchID = branch.GetBranchID(ctx)
	}

	// Verificar se existe configuração para a filial
	var exists bool
	err = conn.QueryRow(ctx, fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s.fiscal_configurations WHERE branch_id = $1 AND tenant_id = $2)", schema), branchID, tenantID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("falha ao verificar se existe configuração para a filial: %w", err)
	}

	return exists, nil
}
