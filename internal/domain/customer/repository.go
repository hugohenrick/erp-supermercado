package customer

import (
	"context"
)

// Repository define a interface para operações de repositório de clientes
type Repository interface {
	// Create cria um novo cliente
	Create(ctx context.Context, c *Customer) error

	// FindByID busca um cliente pelo ID
	FindByID(ctx context.Context, id string) (*Customer, error)

	// FindByDocument busca um cliente pelo documento (CPF/CNPJ)
	FindByDocument(ctx context.Context, tenantID, document string) (*Customer, error)

	// FindByBranch lista os clientes de uma determinada filial
	FindByBranch(ctx context.Context, branchID string, limit, offset int) ([]*Customer, error)

	// List lista os clientes de um tenant com paginação
	List(ctx context.Context, tenantID string, limit, offset int) ([]*Customer, error)

	// Update atualiza os dados de um cliente existente
	Update(ctx context.Context, c *Customer) error

	// Delete remove um cliente
	Delete(ctx context.Context, id string) error

	// UpdateStatus atualiza o status de um cliente
	UpdateStatus(ctx context.Context, id string, status Status) error

	// CountByTenant conta quantos clientes existem para um tenant
	CountByTenant(ctx context.Context, tenantID string) (int, error)

	// CountByBranch conta quantos clientes existem para uma filial
	CountByBranch(ctx context.Context, branchID string) (int, error)

	// Exists verifica se um cliente existe
	Exists(ctx context.Context, id string) (bool, error)

	// ExistsByDocument verifica se um cliente existe pelo documento
	ExistsByDocument(ctx context.Context, tenantID, document string) (bool, error)

	// FindByName busca clientes pelo nome
	FindByName(ctx context.Context, tenantID, name string, limit, offset int) ([]*Customer, error)

	// FindByType busca clientes pelo tipo
	FindByType(ctx context.Context, tenantID string, customerType CustomerType, limit, offset int) ([]*Customer, error)

	// FindBySalesman busca clientes por vendedor
	FindBySalesman(ctx context.Context, salesmanID string, limit, offset int) ([]*Customer, error)

	// FindByPriceTable busca clientes por tabela de preços
	FindByPriceTable(ctx context.Context, priceTableID string, limit, offset int) ([]*Customer, error)

	// FindByPaymentMethod busca clientes por forma de pagamento
	FindByPaymentMethod(ctx context.Context, paymentMethodID string, limit, offset int) ([]*Customer, error)

	// FindByStatus busca clientes por status
	FindByStatus(ctx context.Context, tenantID string, status Status, limit, offset int) ([]*Customer, error)

	// FindByTaxRegime busca clientes por regime tributário
	FindByTaxRegime(ctx context.Context, tenantID string, taxRegime TaxRegime, limit, offset int) ([]*Customer, error)

	// UpdateCreditLimit atualiza o limite de crédito do cliente
	UpdateCreditLimit(ctx context.Context, id string, creditLimit float64) error

	// UpdatePaymentTerm atualiza o prazo de pagamento do cliente
	UpdatePaymentTerm(ctx context.Context, id string, paymentTerm int) error
}
