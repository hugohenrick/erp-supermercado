package tenant

import "errors"

// Erros comuns relacionados a operações de tenant
var (
	// ErrTenantNotSpecified ocorre quando um ID de tenant não é fornecido
	ErrTenantNotSpecified = errors.New("tenant ID não especificado")

	// ErrTenantNotFound ocorre quando um tenant não é encontrado
	ErrTenantNotFound = errors.New("tenant não encontrado")

	// ErrTenantNotActive ocorre quando um tenant não está com status ativo
	ErrTenantNotActive = errors.New("tenant não está ativo")
)
