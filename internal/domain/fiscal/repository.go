package fiscal

import (
	"context"
)

// Repository define a interface para operações de repositório de configurações fiscais
type Repository interface {
	// Create cria uma nova configuração fiscal
	Create(ctx context.Context, config *Configuration) error

	// FindByID busca uma configuração pelo ID
	FindByID(ctx context.Context, id string) (*Configuration, error)

	// FindByBranch busca a configuração fiscal de uma filial
	FindByBranch(ctx context.Context, branchID string) (*Configuration, error)

	// List lista as configurações de um tenant com paginação
	List(ctx context.Context, tenantID string, limit, offset int) ([]*Configuration, error)

	// Update atualiza os dados de uma configuração existente
	Update(ctx context.Context, config *Configuration) error

	// Delete remove uma configuração
	Delete(ctx context.Context, id string) error

	// UpdateNFeNextNumber atualiza o próximo número de NFe
	UpdateNFeNextNumber(ctx context.Context, id string, nextNumber int) error

	// UpdateNFCeNextNumber atualiza o próximo número de NFCe
	UpdateNFCeNextNumber(ctx context.Context, id string, nextNumber int) error

	// GetAndIncrementNFeNumber obtém e incrementa o próximo número de NFe
	GetAndIncrementNFeNumber(ctx context.Context, branchID string) (int, error)

	// GetAndIncrementNFCeNumber obtém e incrementa o próximo número de NFCe
	GetAndIncrementNFCeNumber(ctx context.Context, branchID string) (int, error)

	// Exists verifica se uma configuração existe
	Exists(ctx context.Context, id string) (bool, error)

	// ExistsByBranch verifica se uma configuração existe para a filial
	ExistsByBranch(ctx context.Context, branchID string) (bool, error)
}
