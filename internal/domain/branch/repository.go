package branch

import (
	"context"
)

// Repository define as operações de persistência para filiais
type Repository interface {
	// Create persiste uma nova filial
	Create(ctx context.Context, branch *Branch) error

	// FindByID busca uma filial pelo ID
	FindByID(ctx context.Context, id string) (*Branch, error)

	// FindByTenantAndID busca uma filial pelo ID do tenant e ID da filial
	FindByTenantAndID(ctx context.Context, tenantID, id string) (*Branch, error)

	// FindMainBranch busca a filial principal (matriz) de um tenant
	FindMainBranch(ctx context.Context, tenantID string) (*Branch, error)

	// Update atualiza uma filial existente
	Update(ctx context.Context, branch *Branch) error

	// Delete remove uma filial
	Delete(ctx context.Context, id string) error

	// ListByTenant retorna uma lista paginada de filiais de um tenant
	ListByTenant(ctx context.Context, tenantID string, limit, offset int) ([]*Branch, error)

	// CountByTenant retorna o número total de filiais de um tenant
	CountByTenant(ctx context.Context, tenantID string) (int, error)

	// UpdateStatus atualiza o status de uma filial
	UpdateStatus(ctx context.Context, id string, status Status) error

	// Exists verifica se uma filial existe pelo ID
	Exists(ctx context.Context, id string) (bool, error)
}
