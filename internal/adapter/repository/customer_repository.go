package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hugohenrick/erp-supermercado/internal/domain/customer"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Erros específicos do repositório
var (
	ErrCustomerNotFound      = errors.New("cliente não encontrado")
	ErrCustomerDuplicateKey  = errors.New("cliente com mesmo documento já existe")
	ErrCustomerDatabaseError = errors.New("erro de banco de dados")
	ErrCustomerNotAllowed    = errors.New("operação não permitida para este cliente")
)

// CustomerRepository implementa a interface customer.Repository
type CustomerRepository struct {
	db *pgxpool.Pool
}

// NewCustomerRepository cria uma nova instância de CustomerRepository
func NewCustomerRepository(db *pgxpool.Pool) customer.Repository {
	return &CustomerRepository{
		db: db,
	}
}

// Create implementa customer.Repository.Create
func (r *CustomerRepository) Create(ctx context.Context, c *customer.Customer) error {
	// Verificar se já existe um cliente com o mesmo documento no tenant
	exists, err := r.ExistsByDocument(ctx, c.TenantID, c.Document)
	if err != nil {
		return fmt.Errorf("erro ao verificar existência do cliente: %w", err)
	}
	if exists {
		return ErrCustomerDuplicateKey
	}

	// Converter endereços e contatos para JSON
	addresses, err := json.Marshal(c.Addresses)
	if err != nil {
		return fmt.Errorf("erro ao converter endereços para JSON: %w", err)
	}

	contacts, err := json.Marshal(c.Contacts)
	if err != nil {
		return fmt.Errorf("erro ao converter contatos para JSON: %w", err)
	}

	// Inserir o cliente
	_, err = r.db.Exec(ctx,
		`INSERT INTO customers (
			id, tenant_id, branch_id, person_type, name, trade_name, document,
			state_document, city_document, tax_regime, customer_type, status,
			credit_limit, payment_term, website, observations, fiscal_notes,
			addresses, contacts, last_purchase_at, created_at, updated_at,
			external_code, salesman_id, price_table_id, payment_method_id,
			suframa, reference_code
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14,
			$15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28
		)`,
		c.ID, c.TenantID, c.BranchID, c.PersonType, c.Name, c.TradeName,
		c.Document, c.StateDocument, c.CityDocument, c.TaxRegime,
		c.CustomerType, c.Status, c.CreditLimit, c.PaymentTerm,
		c.Website, c.Observations, c.FiscalNotes, addresses, contacts,
		c.LastPurchaseAt, c.CreatedAt, c.UpdatedAt, c.ExternalCode,
		c.SalesmanID, c.PriceTableID, c.PaymentMethodID, c.SUFRAMA,
		c.ReferenceCode)

	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return ErrCustomerDuplicateKey
		}
		return fmt.Errorf("erro ao criar cliente: %w", err)
	}

	return nil
}

// FindByID implementa customer.Repository.FindByID
func (r *CustomerRepository) FindByID(ctx context.Context, id string) (*customer.Customer, error) {
	var c customer.Customer
	var addressesJSON, contactsJSON []byte

	err := r.db.QueryRow(ctx,
		`SELECT 
			id, tenant_id, branch_id, person_type, name, trade_name, document,
			state_document, city_document, tax_regime, customer_type, status,
			credit_limit, payment_term, website, observations, fiscal_notes,
			addresses, contacts, last_purchase_at, created_at, updated_at,
			external_code, salesman_id, price_table_id, payment_method_id,
			suframa, reference_code
		FROM customers WHERE id = $1`,
		id).Scan(
		&c.ID, &c.TenantID, &c.BranchID, &c.PersonType, &c.Name, &c.TradeName,
		&c.Document, &c.StateDocument, &c.CityDocument, &c.TaxRegime,
		&c.CustomerType, &c.Status, &c.CreditLimit, &c.PaymentTerm,
		&c.Website, &c.Observations, &c.FiscalNotes, &addressesJSON,
		&contactsJSON, &c.LastPurchaseAt, &c.CreatedAt, &c.UpdatedAt,
		&c.ExternalCode, &c.SalesmanID, &c.PriceTableID, &c.PaymentMethodID,
		&c.SUFRAMA, &c.ReferenceCode)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCustomerNotFound
		}
		return nil, fmt.Errorf("erro ao buscar cliente: %w", err)
	}

	// Converter JSON para structs
	if err := json.Unmarshal(addressesJSON, &c.Addresses); err != nil {
		return nil, fmt.Errorf("erro ao converter endereços: %w", err)
	}

	if err := json.Unmarshal(contactsJSON, &c.Contacts); err != nil {
		return nil, fmt.Errorf("erro ao converter contatos: %w", err)
	}

	return &c, nil
}

