package dto

import (
	"time"

	"github.com/hugohenrick/erp-supermercado/internal/domain/branch"
)

// AddressRequest representa a estrutura de dados para endereço
type AddressRequest struct {
	Street     string `json:"street"`
	Number     string `json:"number"`
	Complement string `json:"complement"`
	District   string `json:"district"`
	City       string `json:"city"`
	State      string `json:"state"`
	ZipCode    string `json:"zip_code"`
	Country    string `json:"country"`
}

// BranchRequest representa a estrutura de dados para criação/atualização de filial
type BranchRequest struct {
	Name     string         `json:"name" binding:"required"`
	Code     string         `json:"code" binding:"required"`
	Type     string         `json:"type" binding:"required"`
	Document string         `json:"document"`
	Phone    string         `json:"phone"`
	Email    string         `json:"email"`
	Address  AddressRequest `json:"address"`
	IsMain   bool           `json:"is_main"`
}

// AddressResponse representa a estrutura de resposta para endereço
type AddressResponse struct {
	Street     string `json:"street"`
	Number     string `json:"number"`
	Complement string `json:"complement"`
	District   string `json:"district"`
	City       string `json:"city"`
	State      string `json:"state"`
	ZipCode    string `json:"zip_code"`
	Country    string `json:"country"`
}

// BranchResponse representa a estrutura de resposta para filial
type BranchResponse struct {
	ID        string          `json:"id"`
	TenantID  string          `json:"tenant_id"`
	Name      string          `json:"name"`
	Code      string          `json:"code"`
	Type      string          `json:"type"`
	Document  string          `json:"document"`
	Phone     string          `json:"phone"`
	Email     string          `json:"email"`
	Address   AddressResponse `json:"address"`
	Status    string          `json:"status"`
	IsMain    bool            `json:"is_main"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// BranchListResponse representa a resposta de listagem de filiais
type BranchListResponse struct {
	Branches   []BranchResponse `json:"branches"`
	TotalCount int              `json:"total_count"`
	Page       int              `json:"page"`
	PageSize   int              `json:"page_size"`
	TotalPages int              `json:"total_pages"`
}

// ToBranchResponse converte um modelo de domínio em uma resposta DTO
func ToBranchResponse(b *branch.Branch) BranchResponse {
	return BranchResponse{
		ID:       b.ID,
		TenantID: b.TenantID,
		Name:     b.Name,
		Code:     b.Code,
		Type:     string(b.Type),
		Document: b.Document,
		Phone:    b.Phone,
		Email:    b.Email,
		Address: AddressResponse{
			Street:     b.Address.Street,
			Number:     b.Address.Number,
			Complement: b.Address.Complement,
			District:   b.Address.District,
			City:       b.Address.City,
			State:      b.Address.State,
			ZipCode:    b.Address.ZipCode,
			Country:    b.Address.Country,
		},
		Status:    string(b.Status),
		IsMain:    b.IsMain,
		CreatedAt: b.CreatedAt,
		UpdatedAt: b.UpdatedAt,
	}
}

// ToBranchListResponse converte uma lista de filiais para o formato de resposta
func ToBranchListResponse(branches []*branch.Branch, totalCount, page, pageSize int) BranchListResponse {
	response := BranchListResponse{
		Branches:   make([]BranchResponse, len(branches)),
		TotalCount: totalCount,
		Page:       page,
		PageSize:   pageSize,
	}

	for i, b := range branches {
		response.Branches[i] = ToBranchResponse(b)
	}

	// Calcular total de páginas
	response.TotalPages = (totalCount + pageSize - 1) / pageSize
	if response.TotalPages == 0 {
		response.TotalPages = 1
	}

	return response
}
