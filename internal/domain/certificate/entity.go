package certificate

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Certificado digital vinculado a uma filial
type Certificate struct {
	ID              string    `json:"id"`
	TenantID        string    `json:"tenant_id"`
	BranchID        string    `json:"branch_id"`
	Name            string    `json:"name"`
	CertificateData []byte    `json:"-"` // Não expor ao serializar para JSON
	CertificatePath string    `json:"certificate_path"`
	Password        string    `json:"-"` // Não expor ao serializar para JSON
	ExpirationDate  time.Time `json:"expiration_date"`
	IsActive        bool      `json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// NewCertificate cria um novo certificado digital
func NewCertificate(tenantID, branchID, name string, expirationDate time.Time) (*Certificate, error) {
	if tenantID == "" {
		return nil, errors.New("tenant ID é obrigatório")
	}
	if branchID == "" {
		return nil, errors.New("branch ID é obrigatório")
	}
	if name == "" {
		return nil, errors.New("nome do certificado é obrigatório")
	}
	if expirationDate.Before(time.Now()) {
		return nil, errors.New("data de validade do certificado já passou")
	}

	now := time.Now()
	return &Certificate{
		ID:             uuid.New().String(),
		TenantID:       tenantID,
		BranchID:       branchID,
		Name:           name,
		ExpirationDate: expirationDate,
		IsActive:       true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

// StoreCertificateData armazena os dados binários do certificado
func (c *Certificate) StoreCertificateData(data []byte, password string) error {
	if len(data) == 0 {
		return errors.New("dados do certificado não podem estar vazios")
	}
	if password == "" {
		return errors.New("senha do certificado é obrigatória")
	}

	c.CertificateData = data
	c.Password = password
	c.UpdatedAt = time.Now()
	return nil
}

// StoreCertificatePath armazena o caminho do arquivo do certificado
func (c *Certificate) StoreCertificatePath(path string, password string) error {
	if path == "" {
		return errors.New("caminho do certificado não pode estar vazio")
	}
	if password == "" {
		return errors.New("senha do certificado é obrigatória")
	}

	c.CertificatePath = path
	c.Password = password
	c.UpdatedAt = time.Now()
	return nil
}

// Activate ativa o certificado
func (c *Certificate) Activate() {
	c.IsActive = true
	c.UpdatedAt = time.Now()
}

// Deactivate desativa o certificado
func (c *Certificate) Deactivate() {
	c.IsActive = false
	c.UpdatedAt = time.Now()
}

// IsExpired verifica se o certificado está expirado
func (c *Certificate) IsExpired() bool {
	return time.Now().After(c.ExpirationDate)
}

// RenewExpiration atualiza a data de validade do certificado
func (c *Certificate) RenewExpiration(newExpiration time.Time) error {
	if newExpiration.Before(time.Now()) {
		return errors.New("nova data de validade deve ser no futuro")
	}
	c.ExpirationDate = newExpiration
	c.UpdatedAt = time.Now()
	return nil
}
