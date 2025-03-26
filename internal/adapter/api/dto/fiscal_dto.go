package dto

import (
	"time"

	"github.com/hugohenrick/erp-supermercado/internal/domain/fiscal"
)

// FiscalConfigRequest representa os dados para criar/atualizar uma configuração fiscal
type FiscalConfigRequest struct {
	BranchID      string `json:"branch_id" binding:"required"`
	CertificateID string `json:"certificate_id,omitempty"`

	// Configurações NFe
	NFeSeries      string                   `json:"nfe_series" binding:"required"`
	NFeNextNumber  int                      `json:"nfe_next_number" binding:"required,min=1"`
	NFeEnvironment fiscal.FiscalEnvironment `json:"nfe_environment" binding:"required"`
	NFeCSCID       string                   `json:"nfe_csc_id,omitempty"`
	NFeCSCToken    string                   `json:"nfe_csc_token,omitempty"`

	// Configurações NFCe
	NFCeSeries      string                   `json:"nfce_series" binding:"required"`
	NFCeNextNumber  int                      `json:"nfce_next_number" binding:"required,min=1"`
	NFCeEnvironment fiscal.FiscalEnvironment `json:"nfce_environment" binding:"required"`
	NFCeCSCID       string                   `json:"nfce_csc_id,omitempty"`
	NFCeCSCToken    string                   `json:"nfce_csc_token,omitempty"`

	// Configurações Gerais
	FiscalCSC          string `json:"fiscal_csc,omitempty"`
	FiscalCSCID        string `json:"fiscal_csc_id,omitempty"`
	ContingencyEnabled bool   `json:"contingency_enabled"`

	// SMTP para envio de documentos fiscais
	SMTPHost     string `json:"smtp_host,omitempty"`
	SMTPPort     int    `json:"smtp_port,omitempty"`
	SMTPUsername string `json:"smtp_username,omitempty"`
	SMTPPassword string `json:"smtp_password,omitempty"`

	// Configurações de impressão
	PrintDANFEMode   fiscal.PrintMode `json:"print_danfe_mode"`
	PrinterName      string           `json:"printer_name,omitempty"`
	PrinterPaperSize string           `json:"printer_paper_size,omitempty"`
}

// FiscalConfigResponse representa a resposta com dados de uma configuração fiscal
type FiscalConfigResponse struct {
	ID              string `json:"id"`
	BranchID        string `json:"branch_id"`
	BranchName      string `json:"branch_name,omitempty"`
	CertificateID   string `json:"certificate_id,omitempty"`
	CertificateName string `json:"certificate_name,omitempty"`

	// Configurações NFe
	NFeSeries      string                   `json:"nfe_series"`
	NFeNextNumber  int                      `json:"nfe_next_number"`
	NFeEnvironment fiscal.FiscalEnvironment `json:"nfe_environment"`
	NFeCSCID       string                   `json:"nfe_csc_id,omitempty"`
	NFeCSCToken    string                   `json:"nfe_csc_token,omitempty"`

	// Configurações NFCe
	NFCeSeries      string                   `json:"nfce_series"`
	NFCeNextNumber  int                      `json:"nfce_next_number"`
	NFCeEnvironment fiscal.FiscalEnvironment `json:"nfce_environment"`
	NFCeCSCID       string                   `json:"nfce_csc_id,omitempty"`
	NFCeCSCToken    string                   `json:"nfce_csc_token,omitempty"`

	// Configurações Gerais
	FiscalCSC          string `json:"fiscal_csc,omitempty"`
	FiscalCSCID        string `json:"fiscal_csc_id,omitempty"`
	ContingencyEnabled bool   `json:"contingency_enabled"`

	// SMTP para envio de documentos fiscais
	SMTPHost     string `json:"smtp_host,omitempty"`
	SMTPPort     int    `json:"smtp_port,omitempty"`
	SMTPUsername string `json:"smtp_username,omitempty"`

	// Configurações de impressão
	PrintDANFEMode   fiscal.PrintMode `json:"print_danfe_mode"`
	PrinterName      string           `json:"printer_name,omitempty"`
	PrinterPaperSize string           `json:"printer_paper_size,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// FiscalConfigListResponse representa a resposta com uma lista de configurações fiscais
type FiscalConfigListResponse struct {
	Configurations []FiscalConfigResponse `json:"configurations"`
	Total          int                    `json:"total"`
	Page           int                    `json:"page"`
	PageSize       int                    `json:"page_size"`
}

// NewFiscalConfigResponse cria um novo FiscalConfigResponse a partir de uma configuração fiscal
func NewFiscalConfigResponse(config *fiscal.Configuration) *FiscalConfigResponse {
	return &FiscalConfigResponse{
		ID:            config.ID,
		BranchID:      config.BranchID,
		CertificateID: config.CertificateID,

		NFeSeries:      config.NFeSeries,
		NFeNextNumber:  config.NFeNextNumber,
		NFeEnvironment: config.NFeEnvironment,
		NFeCSCID:       config.NFeCSCID,
		NFeCSCToken:    config.NFeCSCToken,

		NFCeSeries:      config.NFCeSeries,
		NFCeNextNumber:  config.NFCeNextNumber,
		NFCeEnvironment: config.NFCeEnvironment,
		NFCeCSCID:       config.NFCeCSCID,
		NFCeCSCToken:    config.NFCeCSCToken,

		FiscalCSC:          config.FiscalCSC,
		FiscalCSCID:        config.FiscalCSCID,
		ContingencyEnabled: config.ContingencyEnabled,

		SMTPHost:     config.SMTPHost,
		SMTPPort:     config.SMTPPort,
		SMTPUsername: config.SMTPUsername,

		PrintDANFEMode:   config.PrintDANFEMode,
		PrinterName:      config.PrinterName,
		PrinterPaperSize: config.PrinterPaperSize,

		CreatedAt: config.CreatedAt,
		UpdatedAt: config.UpdatedAt,
	}
}

// NewFiscalConfigListResponse cria um novo FiscalConfigListResponse
func NewFiscalConfigListResponse(configs []*fiscal.Configuration, total, page, pageSize int) *FiscalConfigListResponse {
	response := &FiscalConfigListResponse{
		Configurations: make([]FiscalConfigResponse, 0, len(configs)),
		Total:          total,
		Page:           page,
		PageSize:       pageSize,
	}

	for _, config := range configs {
		response.Configurations = append(response.Configurations, *NewFiscalConfigResponse(config))
	}

	return response
}
