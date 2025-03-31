package repository

import (
	"github.com/hugohenrick/erp-supermercado/pkg/domain"
)

// UserRepository define os métodos para acesso a dados de usuários
type UserRepository interface {
	// Create cria um novo usuário
	Create(tenantID string, user *domain.User) error

	// Update atualiza um usuário existente
	Update(tenantID string, user *domain.User) error

	// Delete exclui um usuário pelo ID
	Delete(tenantID string, userID string) error

	// FindByID busca um usuário pelo ID
	FindByID(tenantID string, userID string) (*domain.User, error)

	// FindByEmail busca um usuário pelo email
	FindByEmail(tenantID string, email string) (*domain.User, error)

	// FindByName busca usuários pelo nome (pode retornar múltiplos)
	FindByName(tenantID string, name string) ([]*domain.User, error)

	// FindAll retorna todos os usuários de um tenant
	FindAll(tenantID string) ([]*domain.User, error)
}

// ProductRepository define os métodos para acesso a dados de produtos
type ProductRepository interface {
	// Create cria um novo produto
	Create(tenantID string, product *domain.Product) error

	// Update atualiza um produto existente
	Update(tenantID string, product *domain.Product) error

	// Delete exclui um produto pelo ID
	Delete(tenantID string, productID string) error

	// FindByID busca um produto pelo ID
	FindByID(tenantID string, productID string) (*domain.Product, error)

	// FindBySKU busca um produto pelo SKU
	FindBySKU(tenantID string, sku string) (*domain.Product, error)

	// FindByName busca produtos pelo nome (pode retornar múltiplos)
	FindByName(tenantID string, name string) ([]*domain.Product, error)

	// FindByCategory busca produtos por categoria
	FindByCategory(tenantID string, category string) ([]*domain.Product, error)

	// FindAll retorna todos os produtos de um tenant
	FindAll(tenantID string) ([]*domain.Product, error)

	// UpdateStock atualiza o estoque de um produto
	UpdateStock(tenantID string, productID string, quantity int) error
}

// CustomerRepository define os métodos para acesso a dados de clientes
type CustomerRepository interface {
	// Create cria um novo cliente
	Create(tenantID string, customer *domain.Customer) error

	// Update atualiza um cliente existente
	Update(tenantID string, customer *domain.Customer) error

	// Delete exclui um cliente pelo ID
	Delete(tenantID string, customerID string) error

	// FindByID busca um cliente pelo ID
	FindByID(tenantID string, customerID string) (*domain.Customer, error)

	// FindByDocument busca um cliente pelo documento (CPF/CNPJ)
	FindByDocument(tenantID string, document string) (*domain.Customer, error)

	// FindByEmail busca um cliente pelo email
	FindByEmail(tenantID string, email string) (*domain.Customer, error)

	// FindByName busca clientes pelo nome (pode retornar múltiplos)
	FindByName(tenantID string, name string) ([]*domain.Customer, error)

	// FindAll retorna todos os clientes de um tenant
	FindAll(tenantID string) ([]*domain.Customer, error)
}

