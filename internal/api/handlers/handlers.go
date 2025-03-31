package handlers

import (
	"github.com/hugohenrick/erp-supermercado/pkg/logger"
	"github.com/hugohenrick/erp-supermercado/pkg/repository"
)

// Handlers contém todos os handlers da API
type Handlers struct {
	AuthHandler     *AuthHandler
	UserHandler     *UserHandler
	ProductHandler  *ProductHandler
	CustomerHandler *CustomerHandler
	MCPHandler      *MCPHandler
}

// NewHandlers cria uma nova instância de Handlers
func NewHandlers(
	logger logger.Logger,
	userRepo repository.UserRepository,
	productRepo repository.ProductRepository,
	customerRepo repository.CustomerRepository,
) (*Handlers, error) {
	// Criar handler MCP
	mcpHandler, err := NewMCPHandler(logger, userRepo, productRepo, customerRepo)
	if err != nil {
		return nil, err
	}

	// Os outros handlers seriam instanciados aqui...
	// Para fins de simplicidade, apenas vamos criar stubs para eles
	authHandler := &AuthHandler{}
	userHandler := &UserHandler{}
	productHandler := &ProductHandler{}
	customerHandler := &CustomerHandler{}

	return &Handlers{
		AuthHandler:     authHandler,
		UserHandler:     userHandler,
		ProductHandler:  productHandler,
		CustomerHandler: customerHandler,
		MCPHandler:      mcpHandler,
	}, nil
}

// Stubs dos handlers que seriam implementados completamente
type AuthHandler struct{}
type UserHandler struct{}
type ProductHandler struct{}
type CustomerHandler struct{}
