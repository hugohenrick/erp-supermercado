package customer

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrEmptyName       = errors.New("nome não pode ser vazio")
	ErrEmptyDocument   = errors.New("documento não pode ser vazio")
	ErrInvalidDocument = errors.New("documento inválido")
	ErrInvalidEmail    = errors.New("email inválido")
)

// PersonType define o tipo de pessoa (física ou jurídica)
type PersonType string

const (
	PersonTypePF PersonType = "PF" // Pessoa Física
	PersonTypePJ PersonType = "PJ" // Pessoa Jurídica
)

// Status representa o estado do cliente
type Status string

const (
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
	StatusBlocked  Status = "blocked"
)

// CustomerType define o tipo de cliente
type CustomerType string

const (
	TypeFinal     CustomerType = "final"     // Consumidor Final
	TypeReseller  CustomerType = "reseller"  // Revendedor
	TypeWholesale CustomerType = "wholesale" // Atacadista
)

// TaxRegime define o regime tributário
type TaxRegime string

const (
	TaxRegimeSimples   TaxRegime = "simples"   // Simples Nacional
	TaxRegimeMEI       TaxRegime = "mei"       // Microempreendedor Individual
	TaxRegimePresumido TaxRegime = "presumido" // Lucro Presumido
	TaxRegimeReal      TaxRegime = "real"      // Lucro Real
)

// Address representa o endereço do cliente
type Address struct {
	Street          string `json:"street"`           // Logradouro
	Number          string `json:"number"`           // Número
	Complement      string `json:"complement"`       // Complemento
	District        string `json:"district"`         // Bairro
	City            string `json:"city"`             // Cidade
	State           string `json:"state"`            // Estado
	ZipCode         string `json:"zip_code"`         // CEP
	Country         string `json:"country"`          // País
	CityCode        string `json:"city_code"`        // Código IBGE da Cidade
	StateCode       string `json:"state_code"`       // Código IBGE do Estado
	CountryCode     string `json:"country_code"`     // Código do País
	AddressType     string `json:"address_type"`     // Tipo de Endereço (Entrega, Cobrança, etc)
	MainAddress     bool   `json:"main_address"`     // Endereço Principal
	DeliveryAddress bool   `json:"delivery_address"` // Endereço de Entrega
}

// Contact representa um contato do cliente
type Contact struct {
	Name        string `json:"name"`         // Nome do Contato
	Department  string `json:"department"`   // Departamento
	Phone       string `json:"phone"`        // Telefone
	MobilePhone string `json:"mobile_phone"` // Celular
	Email       string `json:"email"`        // Email
	Position    string `json:"position"`     // Cargo
	MainContact bool   `json:"main_contact"` // Contato Principal
}

// Customer representa um cliente no sistema
type Customer struct {
	ID              string       `json:"id"`                // ID do Cliente
	TenantID        string       `json:"tenant_id"`         // ID do Tenant
	BranchID        string       `json:"branch_id"`         // ID da Filial
	PersonType      PersonType   `json:"person_type"`       // Tipo de Pessoa (PF/PJ)
	Name            string       `json:"name"`              // Nome/Razão Social
	TradeName       string       `json:"trade_name"`        // Nome Fantasia
	Document        string       `json:"document"`          // CPF/CNPJ
	StateDocument   string       `json:"state_document"`    // Inscrição Estadual
	CityDocument    string       `json:"city_document"`     // Inscrição Municipal
	TaxRegime       TaxRegime    `json:"tax_regime"`        // Regime Tributário
	CustomerType    CustomerType `json:"customer_type"`     // Tipo de Cliente
	Status          Status       `json:"status"`            // Status do Cliente
	CreditLimit     float64      `json:"credit_limit"`      // Limite de Crédito
	PaymentTerm     int          `json:"payment_term"`      // Prazo de Pagamento (em dias)
	Website         string       `json:"website"`           // Website
	Observations    string       `json:"observations"`      // Observações
	FiscalNotes     string       `json:"fiscal_notes"`      // Observações para Nota Fiscal
	Addresses       []Address    `json:"addresses"`         // Endereços
	Contacts        []Contact    `json:"contacts"`          // Contatos
	LastPurchaseAt  *time.Time   `json:"last_purchase_at"`  // Data da Última Compra
	CreatedAt       time.Time    `json:"created_at"`        // Data de Criação
	UpdatedAt       time.Time    `json:"updated_at"`        // Data de Atualização
	ExternalCode    string       `json:"external_code"`     // Código Externo (integração)
	SalesmanID      string       `json:"salesman_id"`       // ID do Vendedor
	PriceTableID    string       `json:"price_table_id"`    // ID da Tabela de Preços
	PaymentMethodID string       `json:"payment_method_id"` // ID da Forma de Pagamento
	SUFRAMA         string       `json:"suframa"`           // Código SUFRAMA
	ReferenceCode   string       `json:"reference_code"`    // Código de Referência
}

