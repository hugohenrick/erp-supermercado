-- Tabela para certificados digitais
CREATE TABLE IF NOT EXISTS branch_certificates (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    branch_id UUID NOT NULL REFERENCES branches(id),
    name VARCHAR(100) NOT NULL,
    certificate_data BYTEA,                      -- Dados binários do certificado .pfx
    certificate_path VARCHAR(255),               -- Alternativa: caminho do arquivo
    password VARCHAR(255) NOT NULL,              -- Senha do certificado (deveria ser criptografada)
    expiration_date DATE NOT NULL,               -- Data de validade
    is_active BOOLEAN NOT NULL DEFAULT true,     -- Se é o certificado ativo
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(tenant_id, branch_id, name)
);

-- Tabela para configurações fiscais
CREATE TABLE IF NOT EXISTS fiscal_configurations (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    branch_id UUID NOT NULL REFERENCES branches(id),
    certificate_id UUID REFERENCES branch_certificates(id),
    
    -- Configurações de NFe
    nfe_series VARCHAR(3) NOT NULL DEFAULT '1',
    nfe_next_number INTEGER NOT NULL DEFAULT 1,
    nfe_environment VARCHAR(10) NOT NULL DEFAULT 'homologation', -- production, homologation
    nfe_csc_id VARCHAR(20),                                      -- ID do CSC (para NFCe)
    nfe_csc_token VARCHAR(100),                                  -- Token CSC (para NFCe)
    
    -- Configurações de NFCe
    nfce_series VARCHAR(3) NOT NULL DEFAULT '1',
    nfce_next_number INTEGER NOT NULL DEFAULT 1,
    nfce_environment VARCHAR(10) NOT NULL DEFAULT 'homologation', -- production, homologation
    nfce_csc_id VARCHAR(20),                                      -- ID do CSC (específico para NFCe)
    nfce_csc_token VARCHAR(100),                                  -- Token CSC (específico para NFCe)
    
    -- Configurações Gerais
    fiscal_csc VARCHAR(100),                 -- Código de Segurança do Contribuinte
    fiscal_csc_id VARCHAR(20),               -- ID do CSC
    contingency_enabled BOOLEAN DEFAULT false,
    
    -- Configuração SMTP para envio dos documentos fiscais
    smtp_host VARCHAR(100),
    smtp_port INTEGER,
    smtp_username VARCHAR(100),
    smtp_password VARCHAR(100),
    
    -- Impressão
    print_danfe_mode VARCHAR(20) DEFAULT 'normal', -- normal, contingency, none
    printer_name VARCHAR(100),
    printer_paper_size VARCHAR(20) DEFAULT 'A4',
    
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(tenant_id, branch_id)
);

CREATE INDEX IF NOT EXISTS idx_branch_certificates_tenant_id ON branch_certificates(tenant_id);
CREATE INDEX IF NOT EXISTS idx_branch_certificates_branch_id ON branch_certificates(branch_id);
CREATE INDEX IF NOT EXISTS idx_branch_certificates_is_active ON branch_certificates(is_active);
CREATE INDEX IF NOT EXISTS idx_branch_certificates_expiration_date ON branch_certificates(expiration_date);

CREATE INDEX IF NOT EXISTS idx_fiscal_configurations_tenant_id ON fiscal_configurations(tenant_id);
CREATE INDEX IF NOT EXISTS idx_fiscal_configurations_branch_id ON fiscal_configurations(branch_id);
CREATE INDEX IF NOT EXISTS idx_fiscal_configurations_certificate_id ON fiscal_configurations(certificate_id); 