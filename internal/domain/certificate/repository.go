package certificate

import (
	"context"
)

// Repository define a interface para operações de repositório de certificados digitais
type Repository interface {
	// Create cria um novo certificado digital
	Create(ctx context.Context, cert *Certificate) error

	// FindByID busca um certificado pelo ID
	FindByID(ctx context.Context, id string) (*Certificate, error)

	// FindByBranch lista os certificados de uma determinada filial
	FindByBranch(ctx context.Context, branchID string) ([]*Certificate, error)

	// FindActiveCertificate busca o certificado ativo de uma filial
	FindActiveCertificate(ctx context.Context, branchID string) (*Certificate, error)

	// List lista os certificados de um tenant com paginação
	List(ctx context.Context, tenantID string, limit, offset int) ([]*Certificate, error)

	// Update atualiza os dados de um certificado existente
	Update(ctx context.Context, cert *Certificate) error

	// Delete remove um certificado
	Delete(ctx context.Context, id string) error

	// Activate ativa um certificado e desativa os demais da mesma filial
	Activate(ctx context.Context, id string) error

	// Deactivate desativa um certificado
	Deactivate(ctx context.Context, id string) error

	// CountByTenant conta quantos certificados existem para um tenant
	CountByTenant(ctx context.Context, tenantID string) (int, error)

	// CountByBranch conta quantos certificados existem para uma filial
	CountByBranch(ctx context.Context, branchID string) (int, error)

	// Exists verifica se um certificado existe
	Exists(ctx context.Context, id string) (bool, error)

	// FindExpiring retorna certificados que expirarão em X dias
	FindExpiring(ctx context.Context, daysToExpire int) ([]*Certificate, error)
}
