package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/hugohenrick/erp-supermercado/internal/domain/tenant"
	pkgtenant "github.com/hugohenrick/erp-supermercado/pkg/tenant"
)

// TenantValidator implementa a interface para validação de tenant
type TenantValidator struct {
	repository tenant.Repository
}

// NewTenantValidator cria uma nova instância de TenantValidator
func NewTenantValidator(repository tenant.Repository) pkgtenant.TenantValidator {
	return &TenantValidator{
		repository: repository,
	}
}

// ValidateTenant verifica se um tenant existe e está ativo
func (v *TenantValidator) ValidateTenant(tenantID string) (bool, error) {
	// Verifica se o tenant existe
	t, err := v.repository.FindByID(context.Background(), tenantID)
	if err != nil {
		if errors.Is(err, ErrTenantNotFound) {
			return false, nil
		}
		return false, err
	}

	// Verifica se o tenant está ativo
	return t.IsActive(), nil
}

// Validate implementa a interface TenantValidator.Validate
func (v *TenantValidator) Validate(c *gin.Context, tenantID string) error {
	// Validar se o ID é válido
	if tenantID == "" {
		return pkgtenant.ErrTenantNotSpecified
	}

	// Buscar o tenant no repositório
	t, err := v.repository.FindByID(c.Request.Context(), tenantID)
	if err != nil {
		if errors.Is(err, ErrTenantNotFound) {
			return pkgtenant.ErrTenantNotFound
		}
		return fmt.Errorf("erro ao buscar tenant: %w", err)
	}

	// Verificar se o tenant está ativo
	if !t.IsActive() {
		return pkgtenant.ErrTenantNotActive
	}

	return nil
}
