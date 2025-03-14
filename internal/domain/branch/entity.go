package branch

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrEmptyID         = errors.New("id não pode ser vazio")
	ErrEmptyName       = errors.New("nome não pode ser vazio")
	ErrEmptyTenantID   = errors.New("ID do tenant não pode ser vazio")
	ErrInvalidBranchID = errors.New("ID de filial inválido")
	ErrBranchNotActive = errors.New("filial não está ativa")
)

// Status representa o estado da filial
type Status string

const (
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
	StatusBlocked  Status = "blocked"
)

// BranchType representa o tipo de filial
type BranchType string

const (
	TypeHeadquarters BranchType = "headquarters" // Matriz
	TypeBranch       BranchType = "branch"       // Filial comum
	TypeVirtual      BranchType = "virtual"      // Loja virtual/e-commerce
	TypeWarehouse    BranchType = "warehouse"    // Apenas depósito
)

// Branch representa uma filial no sistema
type Branch struct {
	ID        string     `json:"id"`
	TenantID  string     `json:"tenant_id"` // ID do tenant ao qual a filial pertence
	Name      string     `json:"name"`
	Code      string     `json:"code"` // Código interno da filial
	Type      BranchType `json:"type"`
	Document  string     `json:"document"` // CNPJ da filial
	Address   Address    `json:"address"`
	Phone     string     `json:"phone"`
	Email     string     `json:"email"`
	Status    Status     `json:"status"`
	IsMain    bool       `json:"is_main"` // Indica se é a matriz
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// Address representa o endereço da filial
type Address struct {
	Street     string `json:"street"`
	Number     string `json:"number"`
	Complement string `json:"complement"`
	District   string `json:"district"`
	City       string `json:"city"`
	State      string `json:"state"`
	ZipCode    string `json:"zip_code"`
	Country    string `json:"country"`
}

// NewBranch cria uma nova filial
func NewBranch(
	tenantID, name, code string,
	branchType BranchType,
	document string,
	address Address,
	phone, email string,
	isMain bool,
) (*Branch, error) {
	if tenantID == "" {
		return nil, ErrEmptyTenantID
	}

	if name == "" {
		return nil, ErrEmptyName
	}

	return &Branch{
		ID:        uuid.New().String(),
		TenantID:  tenantID,
		Name:      name,
		Code:      code,
		Type:      branchType,
		Document:  document,
		Address:   address,
		Phone:     phone,
		Email:     email,
		Status:    StatusActive,
		IsMain:    isMain,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

// IsActive verifica se a filial está ativa
func (b *Branch) IsActive() bool {
	return b.Status == StatusActive
}

// Activate ativa a filial
func (b *Branch) Activate() {
	b.Status = StatusActive
	b.UpdatedAt = time.Now()
}

// Deactivate desativa a filial
func (b *Branch) Deactivate() {
	b.Status = StatusInactive
	b.UpdatedAt = time.Now()
}

// Update atualiza os dados da filial
func (b *Branch) Update(name, code, phone, email string, address Address) error {
	if name == "" {
		return ErrEmptyName
	}

	b.Name = name
	b.Code = code
	b.Phone = phone
	b.Email = email
	b.Address = address
	b.UpdatedAt = time.Now()
	return nil
}
