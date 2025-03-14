package dto

import (
	"time"

	"github.com/hugohenrick/erp-supermercado/internal/domain/customer"
)

// CustomerAddressRequest representa a requisição de endereço do cliente
type CustomerAddressRequest struct {
	Street      string `json:"street" binding:"required"`
	Number      string `json:"number" binding:"required"`
	Complement  string `json:"complement"`
	District    string `json:"district" binding:"required"`
	City        string `json:"city" binding:"required"`
	State       string `json:"state" binding:"required"`
	ZipCode     string `json:"zip_code" binding:"required"`
	Country     string `json:"country" binding:"required"`
	AddressType string `json:"address_type" binding:"required"`
}

// CustomerContactRequest representa a requisição de contato do cliente
type CustomerContactRequest struct {
	Name        string `json:"name" binding:"required"`
	Department  string `json:"department"`
	Phone       string `json:"phone"`
	MobilePhone string `json:"mobile_phone"`
	Email       string `json:"email"`
	Position    string `json:"position"`
	MainContact bool   `json:"main_contact"`
}

// CustomerRequest representa a requisição de cliente
type CustomerRequest struct {
	PersonType      customer.PersonType      `json:"person_type" binding:"required"`
	Name            string                   `json:"name" binding:"required"`
	TradeName       string                   `json:"trade_name"`
	Document        string                   `json:"document" binding:"required"`
	StateDocument   string                   `json:"state_document"`
	CityDocument    string                   `json:"city_document"`
	TaxRegime       customer.TaxRegime       `json:"tax_regime" binding:"required"`
	CustomerType    customer.CustomerType    `json:"customer_type" binding:"required"`
	CreditLimit     float64                  `json:"credit_limit"`
	PaymentTerm     int                      `json:"payment_term"`
	Website         string                   `json:"website"`
	Observations    string                   `json:"observations"`
	FiscalNotes     string                   `json:"fiscal_notes"`
	Addresses       []CustomerAddressRequest `json:"addresses" binding:"required,min=1"`
	Contacts        []CustomerContactRequest `json:"contacts" binding:"required,min=1"`
	ExternalCode    string                   `json:"external_code"`
	SalesmanID      string                   `json:"salesman_id"`
	PriceTableID    string                   `json:"price_table_id"`
	PaymentMethodID string                   `json:"payment_method_id"`
	SUFRAMA         string                   `json:"suframa"`
	ReferenceCode   string                   `json:"reference_code"`
}

// CustomerAddressResponse representa a resposta de endereço do cliente
type CustomerAddressResponse struct {
	Street      string `json:"street"`
	Number      string `json:"number"`
	Complement  string `json:"complement"`
	District    string `json:"district"`
	City        string `json:"city"`
	State       string `json:"state"`
	ZipCode     string `json:"zip_code"`
	Country     string `json:"country"`
	AddressType string `json:"address_type"`
}

// CustomerContactResponse representa a resposta de contato do cliente
type CustomerContactResponse struct {
	Name        string `json:"name"`
	Department  string `json:"department"`
	Phone       string `json:"phone"`
	MobilePhone string `json:"mobile_phone"`
	Email       string `json:"email"`
	Position    string `json:"position"`
	MainContact bool   `json:"main_contact"`
}

