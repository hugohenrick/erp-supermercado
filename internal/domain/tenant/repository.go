package tenant

import (
	"context"
)

// Repository define as operações de persistência para tenants
type Repository interface {
	// Create persiste um novo tenant
	Create(ctx context.Context, tenant *Tenant) error

	// FindByID busca um tenant pelo ID
	FindByID(ctx context.Context, id string) (*Tenant, error)

	// FindByDocument busca um tenant pelo documento (CNPJ)
	FindByDocument(ctx context.Context, document string) (*Tenant, error)

	// Update atualiza um tenant existente
	Update(ctx context.Context, tenant *Tenant) error

	// Delete remove um tenant
	Delete(ctx context.Context, id string) error

	// List retorna uma lista paginada de tenants
	List(ctx context.Context, limit, offset int) ([]*Tenant, error)

	// Count retorna o número total de tenants
	Count(ctx context.Context) (int, error)

	// UpdateStatus atualiza o status de um tenant
	UpdateStatus(ctx context.Context, id string, status Status) error

	// Exists verifica se um tenant existe pelo ID
	Exists(ctx context.Context, id string) (bool, error)
}