// SaleRepository define os métodos para acesso a dados de vendas
type SaleRepository interface {
	// Create cria uma nova venda
	Create(tenantID string, sale *domain.Sale) error

	// Update atualiza uma venda existente
	Update(tenantID string, sale *domain.Sale) error

	// Delete exclui uma venda pelo ID
	Delete(tenantID string, saleID string) error

	// FindByID busca uma venda pelo ID
	FindByID(tenantID string, saleID string) (*domain.Sale, error)

	// FindByCustomer busca vendas por cliente
	FindByCustomer(tenantID string, customerID string) ([]*domain.Sale, error)

	// FindByDateRange busca vendas em um intervalo de datas
	FindByDateRange(tenantID string, startDate, endDate string) ([]*domain.Sale, error)

	// FindAll retorna todas as vendas de um tenant
	FindAll(tenantID string) ([]*domain.Sale, error)

	// AddSaleItem adiciona um item a uma venda
	AddSaleItem(tenantID string, saleID string, item *domain.SaleItem) error

	// UpdateSaleItem atualiza um item de venda
	UpdateSaleItem(tenantID string, item *domain.SaleItem) error

	// DeleteSaleItem remove um item de venda
	DeleteSaleItem(tenantID string, saleID string, itemID string) error

	// FindSaleItems busca todos os itens de uma venda
	FindSaleItems(tenantID string, saleID string) ([]*domain.SaleItem, error)

	// AddPayment adiciona um pagamento a uma venda
	AddPayment(tenantID string, saleID string, payment *domain.Payment) error

	// UpdatePayment atualiza um pagamento
	UpdatePayment(tenantID string, payment *domain.Payment) error

	// DeletePayment remove um pagamento
	DeletePayment(tenantID string, saleID string, paymentID string) error

	// FindPayments busca todos os pagamentos de uma venda
	FindPayments(tenantID string, saleID string) ([]*domain.Payment, error)
}

// SupplierRepository define os métodos para acesso a dados de fornecedores
type SupplierRepository interface {
	// Create cria um novo fornecedor
	Create(tenantID string, supplier *domain.Supplier) error

	// Update atualiza um fornecedor existente
	Update(tenantID string, supplier *domain.Supplier) error

	// Delete exclui um fornecedor pelo ID
	Delete(tenantID string, supplierID string) error

	// FindByID busca um fornecedor pelo ID
	FindByID(tenantID string, supplierID string) (*domain.Supplier, error)

	// FindByDocument busca um fornecedor pelo documento (CNPJ)
	FindByDocument(tenantID string, document string) (*domain.Supplier, error)

	// FindByName busca fornecedores pelo nome (pode retornar múltiplos)
	FindByName(tenantID string, name string) ([]*domain.Supplier, error)

	// FindAll retorna todos os fornecedores de um tenant
	FindAll(tenantID string) ([]*domain.Supplier, error)
}

// PurchaseRepository define os métodos para acesso a dados de compras
type PurchaseRepository interface {
	// Create cria uma nova ordem de compra
	Create(tenantID string, purchase *domain.PurchaseOrder) error

	// Update atualiza uma ordem de compra existente
	Update(tenantID string, purchase *domain.PurchaseOrder) error

	// Delete exclui uma ordem de compra pelo ID
	Delete(tenantID string, purchaseID string) error

	// FindByID busca uma ordem de compra pelo ID
	FindByID(tenantID string, purchaseID string) (*domain.PurchaseOrder, error)

	// FindBySupplier busca ordens de compra por fornecedor
	FindBySupplier(tenantID string, supplierID string) ([]*domain.PurchaseOrder, error)

	// FindByDateRange busca ordens de compra em um intervalo de datas
	FindByDateRange(tenantID string, startDate, endDate string) ([]*domain.PurchaseOrder, error)

	// FindAll retorna todas as ordens de compra de um tenant
	FindAll(tenantID string) ([]*domain.PurchaseOrder, error)

	// AddPurchaseItem adiciona um item a uma ordem de compra
	AddPurchaseItem(tenantID string, purchaseID string, item *domain.PurchaseItem) error

	// UpdatePurchaseItem atualiza um item de ordem de compra
	UpdatePurchaseItem(tenantID string, item *domain.PurchaseItem) error

	// DeletePurchaseItem remove um item de ordem de compra
	DeletePurchaseItem(tenantID string, purchaseID string, itemID string) error

	// FindPurchaseItems busca todos os itens de uma ordem de compra
	FindPurchaseItems(tenantID string, purchaseID string) ([]*domain.PurchaseItem, error)

	// ReceivePurchase marca uma ordem de compra como recebida
	ReceivePurchase(tenantID string, purchaseID string, receivedItems []*domain.PurchaseItem) error
}
