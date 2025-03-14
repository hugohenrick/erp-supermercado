package repository

import (
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	domain_tenant "github.com/hugohenrick/erp-supermercado/internal/domain/tenant"
	"github.com/hugohenrick/erp-supermercado/pkg/tenant"
)

// TenantValidator implementa a interface pkg/tenant.TenantValidator
type TenantValidator struct {
	repository domain_tenant.Repository
}

// NewTenantValidator cria um novo validador de tenant
func NewTenantValidator(repository domain_tenant.Repository) *TenantValidator {
	return &TenantValidator{
		repository: repository,
	}
}

// Validate implementa a interface TenantValidator.Validate
func (v *TenantValidator) Validate(c *gin.Context, tenantID string) error {
	// Validar se o ID é válido
	if tenantID == "" {
		return tenant.ErrTenantNotSpecified
	}

	// Buscar o tenant no repositório
	t, err := v.repository.FindByID(c.Request.Context(), tenantID)
	if err != nil {
		if errors.Is(err, ErrTenantNotFound) {
			return tenant.ErrTenantNotFound
		}
		return fmt.Errorf("erro ao buscar tenant: %w", err)
	}

	// Verificar se o tenant está ativo
	if !t.IsActive() {
		return tenant.ErrTenantNotActive
	}

	return nil
}