// FindByDocument implementa customer.Repository.FindByDocument
func (r *CustomerRepository) FindByDocument(ctx context.Context, tenantID, document string) (*customer.Customer, error) {
	var c customer.Customer
	var addressesJSON, contactsJSON []byte

	err := r.db.QueryRow(ctx,
		`SELECT 
			id, tenant_id, branch_id, person_type, name, trade_name, document,
			state_document, city_document, tax_regime, customer_type, status,
			credit_limit, payment_term, website, observations, fiscal_notes,
			addresses, contacts, last_purchase_at, created_at, updated_at,
			external_code, salesman_id, price_table_id, payment_method_id,
			suframa, reference_code
		FROM customers WHERE tenant_id = $1 AND document = $2`,
		tenantID, document).Scan(
		&c.ID, &c.TenantID, &c.BranchID, &c.PersonType, &c.Name, &c.TradeName,
		&c.Document, &c.StateDocument, &c.CityDocument, &c.TaxRegime,
		&c.CustomerType, &c.Status, &c.CreditLimit, &c.PaymentTerm,
		&c.Website, &c.Observations, &c.FiscalNotes, &addressesJSON,
		&contactsJSON, &c.LastPurchaseAt, &c.CreatedAt, &c.UpdatedAt,
		&c.ExternalCode, &c.SalesmanID, &c.PriceTableID, &c.PaymentMethodID,
		&c.SUFRAMA, &c.ReferenceCode)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCustomerNotFound
		}
		return nil, fmt.Errorf("erro ao buscar cliente: %w", err)
	}

	// Converter JSON para structs
	if err := json.Unmarshal(addressesJSON, &c.Addresses); err != nil {
		return nil, fmt.Errorf("erro ao converter endereços: %w", err)
	}

	if err := json.Unmarshal(contactsJSON, &c.Contacts); err != nil {
		return nil, fmt.Errorf("erro ao converter contatos: %w", err)
	}

	return &c, nil
}

// FindByBranch implementa customer.Repository.FindByBranch
func (r *CustomerRepository) FindByBranch(ctx context.Context, branchID string, limit, offset int) ([]*customer.Customer, error) {
	rows, err := r.db.Query(ctx,
		`SELECT 
			id, tenant_id, branch_id, person_type, name, trade_name, document,
			state_document, city_document, tax_regime, customer_type, status,
			credit_limit, payment_term, website, observations, fiscal_notes,
			addresses, contacts, last_purchase_at, created_at, updated_at,
			external_code, salesman_id, price_table_id, payment_method_id,
			suframa, reference_code
		FROM customers 
		WHERE branch_id = $1
		ORDER BY name ASC
		LIMIT $2 OFFSET $3`,
		branchID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar clientes: %w", err)
	}
	defer rows.Close()

	return r.scanCustomerRows(rows)
}

// List implementa customer.Repository.List
func (r *CustomerRepository) List(ctx context.Context, tenantID string, limit, offset int) ([]*customer.Customer, error) {
	rows, err := r.db.Query(ctx,
		`SELECT 
			id, tenant_id, branch_id, person_type, name, trade_name, document,
			state_document, city_document, tax_regime, customer_type, status,
			credit_limit, payment_term, website, observations, fiscal_notes,
			addresses, contacts, last_purchase_at, created_at, updated_at,
			external_code, salesman_id, price_table_id, payment_method_id,
			suframa, reference_code
		FROM customers 
		WHERE tenant_id = $1
		ORDER BY name ASC
		LIMIT $2 OFFSET $3`,
		tenantID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar clientes: %w", err)
	}
	defer rows.Close()

	return r.scanCustomerRows(rows)
}

