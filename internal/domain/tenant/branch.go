package tenant

import (
	"time"
)

// Branch representa uma filial de um tenant
type Branch struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	Name        string    `json:"name"`
	Document    string    `json:"document"` // CNPJ da filial
	IsMain      bool      `json:"is_main"`  // Indica se é a filial principal
	Address     Address   `json:"address"`
	Phone       string    `json:"phone"`
	Email       string    `json:"email"`
	Status      Status    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// IsActive verifica se a filial está ativa
func (b *Branch) IsActive() bool {
	return b.Status == StatusActive
}

// NewBranch cria uma nova instância de Branch
func NewBranch(id, tenantID, name, document string, isMain bool, address Address, phone, email string) *Branch {
	now := time.Now()
	return &Branch{
		ID:        id,
		TenantID:  tenantID,
		Name:      name,
		Document:  document,
		IsMain:    isMain,
		Address:   address,
		Phone:     phone,
		Email:     email,
		Status:    StatusActive, // Por padrão, a filial é criada ativa
		CreatedAt: now,
		UpdatedAt: now,
	}
} 