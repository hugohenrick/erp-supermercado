package domain

import (
	"time"
)

// User representa um usuário do sistema
type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Password  string    `json:"-"`
	Role      string    `json:"role"`
	TenantID  string    `json:"tenant_id"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Product representa um produto no sistema
type Product struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	SKU         string    `json:"sku"`
	Price       float64   `json:"price"`
	StockQty    int       `json:"stock_qty"`
	Category    string    `json:"category"`
	TenantID    string    `json:"tenant_id"`
	Active      bool      `json:"active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Customer representa um cliente no sistema
type Customer struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	Document     string    `json:"document"`
	Phone        string    `json:"phone"`
	Address      string    `json:"address"`
	City         string    `json:"city"`
	State        string    `json:"state"`
	ZipCode      string    `json:"zip_code"`
	TenantID     string    `json:"tenant_id"`
	CustomerType string    `json:"customer_type"` // pessoa física ou jurídica
	Active       bool      `json:"active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Tenant representa um inquilino do sistema (supermercado)
type Tenant struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Document    string    `json:"document"`
	Phone       string    `json:"phone"`
	Email       string    `json:"email"`
	Address     string    `json:"address"`
	City        string    `json:"city"`
	State       string    `json:"state"`
	ZipCode     string    `json:"zip_code"`
	Active      bool      `json:"active"`
	PlanID      string    `json:"plan_id"`
	PlanExpires time.Time `json:"plan_expires"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Sale representa uma venda no sistema
type Sale struct {
	ID           string     `json:"id"`
	CustomerID   string     `json:"customer_id"`
	Customer     *Customer  `json:"customer,omitempty"`
	UserID       string     `json:"user_id"`
	TenantID     string     `json:"tenant_id"`
	Total        float64    `json:"total"`
	Discount     float64    `json:"discount"`
	TaxAmount    float64    `json:"tax_amount"`
	Status       string     `json:"status"` // pendente, concluída, cancelada
	PaymentType  string     `json:"payment_type"`
	PaymentSplit []Payment  `json:"payment_split,omitempty"`
	Items        []SaleItem `json:"items,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// SaleItem representa um item de venda
type SaleItem struct {
	ID        string    `json:"id"`
	SaleID    string    `json:"sale_id"`
	ProductID string    `json:"product_id"`
	Product   *Product  `json:"product,omitempty"`
	Quantity  int       `json:"quantity"`
	Price     float64   `json:"price"` // preço no momento da venda
	Discount  float64   `json:"discount"`
	Total     float64   `json:"total"`
	CreatedAt time.Time `json:"created_at"`
}

// Payment representa um pagamento
type Payment struct {
	ID             string    `json:"id"`
	SaleID         string    `json:"sale_id"`
	PaymentType    string    `json:"payment_type"` // dinheiro, cartão crédito, cartão débito, etc
	Amount         float64   `json:"amount"`
	CardBrand      string    `json:"card_brand,omitempty"`
	CardLastDigits string    `json:"card_last_digits,omitempty"`
	Installments   int       `json:"installments"`
	Status         string    `json:"status"` // pendente, aprovado, recusado
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// Supplier representa um fornecedor
type Supplier struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Document  string    `json:"document"`
	Phone     string    `json:"phone"`
	Email     string    `json:"email"`
	Address   string    `json:"address"`
	City      string    `json:"city"`
	State     string    `json:"state"`
	ZipCode   string    `json:"zip_code"`
	TenantID  string    `json:"tenant_id"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// PurchaseOrder representa uma ordem de compra
type PurchaseOrder struct {
	ID           string         `json:"id"`
	SupplierID   string         `json:"supplier_id"`
	Supplier     *Supplier      `json:"supplier,omitempty"`
	UserID       string         `json:"user_id"`
	TenantID     string         `json:"tenant_id"`
	Status       string         `json:"status"` // pendente, recebida, cancelada
	Total        float64        `json:"total"`
	ExpectedDate time.Time      `json:"expected_date"`
	ReceivedDate *time.Time     `json:"received_date,omitempty"`
	Items        []PurchaseItem `json:"items,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

// PurchaseItem representa um item de compra
type PurchaseItem struct {
	ID          string    `json:"id"`
	PurchaseID  string    `json:"purchase_id"`
	ProductID   string    `json:"product_id"`
	Product     *Product  `json:"product,omitempty"`
	Quantity    int       `json:"quantity"`
	ReceivedQty int       `json:"received_qty"`
	Price       float64   `json:"price"`
	Total       float64   `json:"total"`
	CreatedAt   time.Time `json:"created_at"`
}
