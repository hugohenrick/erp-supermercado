package dto

import (
	"time"

	"github.com/hugohenrick/erp-supermercado/internal/domain/certificate"
)

// CertificateRequest representa os dados para criar/atualizar um certificado digital
type CertificateRequest struct {
	BranchID        string    `json:"branch_id" binding:"required"`
	Name            string    `json:"name" binding:"required"`
	CertificateData []byte    `json:"certificate_data,omitempty"`
	CertificatePath string    `json:"certificate_path,omitempty"`
	Password        string    `json:"password" binding:"required"`
	ExpirationDate  time.Time `json:"expiration_date" binding:"required"`
	IsActive        bool      `json:"is_active"`
}

// CertificateUploadRequest representa os dados para upload de certificado
type CertificateUploadRequest struct {
	BranchID       string    `form:"branch_id" binding:"required"`
	Name           string    `form:"name" binding:"required"`
	Password       string    `form:"password" binding:"required"`
	ExpirationDate time.Time `form:"expiration_date" binding:"required"`
	IsActive       bool      `form:"is_active"`
}

// CertificateResponse representa a resposta com dados de um certificado
type CertificateResponse struct {
	ID              string    `json:"id"`
	BranchID        string    `json:"branch_id"`
	BranchName      string    `json:"branch_name,omitempty"`
	Name            string    `json:"name"`
	CertificatePath string    `json:"certificate_path,omitempty"`
	ExpirationDate  time.Time `json:"expiration_date"`
	IsActive        bool      `json:"is_active"`
	IsExpired       bool      `json:"is_expired"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// CertificateListResponse representa a resposta com uma lista de certificados
type CertificateListResponse struct {
	Certificates []CertificateResponse `json:"certificates"`
	Total        int                   `json:"total"`
	Page         int                   `json:"page"`
	PageSize     int                   `json:"page_size"`
}

// NewCertificateResponse cria um novo CertificateResponse a partir de um certificado
func NewCertificateResponse(cert *certificate.Certificate) *CertificateResponse {
	return &CertificateResponse{
		ID:              cert.ID,
		BranchID:        cert.BranchID,
		Name:            cert.Name,
		CertificatePath: cert.CertificatePath,
		ExpirationDate:  cert.ExpirationDate,
		IsActive:        cert.IsActive,
		IsExpired:       cert.IsExpired(),
		CreatedAt:       cert.CreatedAt,
		UpdatedAt:       cert.UpdatedAt,
	}
}

// NewCertificateListResponse cria um novo CertificateListResponse
func NewCertificateListResponse(certificates []*certificate.Certificate, total, page, pageSize int) *CertificateListResponse {
	response := &CertificateListResponse{
		Certificates: make([]CertificateResponse, 0, len(certificates)),
		Total:        total,
		Page:         page,
		PageSize:     pageSize,
	}

	for _, cert := range certificates {
		response.Certificates = append(response.Certificates, *NewCertificateResponse(cert))
	}

	return response
}
