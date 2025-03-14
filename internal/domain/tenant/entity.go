package tenant

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrEmptyID         = errors.New("id não pode ser vazio")
	ErrEmptyName       = errors.New("nome não pode ser vazio")
	ErrEmptyDocument   = errors.New("documento não pode ser vazio")
	ErrInvalidDocument = errors.New("documento inválido")
	ErrInvalidTenantID = errors.New("ID de tenant inválido")
	ErrTenantNotActive = errors.New("tenant não está ativo")
)

// Status representa o estado do tenant
type Status string

const (
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
	StatusBlocked  Status = "blocked"
)

// Tenant representa uma empresa no sistema multi-tenant
type Tenant struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Document    string    `json:"document"` // CNPJ da empresa
	Email       string    `json:"email"`
	Phone       string    `json:"phone"`
	Status      Status    `json:"status"`
	Schema      string    `json:"schema"`       // Nome do schema no banco de dados
	PlanType    string    `json:"plan_type"`    // Tipo de plano contratado
	MaxBranches int       `json:"max_branches"` // Número máximo de filiais permitidas
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// NewTenant cria um novo tenant
func NewTenant(name, document, email, phone, planType string, maxBranches int) (*Tenant, error) {
	if name == "" {
		return nil, ErrEmptyName
	}

	if document == "" {
		return nil, ErrEmptyDocument
	}

	// Aqui poderia haver validação de CNPJ

	id := uuid.New().String()
	schema := "tenant_" + id[:8] // Criamos um schema baseado no ID

	return &Tenant{
		ID:          id,
		Name:        name,
		Document:    document,
		Email:       email,
		Phone:       phone,
		Status:      StatusActive,
		Schema:      schema,
		PlanType:    planType,
		MaxBranches: maxBranches,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}, nil
}

// IsActive verifica se o tenant está ativo
func (t *Tenant) IsActive() bool {
	return t.Status == StatusActive
}

// Activate ativa o tenant
func (t *Tenant) Activate() {
	t.Status = StatusActive
	t.UpdatedAt = time.Now()
}

// Deactivate desativa o tenant
func (t *Tenant) Deactivate() {
	t.Status = StatusInactive
	t.UpdatedAt = time.Now()
}

// Block bloqueia o tenant
func (t *Tenant) Block() {
	t.Status = StatusBlocked
	t.UpdatedAt = time.Now()
}

// ChangePlan altera o plano do tenant
func (t *Tenant) ChangePlan(planType string, maxBranches int) {
	t.PlanType = planType
	t.MaxBranches = maxBranches
	t.UpdatedAt = time.Now()
}

// Update atualiza os dados do tenant
func (t *Tenant) Update(name, email, phone string) error {
	if name == "" {
		return ErrEmptyName
	}

	t.Name = name
	t.Email = email
	t.Phone = phone
	t.UpdatedAt = time.Now()
	return nil
}
