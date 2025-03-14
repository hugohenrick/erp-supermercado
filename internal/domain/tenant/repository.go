package tenant

import (
	"context"
)

// Repository define a interface para operações de repositório de tenants
type Repository interface {
	// Create cria um novo tenant
	Create(ctx context.Context, t *Tenant) error

	// FindByID busca um tenant pelo ID
	FindByID(ctx context.Context, id string) (*Tenant, error)

	// FindByDocument busca um tenant pelo documento
	FindByDocument(ctx context.Context, document string) (*Tenant, error)

	// List lista tenants com paginação
	List(ctx context.Context, limit, offset int) ([]*Tenant, error)

	// Update atualiza os dados de um tenant existente
	Update(ctx context.Context, t *Tenant) error

	// Delete remove um tenant
	Delete(ctx context.Context, id string) error

	// UpdateStatus atualiza o status de um tenant
	UpdateStatus(ctx context.Context, id string, status Status) error

	// Count conta quantos tenants existem
	Count(ctx context.Context) (int, error)

	// FindByNameLike busca tenants pelo nome
	FindByNameLike(ctx context.Context, name string, limit, offset int) ([]*Tenant, error)

	// Exists verifica se um tenant existe
	Exists(ctx context.Context, id string) (bool, error)

	// ExistsByDocument verifica se um tenant existe pelo documento
	ExistsByDocument(ctx context.Context, document string) (bool, error)
}

// BranchRepository define a interface para operações de repositório de filiais
type BranchRepository interface {
	// Create cria uma nova filial
	Create(ctx context.Context, b *Branch) error

	// FindByID busca uma filial pelo ID
	FindByID(ctx context.Context, id string) (*Branch, error)

	// List lista filiais de um tenant com paginação
	List(ctx context.Context, tenantID string, limit, offset int) ([]*Branch, error)

	// Update atualiza os dados de uma filial existente
	Update(ctx context.Context, b *Branch) error

	// Delete remove uma filial
	Delete(ctx context.Context, id string) error

	// UpdateStatus atualiza o status de uma filial
	UpdateStatus(ctx context.Context, id string, status Status) error

	// FindMainBranch busca a filial principal de um tenant
	FindMainBranch(ctx context.Context, tenantID string) (*Branch, error)

	// Count conta quantas filiais existem para um tenant
	Count(ctx context.Context, tenantID string) (int, error)

	// Exists verifica se uma filial existe
	Exists(ctx context.Context, id string) (bool, error)
}
