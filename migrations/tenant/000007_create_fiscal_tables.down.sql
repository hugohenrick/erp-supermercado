-- Remover índices da tabela fiscal_configurations
DROP INDEX IF EXISTS idx_fiscal_configurations_certificate_id;
DROP INDEX IF EXISTS idx_fiscal_configurations_branch_id;
DROP INDEX IF EXISTS idx_fiscal_configurations_tenant_id;

-- Remover índices da tabela branch_certificates
DROP INDEX IF EXISTS idx_branch_certificates_expiration_date;
DROP INDEX IF EXISTS idx_branch_certificates_is_active;
DROP INDEX IF EXISTS idx_branch_certificates_branch_id;
DROP INDEX IF EXISTS idx_branch_certificates_tenant_id;

-- Remover tabelas
DROP TABLE IF EXISTS fiscal_configurations;
DROP TABLE IF EXISTS branch_certificates; 