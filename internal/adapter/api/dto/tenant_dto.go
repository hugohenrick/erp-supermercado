package dto

import (
	"time"

	"github.com/hugohenrick/erp-supermercado/internal/domain/tenant"
)

// TenantRequest representa a estrutura de dados para criação/atualização de tenant
type TenantRequest struct {
	Name        string `json:"name" binding:"required"`
	Document    string `json:"document" binding:"required"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	PlanType    string `json:"plan_type" binding:"required"`
	MaxBranches int    `json:"max_branches" binding:"min=1"`
}

// TenantResponse representa a estrutura de dados de resposta para tenant
type TenantResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Document    string    `json:"document"`
	Email       string    `json:"email"`
	Phone       string    `json:"phone"`
	Status      string    `json:"status"`
	PlanType    string    `json:"plan_type"`
	MaxBranches int       `json:"max_branches"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TenantListResponse representa a resposta de listagem de tenants
type TenantListResponse struct {
	Tenants    []TenantResponse `json:"tenants"`
	TotalCount int              `json:"total_count"`
	Page       int              `json:"page"`
	PageSize   int              `json:"page_size"`
	TotalPages int              `json:"total_pages"`
}

// ToResponse converte um modelo de domínio em uma resposta DTO
func ToTenantResponse(t *tenant.Tenant) TenantResponse {
	return TenantResponse{
		ID:          t.ID,
		Name:        t.Name,
		Document:    t.Document,
		Email:       t.Email,
		Phone:       t.Phone,
		Status:      string(t.Status),
		PlanType:    t.PlanType,
		MaxBranches: t.MaxBranches,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}

// ToTenantListResponse converte uma lista de tenants para o formato de resposta
func ToTenantListResponse(tenants []*tenant.Tenant, totalCount, page, pageSize int) TenantListResponse {
	response := TenantListResponse{
		Tenants:    make([]TenantResponse, len(tenants)),
		TotalCount: totalCount,
		Page:       page,
		PageSize:   pageSize,
	}

	for i, t := range tenants {
		response.Tenants[i] = ToTenantResponse(t)
	}

	// Calcular total de páginas
	response.TotalPages = (totalCount + pageSize - 1) / pageSize
	if response.TotalPages == 0 {
		response.TotalPages = 1
	}

	return response
}
