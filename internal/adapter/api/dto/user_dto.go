package dto

import (
	"time"

	"github.com/hugohenrick/erp-supermercado/internal/domain/user"
)

// UserRequest representa os dados de um usuário para criação ou atualização
type UserRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password"`
	BranchID string `json:"branch_id"`
	Role     string `json:"role" binding:"required"`
}

// UserResponse representa a resposta com dados de um usuário
type UserResponse struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	BranchID    string    `json:"branch_id,omitempty"`
	Name        string    `json:"name"`
	Email       string    `json:"email"`
	Role        string    `json:"role"`
	Status      string    `json:"status"`
	LastLoginAt time.Time `json:"last_login_at,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// UserListResponse representa a resposta com a lista de usuários paginada
type UserListResponse struct {
	Data       []UserResponse `json:"data"`
	TotalCount int            `json:"total_count"`
	Page       int            `json:"page"`
	PageSize   int            `json:"page_size"`
	TotalPages int            `json:"total_pages"`
}

// ChangePasswordRequest representa os dados para alteração de senha
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=6"`
}

// ToUserResponse converte um usuário do domínio para DTO de resposta
func ToUserResponse(u *user.User) UserResponse {
	return UserResponse{
		ID:          u.ID,
		TenantID:    u.TenantID,
		BranchID:    u.BranchID,
		Name:        u.Name,
		Email:       u.Email,
		Role:        string(u.Role),
		Status:      string(u.Status),
		LastLoginAt: u.LastLoginAt,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}
}

// ToUserListResponse converte uma lista de usuários do domínio para DTO de resposta paginada
func ToUserListResponse(users []*user.User, totalCount, page, pageSize int) UserListResponse {
	data := make([]UserResponse, len(users))
	for i, u := range users {
		data[i] = ToUserResponse(u)
	}

	totalPages := calculateTotalPages(totalCount, pageSize)

	return UserListResponse{
		Data:       data,
		TotalCount: totalCount,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}
}
