package user

import (
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Role representa o papel/função do usuário
type Role string

// Status representa o status do usuário
type Status string

// Constantes para Role
const (
	RoleAdmin   Role = "admin"   // Administrador do sistema
	RoleManager Role = "manager" // Gerente de filial
	RoleStaff   Role = "staff"   // Funcionário regular
)

// Constantes para Status
const (
	StatusActive   Status = "active"   // Usuário ativo
	StatusInactive Status = "inactive" // Usuário inativo
	StatusBlocked  Status = "blocked"  // Usuário bloqueado
)

// User representa um usuário do sistema
type User struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	BranchID    string    `json:"branch_id"`
	Name        string    `json:"name"`
	Email       string    `json:"email"`
	Password    string    `json:"-"` // O campo senha não é retornado nas respostas JSON
	Role        Role      `json:"role"`
	Status      Status    `json:"status"`
	LastLoginAt time.Time `json:"last_login_at"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// SetPassword configura a senha do usuário com hash
func (u *User) SetPassword(password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hashedPassword)
	return nil
}

// CheckPassword verifica se a senha fornecida é válida
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}

// IsActive verifica se o usuário está ativo
func (u *User) IsActive() bool {
	return u.Status == StatusActive
}

// IsAdmin verifica se o usuário é um administrador
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

// IsManager verifica se o usuário é um gerente
func (u *User) IsManager() bool {
	return u.Role == RoleManager
}

// HasAccessToTenant verifica se o usuário tem acesso ao tenant especificado
func (u *User) HasAccessToTenant(tenantID string) bool {
	return u.TenantID == tenantID
}

// HasAccessToBranch verifica se o usuário tem acesso à filial especificada
// Administradores têm acesso a todas as filiais do seu tenant
func (u *User) HasAccessToBranch(branchID string) bool {
	// Administradores têm acesso a todas as filiais
	if u.IsAdmin() {
		return true
	}
	// Outros usuários só têm acesso à sua própria filial
	return u.BranchID == branchID
}