// Update implementa customer.Repository.Update
func (r *CustomerRepository) Update(ctx context.Context, c *customer.Customer) error {
	// Verificar se o cliente existe
	exists, err := r.Exists(ctx, c.ID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrCustomerNotFound
	}

	// Converter endereços e contatos para JSON
	addresses, err := json.Marshal(c.Addresses)
	if err != nil {
		return fmt.Errorf("erro ao converter endereços para JSON: %w", err)
	}

	contacts, err := json.Marshal(c.Contacts)
	if err != nil {
		return fmt.Errorf("erro ao converter contatos para JSON: %w", err)
	}

	// Atualizar o cliente
	_, err = r.db.Exec(ctx,
		`UPDATE customers SET
			person_type = $1, name = $2, trade_name = $3, document = $4,
			state_document = $5, city_document = $6, tax_regime = $7,
			customer_type = $8, status = $9, credit_limit = $10,
			payment_term = $11, website = $12, observations = $13,
			fiscal_notes = $14, addresses = $15, contacts = $16,
			last_purchase_at = $17, updated_at = $18, external_code = $19,
			salesman_id = $20, price_table_id = $21, payment_method_id = $22,
			suframa = $23, reference_code = $24
		WHERE id = $25 AND tenant_id = $26`,
		c.PersonType, c.Name, c.TradeName, c.Document, c.StateDocument,
		c.CityDocument, c.TaxRegime, c.CustomerType, c.Status, c.CreditLimit,
		c.PaymentTerm, c.Website, c.Observations, c.FiscalNotes, addresses,
		contacts, c.LastPurchaseAt, c.UpdatedAt, c.ExternalCode, c.SalesmanID,
		c.PriceTableID, c.PaymentMethodID, c.SUFRAMA, c.ReferenceCode,
		c.ID, c.TenantID)

	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return ErrCustomerDuplicateKey
		}
		return fmt.Errorf("erro ao atualizar cliente: %w", err)
	}

	return nil
}

// Delete implementa customer.Repository.Delete
func (r *CustomerRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.Exec(ctx, "DELETE FROM customers WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("erro ao excluir cliente: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrCustomerNotFound
	}

	return nil
}

// UpdateStatus implementa customer.Repository.UpdateStatus
func (r *CustomerRepository) UpdateStatus(ctx context.Context, id string, status customer.Status) error {
	result, err := r.db.Exec(ctx,
		"UPDATE customers SET status = $1, updated_at = $2 WHERE id = $3",
		status, time.Now(), id)

	if err != nil {
		return fmt.Errorf("erro ao atualizar status do cliente: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrCustomerNotFound
	}

	return nil
}

// CountByTenant implementa customer.Repository.CountByTenant
func (r *CustomerRepository) CountByTenant(ctx context.Context, tenantID string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		"SELECT COUNT(*) FROM customers WHERE tenant_id = $1",
		tenantID).Scan(&count)

	if err != nil {
		return 0, fmt.Errorf("erro ao contar clientes: %w", err)
	}

	return count, nil
}

// CountByBranch implementa customer.Repository.CountByBranch
func (r *CustomerRepository) CountByBranch(ctx context.Context, branchID string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		"SELECT COUNT(*) FROM customers WHERE branch_id = $1",
		branchID).Scan(&count)

	if err != nil {
		return 0, fmt.Errorf("erro ao contar clientes: %w", err)
	}

	return count, nil
}

// Exists implementa customer.Repository.Exists
func (r *CustomerRepository) Exists(ctx context.Context, id string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM customers WHERE id = $1)",
		id).Scan(&exists)

	if err != nil {
		return false, fmt.Errorf("erro ao verificar existência do cliente: %w", err)
	}

	return exists, nil
}

// ExistsByDocument implementa customer.Repository.ExistsByDocument
func (r *CustomerRepository) ExistsByDocument(ctx context.Context, tenantID, document string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM customers WHERE tenant_id = $1 AND document = $2)",
		tenantID, document).Scan(&exists)

	if err != nil {
		return false, fmt.Errorf("erro ao verificar existência do cliente: %w", err)
	}

	return exists, nil
}

// FindByName implementa customer.Repository.FindByName
func (r *CustomerRepository) FindByName(ctx context.Context, tenantID, name string, limit, offset int) ([]*customer.Customer, error) {
	rows, err := r.db.Query(ctx,
		`SELECT 
			id, tenant_id, branch_id, person_type, name, trade_name, document,
			state_document, city_document, tax_regime, customer_type, status,
			credit_limit, payment_term, website, observations, fiscal_notes,
			addresses, contacts, last_purchase_at, created_at, updated_at,
			external_code, salesman_id, price_table_id, payment_method_id,
			suframa, reference_code
		FROM customers 
		WHERE tenant_id = $1 AND name ILIKE $2
		ORDER BY name ASC
		LIMIT $3 OFFSET $4`,
		tenantID, "%"+name+"%", limit, offset)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar clientes: %w", err)
	}
	defer rows.Close()

	return r.scanCustomerRows(rows)
}

