package user

import (
	"context"
)

// Repository define a interface para operações de repositório de usuários
type Repository interface {
	// Create cria um novo usuário
	Create(ctx context.Context, u *User) error

	// FindByID busca um usuário pelo ID
	FindByID(ctx context.Context, id string) (*User, error)

	// FindByEmail busca um usuário pelo email dentro de um tenant
	FindByEmail(ctx context.Context, tenantID, email string) (*User, error)

	// FindByBranch lista os usuários de uma determinada filial
	FindByBranch(ctx context.Context, branchID string, limit, offset int) ([]*User, error)

	// List lista os usuários de um tenant com paginação
	List(ctx context.Context, tenantID string, limit, offset int) ([]*User, error)

	// Update atualiza os dados de um usuário existente
	Update(ctx context.Context, u *User) error

	// Delete remove um usuário do sistema
	Delete(ctx context.Context, id string) error

	// UpdateStatus atualiza o status de um usuário
	UpdateStatus(ctx context.Context, id string, status Status) error

	// UpdatePassword atualiza a senha de um usuário
	UpdatePassword(ctx context.Context, id, hashedPassword string) error

	// UpdateLastLogin atualiza o timestamp de último login do usuário
	UpdateLastLogin(ctx context.Context, id string) error

	// CountByTenant conta quantos usuários existem para um tenant
	CountByTenant(ctx context.Context, tenantID string) (int, error)

	// CountByBranch conta quantos usuários existem para uma filial
	CountByBranch(ctx context.Context, branchID string) (int, error)

	// Exists verifica se um usuário existe
	Exists(ctx context.Context, id string) (bool, error)
}