// CustomerResponse representa a resposta de cliente
type CustomerResponse struct {
	ID              string                    `json:"id"`
	TenantID        string                    `json:"tenant_id"`
	BranchID        string                    `json:"branch_id"`
	PersonType      customer.PersonType       `json:"person_type"`
	Name            string                    `json:"name"`
	TradeName       string                    `json:"trade_name"`
	Document        string                    `json:"document"`
	StateDocument   string                    `json:"state_document"`
	CityDocument    string                    `json:"city_document"`
	TaxRegime       customer.TaxRegime        `json:"tax_regime"`
	CustomerType    customer.CustomerType     `json:"customer_type"`
	Status          customer.Status           `json:"status"`
	CreditLimit     float64                   `json:"credit_limit"`
	PaymentTerm     int                       `json:"payment_term"`
	Website         string                    `json:"website"`
	Observations    string                    `json:"observations"`
	FiscalNotes     string                    `json:"fiscal_notes"`
	Addresses       []CustomerAddressResponse `json:"addresses"`
	Contacts        []CustomerContactResponse `json:"contacts"`
	LastPurchaseAt  *time.Time                `json:"last_purchase_at"`
	CreatedAt       time.Time                 `json:"created_at"`
	UpdatedAt       time.Time                 `json:"updated_at"`
	ExternalCode    string                    `json:"external_code"`
	SalesmanID      string                    `json:"salesman_id"`
	PriceTableID    string                    `json:"price_table_id"`
	PaymentMethodID string                    `json:"payment_method_id"`
	SUFRAMA         string                    `json:"suframa"`
	ReferenceCode   string                    `json:"reference_code"`
}

// CustomerListResponse representa a resposta de lista de clientes
type CustomerListResponse struct {
	Items      []CustomerResponse `json:"items"`
	Total      int                `json:"total"`
	Page       int                `json:"page"`
	Size       int                `json:"size"`
	TotalPages int                `json:"total_pages"`
}

// ToCustomerResponse converte um cliente do domínio para DTO
func ToCustomerResponse(c *customer.Customer) *CustomerResponse {
	addresses := make([]CustomerAddressResponse, len(c.Addresses))
	for i, addr := range c.Addresses {
		addresses[i] = CustomerAddressResponse{
			Street:      addr.Street,
			Number:      addr.Number,
			Complement:  addr.Complement,
			District:    addr.District,
			City:        addr.City,
			State:       addr.State,
			ZipCode:     addr.ZipCode,
			Country:     addr.Country,
			AddressType: addr.AddressType,
		}
	}

	contacts := make([]CustomerContactResponse, len(c.Contacts))
	for i, cont := range c.Contacts {
		contacts[i] = CustomerContactResponse{
			Name:        cont.Name,
			Department:  cont.Department,
			Phone:       cont.Phone,
			MobilePhone: cont.MobilePhone,
			Email:       cont.Email,
			Position:    cont.Position,
			MainContact: cont.MainContact,
		}
	}

	return &CustomerResponse{
		ID:              c.ID,
		TenantID:        c.TenantID,
		BranchID:        c.BranchID,
		PersonType:      c.PersonType,
		Name:            c.Name,
		TradeName:       c.TradeName,
		Document:        c.Document,
		StateDocument:   c.StateDocument,
		CityDocument:    c.CityDocument,
		TaxRegime:       c.TaxRegime,
		CustomerType:    c.CustomerType,
		Status:          c.Status,
		CreditLimit:     c.CreditLimit,
		PaymentTerm:     c.PaymentTerm,
		Website:         c.Website,
		Observations:    c.Observations,
		FiscalNotes:     c.FiscalNotes,
		Addresses:       addresses,
		Contacts:        contacts,
		LastPurchaseAt:  c.LastPurchaseAt,
		CreatedAt:       c.CreatedAt,
		UpdatedAt:       c.UpdatedAt,
		ExternalCode:    c.ExternalCode,
		SalesmanID:      c.SalesmanID,
		PriceTableID:    c.PriceTableID,
		PaymentMethodID: c.PaymentMethodID,
		SUFRAMA:         c.SUFRAMA,
		ReferenceCode:   c.ReferenceCode,
	}
}

// ToCustomerListResponse converte uma lista de clientes do domínio para DTO
func ToCustomerListResponse(customers []*customer.Customer, total, page, size, totalPages int) *CustomerListResponse {
	items := make([]CustomerResponse, len(customers))
	for i, c := range customers {
		items[i] = *ToCustomerResponse(c)
	}

	return &CustomerListResponse{
		Items:      items,
		Total:      total,
		Page:       page,
		Size:       size,
		TotalPages: totalPages,
	}
}