// FindByType implementa customer.Repository.FindByType
func (r *CustomerRepository) FindByType(ctx context.Context, tenantID string, customerType customer.CustomerType, limit, offset int) ([]*customer.Customer, error) {
	rows, err := r.db.Query(ctx,
		`SELECT 
			id, tenant_id, branch_id, person_type, name, trade_name, document,
			state_document, city_document, tax_regime, customer_type, status,
			credit_limit, payment_term, website, observations, fiscal_notes,
			addresses, contacts, last_purchase_at, created_at, updated_at,
			external_code, salesman_id, price_table_id, payment_method_id,
			suframa, reference_code
		FROM customers 
		WHERE tenant_id = $1 AND customer_type = $2
		ORDER BY name ASC
		LIMIT $3 OFFSET $4`,
		tenantID, customerType, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar clientes: %w", err)
	}
	defer rows.Close()

	return r.scanCustomerRows(rows)
}

// FindBySalesman implementa customer.Repository.FindBySalesman
func (r *CustomerRepository) FindBySalesman(ctx context.Context, salesmanID string, limit, offset int) ([]*customer.Customer, error) {
	rows, err := r.db.Query(ctx,
		`SELECT 
			id, tenant_id, branch_id, person_type, name, trade_name, document,
			state_document, city_document, tax_regime, customer_type, status,
			credit_limit, payment_term, website, observations, fiscal_notes,
			addresses, contacts, last_purchase_at, created_at, updated_at,
			external_code, salesman_id, price_table_id, payment_method_id,
			suframa, reference_code
		FROM customers 
		WHERE salesman_id = $1
		ORDER BY name ASC
		LIMIT $2 OFFSET $3`,
		salesmanID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar clientes: %w", err)
	}
	defer rows.Close()

	return r.scanCustomerRows(rows)
}

// FindByPriceTable implementa customer.Repository.FindByPriceTable
func (r *CustomerRepository) FindByPriceTable(ctx context.Context, priceTableID string, limit, offset int) ([]*customer.Customer, error) {
	rows, err := r.db.Query(ctx,
		`SELECT 
			id, tenant_id, branch_id, person_type, name, trade_name, document,
			state_document, city_document, tax_regime, customer_type, status,
			credit_limit, payment_term, website, observations, fiscal_notes,
			addresses, contacts, last_purchase_at, created_at, updated_at,
			external_code, salesman_id, price_table_id, payment_method_id,
			suframa, reference_code
		FROM customers 
		WHERE price_table_id = $1
		ORDER BY name ASC
		LIMIT $2 OFFSET $3`,
		priceTableID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar clientes: %w", err)
	}
	defer rows.Close()

	return r.scanCustomerRows(rows)
}

// FindByPaymentMethod implementa customer.Repository.FindByPaymentMethod
func (r *CustomerRepository) FindByPaymentMethod(ctx context.Context, paymentMethodID string, limit, offset int) ([]*customer.Customer, error) {
	rows, err := r.db.Query(ctx,
		`SELECT 
			id, tenant_id, branch_id, person_type, name, trade_name, document,
			state_document, city_document, tax_regime, customer_type, status,
			credit_limit, payment_term, website, observations, fiscal_notes,
			addresses, contacts, last_purchase_at, created_at, updated_at,
			external_code, salesman_id, price_table_id, payment_method_id,
			suframa, reference_code
		FROM customers 
		WHERE payment_method_id = $1
		ORDER BY name ASC
		LIMIT $2 OFFSET $3`,
		paymentMethodID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar clientes: %w", err)
	}
	defer rows.Close()

	return r.scanCustomerRows(rows)
}

