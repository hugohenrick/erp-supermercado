package intent

import (
	"github.com/hugohenrick/erp-supermercado/internal/domain/customer"
	"github.com/hugohenrick/erp-supermercado/pkg/logger"
	"github.com/hugohenrick/erp-supermercado/pkg/mcp/intent/adapter"
	"github.com/hugohenrick/erp-supermercado/pkg/repository"
)

// InitCustomerIntentHandler initializes the customer intent handler with the proper repository
// This creates an adapter that bridges the gap between the internal domain model and the simplified
// MCP domain model
func InitCustomerIntentHandler(log logger.Logger, internalRepo customer.Repository) *CustomerIntentHandler {
	// Create the adapter
	adapterRepo := adapter.NewCustomerRepositoryAdapter(internalRepo, log)

	// Create the handler with the adapter
	return NewCustomerIntentHandler(log, adapterRepo)
}

// Use the other intent handlers directly as they are, since they already use the pkg/repository interfaces

// InitProductIntentHandler initializes the product intent handler
func InitProductIntentHandler(log logger.Logger, productRepo repository.ProductRepository) *ProductIntentHandler {
	return NewProductIntentHandler(log, productRepo)
}

// InitUserIntentHandler initializes the user intent handler
func InitUserIntentHandler(log logger.Logger, userRepo repository.UserRepository) *UserIntentHandler {
	return NewUserIntentHandler(log, userRepo)
}