// NewCustomer cria um novo cliente
func NewCustomer(
	tenantID string,
	branchID string,
	personType PersonType,
	name string,
	document string,
) (*Customer, error) {
	if name == "" {
		return nil, ErrEmptyName
	}

	if document == "" {
		return nil, ErrEmptyDocument
	}

	// Aqui poderia haver validação de CPF/CNPJ

	now := time.Now()
	return &Customer{
		ID:         uuid.New().String(),
		TenantID:   tenantID,
		BranchID:   branchID,
		PersonType: personType,
		Name:       name,
		Document:   document,
		Status:     StatusActive,
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}

// IsActive verifica se o cliente está ativo
func (c *Customer) IsActive() bool {
	return c.Status == StatusActive
}

// Activate ativa o cliente
func (c *Customer) Activate() {
	c.Status = StatusActive
	c.UpdatedAt = time.Now()
}

// Deactivate desativa o cliente
func (c *Customer) Deactivate() {
	c.Status = StatusInactive
	c.UpdatedAt = time.Now()
}

// Block bloqueia o cliente
func (c *Customer) Block() {
	c.Status = StatusBlocked
	c.UpdatedAt = time.Now()
}

// AddAddress adiciona um endereço ao cliente
func (c *Customer) AddAddress(addr Address) {
	c.Addresses = append(c.Addresses, addr)
	c.UpdatedAt = time.Now()
}

// AddContact adiciona um contato ao cliente
func (c *Customer) AddContact(contact Contact) {
	c.Contacts = append(c.Contacts, contact)
	c.UpdatedAt = time.Now()
}

// Update atualiza os dados do cliente
func (c *Customer) Update(
	name string,
	tradeName string,
	stateDocument string,
	cityDocument string,
	taxRegime TaxRegime,
	customerType CustomerType,
	creditLimit float64,
	paymentTerm int,
	website string,
	observations string,
	fiscalNotes string,
	externalCode string,
	salesmanID string,
	priceTableID string,
	paymentMethodID string,
	suframa string,
	referenceCode string,
) error {
	if name == "" {
		return ErrEmptyName
	}

	c.Name = name
	c.TradeName = tradeName
	c.StateDocument = stateDocument
	c.CityDocument = cityDocument
	c.TaxRegime = taxRegime
	c.CustomerType = customerType
	c.CreditLimit = creditLimit
	c.PaymentTerm = paymentTerm
	c.Website = website
	c.Observations = observations
	c.FiscalNotes = fiscalNotes
	c.ExternalCode = externalCode
	c.SalesmanID = salesmanID
	c.PriceTableID = priceTableID
	c.PaymentMethodID = paymentMethodID
	c.SUFRAMA = suframa
	c.ReferenceCode = referenceCode
	c.UpdatedAt = time.Now()

	return nil
}

// UpdateLastPurchase atualiza a data da última compra
func (c *Customer) UpdateLastPurchase() {
	now := time.Now()
	c.LastPurchaseAt = &now
	c.UpdatedAt = now
}

// GetMainAddress retorna o endereço principal
func (c *Customer) GetMainAddress() *Address {
	for _, addr := range c.Addresses {
		if addr.MainAddress {
			return &addr
		}
	}
	return nil
}

// GetDeliveryAddress retorna o endereço de entrega
func (c *Customer) GetDeliveryAddress() *Address {
	for _, addr := range c.Addresses {
		if addr.DeliveryAddress {
			return &addr
		}
	}
	return nil
}

// GetMainContact retorna o contato principal
func (c *Customer) GetMainContact() *Contact {
	for _, contact := range c.Contacts {
		if contact.MainContact {
			return &contact
		}
	}
	return nil
}
