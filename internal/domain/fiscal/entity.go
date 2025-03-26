package fiscal

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// FiscalEnvironment define o ambiente da SEFAZ
type FiscalEnvironment string

const (
	Production   FiscalEnvironment = "production"
	Homologation FiscalEnvironment = "homologation"
)

// PrintMode define o modo de impressão do DANFE
type PrintMode string

const (
	Normal      PrintMode = "normal"
	Contingency PrintMode = "contingency"
	None        PrintMode = "none"
)

// Configuration contém as configurações fiscais de uma filial
type Configuration struct {
	ID            string `json:"id"`
	TenantID      string `json:"tenant_id"`
	BranchID      string `json:"branch_id"`
	CertificateID string `json:"certificate_id"`

	// Configurações NFe
	NFeSeries      string            `json:"nfe_series"`
	NFeNextNumber  int               `json:"nfe_next_number"`
	NFeEnvironment FiscalEnvironment `json:"nfe_environment"`
	NFeCSCID       string            `json:"nfe_csc_id"`
	NFeCSCToken    string            `json:"nfe_csc_token"`

	// Configurações NFCe
	NFCeSeries      string            `json:"nfce_series"`
	NFCeNextNumber  int               `json:"nfce_next_number"`
	NFCeEnvironment FiscalEnvironment `json:"nfce_environment"`
	NFCeCSCID       string            `json:"nfce_csc_id"`
	NFCeCSCToken    string            `json:"nfce_csc_token"`

	// Configurações Gerais
	FiscalCSC          string `json:"fiscal_csc"`
	FiscalCSCID        string `json:"fiscal_csc_id"`
	ContingencyEnabled bool   `json:"contingency_enabled"`

	// SMTP para envio de documentos fiscais
	SMTPHost     string `json:"smtp_host"`
	SMTPPort     int    `json:"smtp_port"`
	SMTPUsername string `json:"smtp_username"`
	SMTPPassword string `json:"-"` // Não expor ao serializar para JSON

	// Configurações de impressão
	PrintDANFEMode   PrintMode `json:"print_danfe_mode"`
	PrinterName      string    `json:"printer_name"`
	PrinterPaperSize string    `json:"printer_paper_size"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NewConfiguration cria uma nova configuração fiscal
func NewConfiguration(
	tenantID string,
	branchID string,
	certificateID string,
) (*Configuration, error) {
	if tenantID == "" {
		return nil, errors.New("tenant ID é obrigatório")
	}
	if branchID == "" {
		return nil, errors.New("branch ID é obrigatório")
	}

	now := time.Now()
	return &Configuration{
		ID:               uuid.New().String(),
		TenantID:         tenantID,
		BranchID:         branchID,
		CertificateID:    certificateID,
		NFeSeries:        "1",
		NFeNextNumber:    1,
		NFeEnvironment:   Homologation,
		NFCeSeries:       "1",
		NFCeNextNumber:   1,
		NFCeEnvironment:  Homologation,
		PrintDANFEMode:   Normal,
		PrinterPaperSize: "A4",
		CreatedAt:        now,
		UpdatedAt:        now,
	}, nil
}

// ConfigureNFe configura os parâmetros de NFe
func (c *Configuration) ConfigureNFe(
	series string,
	nextNumber int,
	environment FiscalEnvironment,
	cscID string,
	cscToken string,
) error {
	if series == "" {
		return errors.New("série da NFe é obrigatória")
	}
	if nextNumber <= 0 {
		return errors.New("número inicial da NFe deve ser maior que zero")
	}

	c.NFeSeries = series
	c.NFeNextNumber = nextNumber
	c.NFeEnvironment = environment
	c.NFeCSCID = cscID
	c.NFeCSCToken = cscToken
	c.UpdatedAt = time.Now()
	return nil
}

// ConfigureNFCe configura os parâmetros de NFCe
func (c *Configuration) ConfigureNFCe(
	series string,
	nextNumber int,
	environment FiscalEnvironment,
	cscID string,
	cscToken string,
) error {
	if series == "" {
		return errors.New("série da NFCe é obrigatória")
	}
	if nextNumber <= 0 {
		return errors.New("número inicial da NFCe deve ser maior que zero")
	}

	c.NFCeSeries = series
	c.NFCeNextNumber = nextNumber
	c.NFCeEnvironment = environment
	c.NFCeCSCID = cscID
	c.NFCeCSCToken = cscToken
	c.UpdatedAt = time.Now()
	return nil
}

// ConfigureSMTP configura os parâmetros de email
func (c *Configuration) ConfigureSMTP(
	host string,
	port int,
	username string,
	password string,
) error {
	if host == "" {
		return errors.New("host SMTP é obrigatório")
	}
	if port <= 0 {
		return errors.New("porta SMTP deve ser maior que zero")
	}
	if username == "" {
		return errors.New("usuário SMTP é obrigatório")
	}

	c.SMTPHost = host
	c.SMTPPort = port
	c.SMTPUsername = username
	c.SMTPPassword = password
	c.UpdatedAt = time.Now()
	return nil
}

// ConfigurePrinting configura os parâmetros de impressão
func (c *Configuration) ConfigurePrinting(
	mode PrintMode,
	printerName string,
	paperSize string,
) {
	c.PrintDANFEMode = mode
	c.PrinterName = printerName
	c.PrinterPaperSize = paperSize
	c.UpdatedAt = time.Now()
}

// EnableContingency habilita o modo de contingência
func (c *Configuration) EnableContingency() {
	c.ContingencyEnabled = true
	c.UpdatedAt = time.Now()
}

// DisableContingency desabilita o modo de contingência
func (c *Configuration) DisableContingency() {
	c.ContingencyEnabled = false
	c.UpdatedAt = time.Now()
}

// GetNextNFeNumber obtém e incrementa o número da próxima NFe
func (c *Configuration) GetNextNFeNumber() int {
	current := c.NFeNextNumber
	c.NFeNextNumber++
	c.UpdatedAt = time.Now()
	return current
}

// GetNextNFCeNumber obtém e incrementa o número da próxima NFCe
func (c *Configuration) GetNextNFCeNumber() int {
	current := c.NFCeNextNumber
	c.NFCeNextNumber++
	c.UpdatedAt = time.Now()
	return current
}