// FindByStatus implementa customer.Repository.FindByStatus
func (r *CustomerRepository) FindByStatus(ctx context.Context, tenantID string, status customer.Status, limit, offset int) ([]*customer.Customer, error) {
	rows, err := r.db.Query(ctx,
		`SELECT 
			id, tenant_id, branch_id, person_type, name, trade_name, document,
			state_document, city_document, tax_regime, customer_type, status,
			credit_limit, payment_term, website, observations, fiscal_notes,
			addresses, contacts, last_purchase_at, created_at, updated_at,
			external_code, salesman_id, price_table_id, payment_method_id,
			suframa, reference_code
		FROM customers 
		WHERE tenant_id = $1 AND status = $2
		ORDER BY name ASC
		LIMIT $3 OFFSET $4`,
		tenantID, status, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar clientes: %w", err)
	}
	defer rows.Close()

	return r.scanCustomerRows(rows)
}

// FindByTaxRegime implementa customer.Repository.FindByTaxRegime
func (r *CustomerRepository) FindByTaxRegime(ctx context.Context, tenantID string, taxRegime customer.TaxRegime, limit, offset int) ([]*customer.Customer, error) {
	rows, err := r.db.Query(ctx,
		`SELECT 
			id, tenant_id, branch_id, person_type, name, trade_name, document,
			state_document, city_document, tax_regime, customer_type, status,
			credit_limit, payment_term, website, observations, fiscal_notes,
			addresses, contacts, last_purchase_at, created_at, updated_at,
			external_code, salesman_id, price_table_id, payment_method_id,
			suframa, reference_code
		FROM customers 
		WHERE tenant_id = $1 AND tax_regime = $2
		ORDER BY name ASC
		LIMIT $3 OFFSET $4`,
		tenantID, taxRegime, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar clientes: %w", err)
	}
	defer rows.Close()

	return r.scanCustomerRows(rows)
}

// UpdateCreditLimit implementa customer.Repository.UpdateCreditLimit
func (r *CustomerRepository) UpdateCreditLimit(ctx context.Context, id string, creditLimit float64) error {
	result, err := r.db.Exec(ctx,
		"UPDATE customers SET credit_limit = $1, updated_at = $2 WHERE id = $3",
		creditLimit, time.Now(), id)

	if err != nil {
		return fmt.Errorf("erro ao atualizar limite de crédito: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrCustomerNotFound
	}

	return nil
}

// UpdatePaymentTerm implementa customer.Repository.UpdatePaymentTerm
func (r *CustomerRepository) UpdatePaymentTerm(ctx context.Context, id string, paymentTerm int) error {
	result, err := r.db.Exec(ctx,
		"UPDATE customers SET payment_term = $1, updated_at = $2 WHERE id = $3",
		paymentTerm, time.Now(), id)

	if err != nil {
		return fmt.Errorf("erro ao atualizar prazo de pagamento: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrCustomerNotFound
	}

	return nil
}

// scanCustomerRows é um método auxiliar para processar resultados de consultas que retornam múltiplos clientes
func (r *CustomerRepository) scanCustomerRows(rows pgx.Rows) ([]*customer.Customer, error) {
	customers := make([]*customer.Customer, 0)

	for rows.Next() {
		var c customer.Customer
		var addressesJSON, contactsJSON []byte

		err := rows.Scan(
			&c.ID, &c.TenantID, &c.BranchID, &c.PersonType, &c.Name, &c.TradeName,
			&c.Document, &c.StateDocument, &c.CityDocument, &c.TaxRegime,
			&c.CustomerType, &c.Status, &c.CreditLimit, &c.PaymentTerm,
			&c.Website, &c.Observations, &c.FiscalNotes, &addressesJSON,
			&contactsJSON, &c.LastPurchaseAt, &c.CreatedAt, &c.UpdatedAt,
			&c.ExternalCode, &c.SalesmanID, &c.PriceTableID, &c.PaymentMethodID,
			&c.SUFRAMA, &c.ReferenceCode)

		if err != nil {
			return nil, fmt.Errorf("erro ao ler cliente: %w", err)
		}

		// Converter JSON para structs
		if err := json.Unmarshal(addressesJSON, &c.Addresses); err != nil {
			return nil, fmt.Errorf("erro ao converter endereços: %w", err)
		}

		if err := json.Unmarshal(contactsJSON, &c.Contacts); err != nil {
			return nil, fmt.Errorf("erro ao converter contatos: %w", err)
		}

		customers = append(customers, &c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erro ao ler resultados: %w", err)
	}

	return customers, nil
}
