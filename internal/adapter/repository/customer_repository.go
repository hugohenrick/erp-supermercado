package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hugohenrick/erp-supermercado/internal/domain/customer"
	pkgbranch "github.com/hugohenrick/erp-supermercado/pkg/branch"
	pkgtenant "github.com/hugohenrick/erp-supermercado/pkg/tenant"
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

// mapTaxRegime converte valores de frontend para os valores de enum do banco de dados
func mapTaxRegime(frontendValue string) string {
	switch strings.ToLower(frontendValue) {
	case "simples", "simple":
		return "SIMPLE"
	case "mei":
		return "SIMPLE" // Consideramos MEI como um tipo de SIMPLE
	case "presumido", "presumed":
		return "PRESUMED"
	case "real":
		return "REAL"
	default:
		return "SIMPLE" // Valor padrão
	}
}

// mapCustomerType converte valores de frontend para os valores de enum do banco de dados
func mapCustomerType(frontendValue string) string {
	switch strings.ToLower(frontendValue) {
	case "final", "customer":
		return "CUSTOMER"
	case "reseller", "supplier":
		return "SUPPLIER"
	case "wholesale", "carrier":
		return "CARRIER"
	default:
		return "CUSTOMER" // Valor padrão
	}
}

// mapCustomerStatus converte valores de frontend para os valores de enum do banco de dados
func mapCustomerStatus(frontendValue string) string {
	switch strings.ToLower(frontendValue) {
	case "active", "ativo":
		return "ACTIVE"
	case "inactive", "inativo":
		return "INACTIVE"
	case "blocked", "bloqueado":
		return "BLOCKED"
	default:
		return "ACTIVE" // Valor padrão
	}
}

// Create implementa customer.Repository.Create
func (r *CustomerRepository) Create(ctx context.Context, c *customer.Customer) error {
	conn, err := r.db.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("falha ao obter conexão: %w", err)
	}
	defer conn.Release()

	// Verificar o tenant ID do customer e do contexto
	tenantIDFromContext := pkgtenant.GetTenantID(ctx)
	fmt.Printf("DEBUG Create Customer - TenantID do customer: '%s', TenantID do contexto: '%s'\n", c.TenantID, tenantIDFromContext)

	// Se o tenant ID do contexto for válido e diferente do tenant ID do customer, vamos usar o do contexto
	if tenantIDFromContext != "" && c.TenantID != tenantIDFromContext {
		fmt.Printf("DEBUG Create Customer - Substituindo TenantID do customer pelo do contexto\n")
		c.TenantID = tenantIDFromContext
	}

	// Verificar se o ID do cliente está vazio e gerar um novo se necessário
	if c.ID == "" {
		c.ID = uuid.New().String()
		fmt.Printf("DEBUG Create Customer - ID vazio, gerando novo: %s\n", c.ID)
	}

	// Primeiro, definir o search_path para public para garantir que acessamos os tenants
	_, err = conn.Exec(ctx, "SET search_path TO public")
	if err != nil {
		return fmt.Errorf("falha ao configurar search_path: %w", err)
	}

	// Obter o schema do tenant a partir do tenant_id
	var schema string
	err = conn.QueryRow(ctx, "SELECT schema FROM public.tenants WHERE id = $1", c.TenantID).Scan(&schema)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errors.New("tenant não encontrado")
		}
		return fmt.Errorf("falha ao obter schema do tenant: %w", err)
	}

	// Debug para verificar o schema recuperado
	fmt.Printf("DEBUG Create Customer - Tenant ID: %s, Schema: %s\n", c.TenantID, schema)

	// Verificar se já existe um cliente com o mesmo documento no tenant
	exists, err := r.ExistsByDocument(ctx, c.TenantID, c.Document)
	if err != nil {
		return fmt.Errorf("erro ao verificar existência do cliente: %w", err)
	}
	if exists {
		return ErrCustomerDuplicateKey
	}

	// Preparar branch_id
	var branchID string

	// Verificar se o branch_id está vazio
	if c.BranchID == "" {
		// Tentar obter do cabeçalho da requisição
		branchIDFromHeader := getBranchIDFromContext(ctx)
		if branchIDFromHeader != "" {
			branchID = branchIDFromHeader
			fmt.Printf("DEBUG Create Customer - Usando BranchID do cabeçalho: %s\n", branchID)
		} else {
			// Se ainda não tiver branch_id, buscar a filial principal do tenant
			mainBranch, err := findMainBranch(ctx, conn, schema, c.TenantID)
			if err != nil {
				return fmt.Errorf("erro ao buscar filial principal: %w", err)
			}

			// Se encontrou a filial principal, usar seu ID
			if mainBranch != "" {
				branchID = mainBranch
				fmt.Printf("DEBUG Create Customer - Usando BranchID da filial principal: %s\n", branchID)
			} else {
				return errors.New("não foi possível determinar a filial - o branch_id é obrigatório")
			}
		}
	} else {
		branchID = c.BranchID
	}

	// Preparar outros IDs relacionados, convertendo strings vazias para nil (NULL no banco)
	var salesmanID, priceTableID, paymentMethodID interface{}

	// Tratar salesman_id
	if c.SalesmanID != "" {
		salesmanID = c.SalesmanID
	} else {
		salesmanID = nil
		fmt.Printf("DEBUG Create Customer - SalesmanID vazio, usando NULL\n")
	}

	// Tratar price_table_id
	if c.PriceTableID != "" {
		priceTableID = c.PriceTableID
	} else {
		priceTableID = nil
		fmt.Printf("DEBUG Create Customer - PriceTableID vazio, usando NULL\n")
	}

	// Tratar payment_method_id
	if c.PaymentMethodID != "" {
		paymentMethodID = c.PaymentMethodID
	} else {
		paymentMethodID = nil
		fmt.Printf("DEBUG Create Customer - PaymentMethodID vazio, usando NULL\n")
	}

	// Converter valores de enum para o formato esperado pelo banco de dados
	taxRegimeDB := mapTaxRegime(string(c.TaxRegime))
	customerTypeDB := mapCustomerType(string(c.CustomerType))
	statusDB := mapCustomerStatus(string(c.Status))

	fmt.Printf("DEBUG Create Customer - Mapeamento de enums: TaxRegime '%s' -> '%s', CustomerType '%s' -> '%s', Status '%s' -> '%s'\n",
		c.TaxRegime, taxRegimeDB, c.CustomerType, customerTypeDB, c.Status, statusDB)

	// Converter endereços e contatos para JSON
	addresses, err := json.Marshal(c.Addresses)
	if err != nil {
		return fmt.Errorf("erro ao converter endereços para JSON: %w", err)
	}

	contacts, err := json.Marshal(c.Contacts)
	if err != nil {
		return fmt.Errorf("erro ao converter contatos para JSON: %w", err)
	}

	// Atualizar timestamps
	now := time.Now()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = now
	}

	// Construir a query usando o schema específico do tenant
	query := fmt.Sprintf(`INSERT INTO %s.customers (
		id, tenant_id, branch_id, person_type, name, trade_name, document,
		state_document, city_document, tax_regime, customer_type, status,
		credit_limit, payment_term, website, observations, fiscal_notes,
		addresses, contacts, last_purchase_at, created_at, updated_at,
		external_code, salesman_id, price_table_id, payment_method_id,
		suframa, reference_code
	) VALUES (
		$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14,
		$15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28
	)`, schema)

	// Inserir o cliente no schema específico do tenant
	_, err = conn.Exec(ctx, query,
		c.ID, c.TenantID, branchID, c.PersonType, c.Name, c.TradeName,
		c.Document, c.StateDocument, c.CityDocument, taxRegimeDB,
		customerTypeDB, statusDB, c.CreditLimit, c.PaymentTerm,
		c.Website, c.Observations, c.FiscalNotes, addresses, contacts,
		c.LastPurchaseAt, c.CreatedAt, c.UpdatedAt, c.ExternalCode,
		salesmanID, priceTableID, paymentMethodID, c.SUFRAMA,
		c.ReferenceCode)

	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return ErrCustomerDuplicateKey
		}
		return fmt.Errorf("erro ao criar cliente: %w", err)
	}

	return nil
}

// getBranchIDFromContext extrai o branch_id do contexto da requisição
func getBranchIDFromContext(ctx context.Context) string {
	// Primeiro, tentar obter do pacote branch
	branchID := pkgbranch.GetBranchID(ctx)
	if branchID != "" {
		return branchID
	}

	// Use type assertion para acessar o contexto Gin, se disponível
	gc, ok := ctx.(*gin.Context)
	if ok {
		return gc.GetHeader("branch-id")
	}

	// Se não for um contexto Gin, pode ser um contexto personalizado
	// que tem o branch_id armazenado de outra forma
	if branchID, ok := ctx.Value("branch_id").(string); ok {
		return branchID
	}

	return ""
}

// findMainBranch busca o ID da filial principal do tenant
func findMainBranch(ctx context.Context, conn *pgxpool.Conn, schema string, tenantID string) (string, error) {
	// Buscar filial principal do tenant
	query := fmt.Sprintf("SELECT id FROM %s.branches WHERE tenant_id = $1 AND is_main = true LIMIT 1", schema)
	var branchID string
	err := conn.QueryRow(ctx, query, tenantID).Scan(&branchID)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Se não houver filial principal, buscar qualquer filial
			query = fmt.Sprintf("SELECT id FROM %s.branches WHERE tenant_id = $1 LIMIT 1", schema)
			err = conn.QueryRow(ctx, query, tenantID).Scan(&branchID)

			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return "", errors.New("nenhuma filial encontrada para este tenant")
				}
				return "", fmt.Errorf("erro ao buscar filial: %w", err)
			}
		} else {
			return "", fmt.Errorf("erro ao buscar filial principal: %w", err)
		}
	}

	return branchID, nil
}

// FindByID implementa customer.Repository.FindByID
func (r *CustomerRepository) FindByID(ctx context.Context, id string) (*customer.Customer, error) {
	conn, err := r.db.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("falha ao obter conexão: %w", err)
	}
	defer conn.Release()

	// Obter tenant ID do contexto
	tenantID := pkgtenant.GetTenantID(ctx)
	if tenantID == "" {
		return nil, errors.New("tenant ID não encontrado no contexto")
	}

	// Primeiro, definir o search_path para public para garantir que acessamos os tenants
	_, err = conn.Exec(ctx, "SET search_path TO public")
	if err != nil {
		return nil, fmt.Errorf("falha ao configurar search_path: %w", err)
	}

	// Obter o schema do tenant a partir do tenant_id
	var schema string
	err = conn.QueryRow(ctx, "SELECT schema FROM public.tenants WHERE id = $1", tenantID).Scan(&schema)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("tenant não encontrado")
		}
		return nil, fmt.Errorf("falha ao obter schema do tenant: %w", err)
	}

	// Debug para verificar o schema recuperado
	fmt.Printf("DEBUG FindByID Customer - Tenant ID: %s, Schema: %s\n", tenantID, schema)

	var c customer.Customer
	var addressesJSON, contactsJSON []byte

	// Usar variáveis para valores que podem ser nulos
	var branchID, salesmanID, priceTableID, paymentMethodID, externalCode, suframa, referenceCode sql.NullString
	var lastPurchaseAt sql.NullTime

	// Construir a query usando o schema específico do tenant
	query := fmt.Sprintf(`SELECT 
		id, tenant_id, branch_id, person_type, name, trade_name, document,
		state_document, city_document, tax_regime, customer_type, status,
		credit_limit, payment_term, website, observations, fiscal_notes,
		addresses, contacts, last_purchase_at, created_at, updated_at,
		external_code, salesman_id, price_table_id, payment_method_id,
		suframa, reference_code
	FROM %s.customers WHERE id = $1`, schema)

	err = conn.QueryRow(ctx, query, id).Scan(
		&c.ID, &c.TenantID, &branchID, &c.PersonType, &c.Name, &c.TradeName,
		&c.Document, &c.StateDocument, &c.CityDocument, &c.TaxRegime,
		&c.CustomerType, &c.Status, &c.CreditLimit, &c.PaymentTerm,
		&c.Website, &c.Observations, &c.FiscalNotes, &addressesJSON,
		&contactsJSON, &lastPurchaseAt, &c.CreatedAt, &c.UpdatedAt,
		&externalCode, &salesmanID, &priceTableID, &paymentMethodID,
		&suframa, &referenceCode)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCustomerNotFound
		}
		return nil, fmt.Errorf("erro ao buscar cliente: %w", err)
	}

	// Atribuir valores nulos aos campos da estrutura apenas se forem válidos
	if branchID.Valid {
		c.BranchID = branchID.String
	}
	if salesmanID.Valid {
		c.SalesmanID = salesmanID.String
	}
	if priceTableID.Valid {
		c.PriceTableID = priceTableID.String
	}
	if paymentMethodID.Valid {
		c.PaymentMethodID = paymentMethodID.String
	}
	if externalCode.Valid {
		c.ExternalCode = externalCode.String
	}
	if suframa.Valid {
		c.SUFRAMA = suframa.String
	}
	if referenceCode.Valid {
		c.ReferenceCode = referenceCode.String
	}
	if lastPurchaseAt.Valid {
		c.LastPurchaseAt = &lastPurchaseAt.Time
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
	conn, err := r.db.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("falha ao obter conexão: %w", err)
	}
	defer conn.Release()

	// Primeiro, definir o search_path para public para garantir que acessamos os tenants
	_, err = conn.Exec(ctx, "SET search_path TO public")
	if err != nil {
		return nil, fmt.Errorf("falha ao configurar search_path: %w", err)
	}

	// Obter o schema do tenant a partir do tenant_id
	var schema string
	err = conn.QueryRow(ctx, "SELECT schema FROM public.tenants WHERE id = $1", tenantID).Scan(&schema)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("tenant não encontrado")
		}
		return nil, fmt.Errorf("falha ao obter schema do tenant: %w", err)
	}

	// Debug para verificar o schema recuperado
	fmt.Printf("DEBUG FindByDocument Customer - Tenant ID: %s, Schema: %s\n", tenantID, schema)

	var c customer.Customer
	var addressesJSON, contactsJSON []byte

	// Usar variáveis para valores que podem ser nulos
	var branchID, salesmanID, priceTableID, paymentMethodID, externalCode, suframa, referenceCode sql.NullString
	var lastPurchaseAt sql.NullTime

	// Construir a query usando o schema específico do tenant
	query := fmt.Sprintf(`SELECT 
		id, tenant_id, branch_id, person_type, name, trade_name, document,
		state_document, city_document, tax_regime, customer_type, status,
		credit_limit, payment_term, website, observations, fiscal_notes,
		addresses, contacts, last_purchase_at, created_at, updated_at,
		external_code, salesman_id, price_table_id, payment_method_id,
		suframa, reference_code
	FROM %s.customers WHERE tenant_id = $1 AND document = $2`, schema)

	err = conn.QueryRow(ctx, query, tenantID, document).Scan(
		&c.ID, &c.TenantID, &branchID, &c.PersonType, &c.Name, &c.TradeName,
		&c.Document, &c.StateDocument, &c.CityDocument, &c.TaxRegime,
		&c.CustomerType, &c.Status, &c.CreditLimit, &c.PaymentTerm,
		&c.Website, &c.Observations, &c.FiscalNotes, &addressesJSON,
		&contactsJSON, &lastPurchaseAt, &c.CreatedAt, &c.UpdatedAt,
		&externalCode, &salesmanID, &priceTableID, &paymentMethodID,
		&suframa, &referenceCode)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCustomerNotFound
		}
		return nil, fmt.Errorf("erro ao buscar cliente: %w", err)
	}

	// Atribuir valores nulos aos campos da estrutura apenas se forem válidos
	if branchID.Valid {
		c.BranchID = branchID.String
	}
	if salesmanID.Valid {
		c.SalesmanID = salesmanID.String
	}
	if priceTableID.Valid {
		c.PriceTableID = priceTableID.String
	}
	if paymentMethodID.Valid {
		c.PaymentMethodID = paymentMethodID.String
	}
	if externalCode.Valid {
		c.ExternalCode = externalCode.String
	}
	if suframa.Valid {
		c.SUFRAMA = suframa.String
	}
	if referenceCode.Valid {
		c.ReferenceCode = referenceCode.String
	}
	if lastPurchaseAt.Valid {
		c.LastPurchaseAt = &lastPurchaseAt.Time
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
	conn, err := r.db.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("falha ao obter conexão: %w", err)
	}
	defer conn.Release()

	// Obter tenant ID do contexto
	tenantID := pkgtenant.GetTenantID(ctx)
	if tenantID == "" {
		return nil, errors.New("tenant ID não encontrado no contexto")
	}

	// Primeiro, definir o search_path para public para garantir que acessamos os tenants
	_, err = conn.Exec(ctx, "SET search_path TO public")
	if err != nil {
		return nil, fmt.Errorf("falha ao configurar search_path: %w", err)
	}

	// Obter o schema do tenant a partir do tenant_id
	var schema string
	err = conn.QueryRow(ctx, "SELECT schema FROM public.tenants WHERE id = $1", tenantID).Scan(&schema)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("tenant não encontrado")
		}
		return nil, fmt.Errorf("falha ao obter schema do tenant: %w", err)
	}

	// Debug para verificar o schema recuperado
	fmt.Printf("DEBUG FindByBranch Customer - Tenant ID: %s, Schema: %s, Branch ID: %s\n", tenantID, schema, branchID)

	// Construir a query usando o schema específico do tenant - agora filtrando por tenant_id também
	query := fmt.Sprintf(`SELECT 
		id, tenant_id, branch_id, person_type, name, trade_name, document,
		state_document, city_document, tax_regime, customer_type, status,
		credit_limit, payment_term, website, observations, fiscal_notes,
		addresses, contacts, last_purchase_at, created_at, updated_at,
		external_code, salesman_id, price_table_id, payment_method_id,
		suframa, reference_code
	FROM %s.customers 
	WHERE tenant_id = $1 AND branch_id = $2
	ORDER BY name ASC
	LIMIT $3 OFFSET $4`, schema)

	rows, err := conn.Query(ctx, query, tenantID, branchID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar clientes por filial: %w", err)
	}
	defer rows.Close()

	return r.scanCustomerRows(rows)
}

// List implementa customer.Repository.List
func (r *CustomerRepository) List(ctx context.Context, tenantID string, limit, offset int) ([]*customer.Customer, error) {
	conn, err := r.db.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("falha ao obter conexão: %w", err)
	}
	defer conn.Release()

	// Primeiro, definir o search_path para public para garantir que acessamos os tenants
	_, err = conn.Exec(ctx, "SET search_path TO public")
	if err != nil {
		return nil, fmt.Errorf("falha ao configurar search_path: %w", err)
	}

	// Obter o schema do tenant a partir do tenant_id
	var schema string
	err = conn.QueryRow(ctx, "SELECT schema FROM public.tenants WHERE id = $1", tenantID).Scan(&schema)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("tenant não encontrado")
		}
		return nil, fmt.Errorf("falha ao obter schema do tenant: %w", err)
	}

	// Debug para verificar o schema recuperado
	fmt.Printf("DEBUG List Customer - Tenant ID: %s, Schema: %s\n", tenantID, schema)

	// Obter o branch ID do contexto
	var branchID string

	// Primeiro tenta obter do pacote branch
	branchID = pkgbranch.GetBranchID(ctx)

	if branchID == "" {
		// Se não encontrar usando o pacote branch, tentar obter do gin.Context
		gc, ok := ctx.(*gin.Context)
		if ok {
			branchID = gc.GetHeader("branch-id")
			if branchID == "" {
				branchID = gc.GetString("branch_id")
			}
		}
	}

	// Se ainda estiver vazio, tentar obter diretamente do valor do contexto
	if branchID == "" {
		if val, ok := ctx.Value("branch_id").(string); ok && val != "" {
			branchID = val
		}
	}

	var rows pgx.Rows

	// Construir a query usando o schema específico do tenant
	if branchID != "" {
		// Se tivermos um branch ID, filtrar por tenant_id e branch_id
		fmt.Printf("DEBUG List Customer - Filtrando por Branch ID: %s\n", branchID)
		query := fmt.Sprintf(`SELECT 
			id, tenant_id, branch_id, person_type, name, trade_name, document,
			state_document, city_document, tax_regime, customer_type, status,
			credit_limit, payment_term, website, observations, fiscal_notes,
			addresses, contacts, last_purchase_at, created_at, updated_at,
			external_code, salesman_id, price_table_id, payment_method_id,
			suframa, reference_code
		FROM %s.customers 
		WHERE tenant_id = $1 AND branch_id = $2
		ORDER BY name ASC
		LIMIT $3 OFFSET $4`, schema)

		rows, err = conn.Query(ctx, query, tenantID, branchID, limit, offset)
	} else {
		// Se não tivermos branch ID, filtrar apenas por tenant_id (comportamento original)
		fmt.Printf("DEBUG List Customer - Nenhum Branch ID encontrado, listando todos os clientes do tenant\n")
		query := fmt.Sprintf(`SELECT 
			id, tenant_id, branch_id, person_type, name, trade_name, document,
			state_document, city_document, tax_regime, customer_type, status,
			credit_limit, payment_term, website, observations, fiscal_notes,
			addresses, contacts, last_purchase_at, created_at, updated_at,
			external_code, salesman_id, price_table_id, payment_method_id,
			suframa, reference_code
		FROM %s.customers 
		WHERE tenant_id = $1
		ORDER BY name ASC
		LIMIT $2 OFFSET $3`, schema)

		rows, err = conn.Query(ctx, query, tenantID, limit, offset)
	}

	if err != nil {
		return nil, fmt.Errorf("erro ao listar clientes: %w", err)
	}
	defer rows.Close()

	return r.scanCustomerRows(rows)
}

// Update implementa customer.Repository.Update
func (r *CustomerRepository) Update(ctx context.Context, c *customer.Customer) error {
	conn, err := r.db.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("falha ao obter conexão: %w", err)
	}
	defer conn.Release()

	// Obter tenant ID do contexto
	tenantID := pkgtenant.GetTenantID(ctx)
	if tenantID == "" {
		return errors.New("tenant ID não encontrado no contexto")
	}

	// Primeiro, definir o search_path para public para garantir que acessamos os tenants
	_, err = conn.Exec(ctx, "SET search_path TO public")
	if err != nil {
		return fmt.Errorf("falha ao configurar search_path: %w", err)
	}

	// Obter o schema do tenant a partir do tenant_id
	var schema string
	err = conn.QueryRow(ctx, "SELECT schema FROM public.tenants WHERE id = $1", tenantID).Scan(&schema)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errors.New("tenant não encontrado")
		}
		return fmt.Errorf("falha ao obter schema do tenant: %w", err)
	}

	// Debug para verificar o schema recuperado
	fmt.Printf("DEBUG Update Customer - Tenant ID: %s, Schema: %s, Customer ID: %s\n", tenantID, schema, c.ID)

	// Verificar se o cliente existe no schema específico
	var exists bool
	query := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s.customers WHERE id = $1 AND tenant_id = $2)", schema)
	err = conn.QueryRow(ctx, query, c.ID, tenantID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("erro ao verificar existência do cliente: %w", err)
	}
	if !exists {
		return ErrCustomerNotFound
	}

	// Obter o branch ID do contexto para verificar se o cliente pertence à filial atual
	branchID := pkgbranch.GetBranchID(ctx)
	if branchID != "" {
		// Verificar se o cliente pertence à filial atual
		var matchesBranch bool
		query := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s.customers WHERE id = $1 AND branch_id = $2)", schema)
		err = conn.QueryRow(ctx, query, c.ID, branchID).Scan(&matchesBranch)
		if err != nil {
			return fmt.Errorf("erro ao verificar filial do cliente: %w", err)
		}
		if !matchesBranch {
			return fmt.Errorf("cliente não pertence à filial atual: %w", ErrCustomerNotAllowed)
		}
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

	// Preparar outros IDs relacionados, convertendo strings vazias para nil (NULL no banco)
	var salesmanID, priceTableID, paymentMethodID interface{}

	// Tratar salesman_id
	if c.SalesmanID != "" {
		salesmanID = c.SalesmanID
	} else {
		salesmanID = nil
	}

	// Tratar price_table_id
	if c.PriceTableID != "" {
		priceTableID = c.PriceTableID
	} else {
		priceTableID = nil
	}

	// Tratar payment_method_id
	if c.PaymentMethodID != "" {
		paymentMethodID = c.PaymentMethodID
	} else {
		paymentMethodID = nil
	}

	// Converter valores de enum para o formato esperado pelo banco de dados
	taxRegimeDB := mapTaxRegime(string(c.TaxRegime))
	customerTypeDB := mapCustomerType(string(c.CustomerType))
	statusDB := mapCustomerStatus(string(c.Status))

	// Atualizar o cliente
	query = fmt.Sprintf(`UPDATE %s.customers SET
		person_type = $1, name = $2, trade_name = $3, document = $4,
		state_document = $5, city_document = $6, tax_regime = $7,
		customer_type = $8, status = $9, credit_limit = $10,
		payment_term = $11, website = $12, observations = $13,
		fiscal_notes = $14, addresses = $15, contacts = $16,
		last_purchase_at = $17, updated_at = $18, external_code = $19,
		salesman_id = $20, price_table_id = $21, payment_method_id = $22,
		suframa = $23, reference_code = $24
	WHERE id = $25 AND tenant_id = $26`, schema)

	_, err = conn.Exec(ctx, query,
		c.PersonType, c.Name, c.TradeName, c.Document, c.StateDocument,
		c.CityDocument, taxRegimeDB, customerTypeDB, statusDB, c.CreditLimit,
		c.PaymentTerm, c.Website, c.Observations, c.FiscalNotes, addresses,
		contacts, c.LastPurchaseAt, c.UpdatedAt, c.ExternalCode, salesmanID,
		priceTableID, paymentMethodID, c.SUFRAMA, c.ReferenceCode,
		c.ID, tenantID)

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
	conn, err := r.db.Acquire(ctx)
	if err != nil {
		return 0, fmt.Errorf("falha ao obter conexão: %w", err)
	}
	defer conn.Release()

	// Primeiro, definir o search_path para public para garantir que acessamos os tenants
	_, err = conn.Exec(ctx, "SET search_path TO public")
	if err != nil {
		return 0, fmt.Errorf("falha ao configurar search_path: %w", err)
	}

	// Obter o schema do tenant a partir do tenant_id
	var schema string
	err = conn.QueryRow(ctx, "SELECT schema FROM public.tenants WHERE id = $1", tenantID).Scan(&schema)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, errors.New("tenant não encontrado")
		}
		return 0, fmt.Errorf("falha ao obter schema do tenant: %w", err)
	}

	// Debug para verificar o schema recuperado
	fmt.Printf("DEBUG CountByTenant Customer - Tenant ID: %s, Schema: %s\n", tenantID, schema)

	// Obter o branch ID do contexto
	var branchID string

	// Primeiro tenta obter do pacote branch
	branchID = pkgbranch.GetBranchID(ctx)

	if branchID == "" {
		// Se não encontrar usando o pacote branch, tentar obter do gin.Context
		gc, ok := ctx.(*gin.Context)
		if ok {
			branchID = gc.GetHeader("branch-id")
			if branchID == "" {
				branchID = gc.GetString("branch_id")
			}
		}
	}

	// Se ainda estiver vazio, tentar obter diretamente do valor do contexto
	if branchID == "" {
		if val, ok := ctx.Value("branch_id").(string); ok && val != "" {
			branchID = val
		}
	}

	var count int

	// Construir a query usando o schema específico do tenant
	if branchID != "" {
		// Se tivermos um branch ID, filtrar por tenant_id e branch_id
		fmt.Printf("DEBUG CountByTenant Customer - Filtrando por Branch ID: %s\n", branchID)
		query := fmt.Sprintf("SELECT COUNT(*) FROM %s.customers WHERE tenant_id = $1 AND branch_id = $2", schema)
		err = conn.QueryRow(ctx, query, tenantID, branchID).Scan(&count)
	} else {
		// Se não tivermos branch ID, filtrar apenas por tenant_id (comportamento original)
		fmt.Printf("DEBUG CountByTenant Customer - Nenhum Branch ID encontrado, contando todos os clientes do tenant\n")
		query := fmt.Sprintf("SELECT COUNT(*) FROM %s.customers WHERE tenant_id = $1", schema)
		err = conn.QueryRow(ctx, query, tenantID).Scan(&count)
	}

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

// Exists verifica se um cliente existe pelo ID
func (r *CustomerRepository) Exists(ctx context.Context, id string) (bool, error) {
	conn, err := r.db.Acquire(ctx)
	if err != nil {
		return false, fmt.Errorf("falha ao obter conexão: %w", err)
	}
	defer conn.Release()

	// Obter tenant ID do contexto
	tenantID := pkgtenant.GetTenantID(ctx)
	if tenantID == "" {
		return false, errors.New("tenant ID não encontrado no contexto")
	}

	// Definir o search_path para public para acessar a tabela de tenants
	_, err = conn.Exec(ctx, "SET search_path TO public")
	if err != nil {
		return false, fmt.Errorf("falha ao configurar search_path: %w", err)
	}

	// Obter o schema do tenant
	var schema string
	err = conn.QueryRow(ctx, "SELECT schema FROM public.tenants WHERE id = $1", tenantID).Scan(&schema)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, errors.New("tenant não encontrado")
		}
		return false, fmt.Errorf("falha ao obter schema do tenant: %w", err)
	}

	// Debug para verificar o schema recuperado
	fmt.Printf("DEBUG Exists - Tenant ID: %s, Schema: %s, Customer ID: %s\n", tenantID, schema, id)

	// Verificar se o cliente existe no schema do tenant
	var exists bool
	query := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s.customers WHERE id = $1)", schema)
	err = conn.QueryRow(ctx, query, id).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("erro ao verificar existência do cliente por ID: %w", err)
	}

	return exists, nil
}

// ExistsByDocument verifica se já existe um cliente com o mesmo documento para o tenant
func (r *CustomerRepository) ExistsByDocument(ctx context.Context, tenantID, document string) (bool, error) {
	conn, err := r.db.Acquire(ctx)
	if err != nil {
		return false, fmt.Errorf("falha ao obter conexão: %w", err)
	}
	defer conn.Release()

	// Definir o search_path para public para acessar a tabela de tenants
	_, err = conn.Exec(ctx, "SET search_path TO public")
	if err != nil {
		return false, fmt.Errorf("falha ao configurar search_path: %w", err)
	}

	// Obter o schema do tenant
	var schema string
	err = conn.QueryRow(ctx, "SELECT schema FROM public.tenants WHERE id = $1", tenantID).Scan(&schema)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, errors.New("tenant não encontrado")
		}
		return false, fmt.Errorf("falha ao obter schema do tenant: %w", err)
	}

	// Debug para verificar o schema recuperado
	fmt.Printf("DEBUG ExistsByDocument - Tenant ID: %s, Schema: %s, Document: %s\n", tenantID, schema, document)

	// Verificar se o documento existe no schema do tenant
	var exists bool
	query := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s.customers WHERE tenant_id = $1 AND document = $2)", schema)
	err = conn.QueryRow(ctx, query, tenantID, document).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("erro ao verificar existência do cliente por documento: %w", err)
	}

	return exists, nil
}

// FindByName busca clientes pelo nome
func (r *CustomerRepository) FindByName(ctx context.Context, tenantID, name string, limit, offset int) ([]*customer.Customer, error) {
	conn, err := r.db.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("falha ao obter conexão: %w", err)
	}
	defer conn.Release()

	// Primeiro, definir o search_path para public para garantir que acessamos os tenants
	_, err = conn.Exec(ctx, "SET search_path TO public")
	if err != nil {
		return nil, fmt.Errorf("falha ao configurar search_path: %w", err)
	}

	// Obter o schema do tenant a partir do tenant_id
	var schema string
	err = conn.QueryRow(ctx, "SELECT schema FROM public.tenants WHERE id = $1", tenantID).Scan(&schema)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("tenant não encontrado")
		}
		return nil, fmt.Errorf("falha ao obter schema do tenant: %w", err)
	}

	// Debug para verificar o schema recuperado
	fmt.Printf("DEBUG FindByName Customer - Tenant ID: %s, Schema: %s\n", tenantID, schema)

	// Obter o branch ID do contexto
	branchID := pkgbranch.GetBranchID(ctx)

	var rows pgx.Rows

	// Construir a query usando o schema específico do tenant
	if branchID != "" {
		// Se tivermos um branch ID, filtrar por tenant_id, name e branch_id
		fmt.Printf("DEBUG FindByName Customer - Filtrando por Branch ID: %s\n", branchID)
		query := fmt.Sprintf(`SELECT 
			id, tenant_id, branch_id, person_type, name, trade_name, document,
			state_document, city_document, tax_regime, customer_type, status,
			credit_limit, payment_term, website, observations, fiscal_notes,
			addresses, contacts, last_purchase_at, created_at, updated_at,
			external_code, salesman_id, price_table_id, payment_method_id,
			suframa, reference_code
		FROM %s.customers 
		WHERE tenant_id = $1 AND branch_id = $2 AND name ILIKE $3
		ORDER BY name ASC
		LIMIT $4 OFFSET $5`, schema)

		rows, err = conn.Query(ctx, query, tenantID, branchID, "%"+name+"%", limit, offset)
	} else {
		// Se não tivermos branch ID, filtrar apenas por tenant_id e name (comportamento original)
		fmt.Printf("DEBUG FindByName Customer - Nenhum Branch ID encontrado, listando por nome sem filtro de filial\n")
		query := fmt.Sprintf(`SELECT 
			id, tenant_id, branch_id, person_type, name, trade_name, document,
			state_document, city_document, tax_regime, customer_type, status,
			credit_limit, payment_term, website, observations, fiscal_notes,
			addresses, contacts, last_purchase_at, created_at, updated_at,
			external_code, salesman_id, price_table_id, payment_method_id,
			suframa, reference_code
		FROM %s.customers 
		WHERE tenant_id = $1 AND name ILIKE $2
		ORDER BY name ASC
		LIMIT $3 OFFSET $4`, schema)

		rows, err = conn.Query(ctx, query, tenantID, "%"+name+"%", limit, offset)
	}

	if err != nil {
		return nil, fmt.Errorf("erro ao buscar clientes por nome: %w", err)
	}
	defer rows.Close()

	return r.scanCustomerRows(rows)
}

// FindByType implementa customer.Repository.FindByType
func (r *CustomerRepository) FindByType(ctx context.Context, tenantID string, customerType customer.CustomerType, limit, offset int) ([]*customer.Customer, error) {
	conn, err := r.db.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("falha ao obter conexão: %w", err)
	}
	defer conn.Release()

	// Primeiro, definir o search_path para public para garantir que acessamos os tenants
	_, err = conn.Exec(ctx, "SET search_path TO public")
	if err != nil {
		return nil, fmt.Errorf("falha ao configurar search_path: %w", err)
	}

	// Obter o schema do tenant a partir do tenant_id
	var schema string
	err = conn.QueryRow(ctx, "SELECT schema FROM public.tenants WHERE id = $1", tenantID).Scan(&schema)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("tenant não encontrado")
		}
		return nil, fmt.Errorf("falha ao obter schema do tenant: %w", err)
	}

	// Debug para verificar o schema recuperado
	fmt.Printf("DEBUG FindByType Customer - Tenant ID: %s, Schema: %s, Type: %s\n", tenantID, schema, customerType)

	// Obter o branch ID do contexto
	branchID := pkgbranch.GetBranchID(ctx)

	var rows pgx.Rows
	typeStr := mapCustomerType(string(customerType))

	// Construir a query usando o schema específico do tenant
	if branchID != "" {
		// Se tivermos um branch ID, filtrar por tenant_id, customer_type e branch_id
		fmt.Printf("DEBUG FindByType Customer - Filtrando por Branch ID: %s\n", branchID)
		query := fmt.Sprintf(`SELECT 
			id, tenant_id, branch_id, person_type, name, trade_name, document,
			state_document, city_document, tax_regime, customer_type, status,
			credit_limit, payment_term, website, observations, fiscal_notes,
			addresses, contacts, last_purchase_at, created_at, updated_at,
			external_code, salesman_id, price_table_id, payment_method_id,
			suframa, reference_code
		FROM %s.customers 
		WHERE tenant_id = $1 AND branch_id = $2 AND customer_type = $3
		ORDER BY name ASC
		LIMIT $4 OFFSET $5`, schema)

		rows, err = conn.Query(ctx, query, tenantID, branchID, typeStr, limit, offset)
	} else {
		// Se não tivermos branch ID, filtrar apenas por tenant_id e customer_type (comportamento original)
		fmt.Printf("DEBUG FindByType Customer - Nenhum Branch ID encontrado, listando por tipo sem filtro de filial\n")
		query := fmt.Sprintf(`SELECT 
			id, tenant_id, branch_id, person_type, name, trade_name, document,
			state_document, city_document, tax_regime, customer_type, status,
			credit_limit, payment_term, website, observations, fiscal_notes,
			addresses, contacts, last_purchase_at, created_at, updated_at,
			external_code, salesman_id, price_table_id, payment_method_id,
			suframa, reference_code
		FROM %s.customers 
		WHERE tenant_id = $1 AND customer_type = $2
		ORDER BY name ASC
		LIMIT $3 OFFSET $4`, schema)

		rows, err = conn.Query(ctx, query, tenantID, typeStr, limit, offset)
	}

	if err != nil {
		return nil, fmt.Errorf("erro ao buscar clientes por tipo: %w", err)
	}
	defer rows.Close()

	return r.scanCustomerRows(rows)
}

// FindBySalesman implementa customer.Repository.FindBySalesman
func (r *CustomerRepository) FindBySalesman(ctx context.Context, salesmanID string, limit, offset int) ([]*customer.Customer, error) {
	conn, err := r.db.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("falha ao obter conexão: %w", err)
	}
	defer conn.Release()

	// Obter tenant ID do contexto
	tenantID := pkgtenant.GetTenantID(ctx)
	if tenantID == "" {
		return nil, errors.New("tenant ID não encontrado no contexto")
	}

	// Primeiro, definir o search_path para public para garantir que acessamos os tenants
	_, err = conn.Exec(ctx, "SET search_path TO public")
	if err != nil {
		return nil, fmt.Errorf("falha ao configurar search_path: %w", err)
	}

	// Obter o schema do tenant a partir do tenant_id
	var schema string
	err = conn.QueryRow(ctx, "SELECT schema FROM public.tenants WHERE id = $1", tenantID).Scan(&schema)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("tenant não encontrado")
		}
		return nil, fmt.Errorf("falha ao obter schema do tenant: %w", err)
	}

	// Debug para verificar o schema recuperado
	fmt.Printf("DEBUG FindBySalesman Customer - Tenant ID: %s, Schema: %s, Salesman ID: %s\n", tenantID, schema, salesmanID)

	// Obter o branch ID do contexto
	branchID := pkgbranch.GetBranchID(ctx)

	var rows pgx.Rows

	// Construir a query usando o schema específico do tenant
	if branchID != "" {
		// Se tivermos um branch ID, filtrar por tenant_id, salesman_id e branch_id
		fmt.Printf("DEBUG FindBySalesman Customer - Filtrando por Branch ID: %s\n", branchID)
		query := fmt.Sprintf(`SELECT 
			id, tenant_id, branch_id, person_type, name, trade_name, document,
			state_document, city_document, tax_regime, customer_type, status,
			credit_limit, payment_term, website, observations, fiscal_notes,
			addresses, contacts, last_purchase_at, created_at, updated_at,
			external_code, salesman_id, price_table_id, payment_method_id,
			suframa, reference_code
		FROM %s.customers 
		WHERE tenant_id = $1 AND branch_id = $2 AND salesman_id = $3
		ORDER BY name ASC
		LIMIT $4 OFFSET $5`, schema)

		rows, err = conn.Query(ctx, query, tenantID, branchID, salesmanID, limit, offset)
	} else {
		// Se não tivermos branch ID, filtrar apenas por tenant_id e salesman_id
		fmt.Printf("DEBUG FindBySalesman Customer - Nenhum Branch ID encontrado, listando por vendedor sem filtro de filial\n")
		query := fmt.Sprintf(`SELECT 
			id, tenant_id, branch_id, person_type, name, trade_name, document,
			state_document, city_document, tax_regime, customer_type, status,
			credit_limit, payment_term, website, observations, fiscal_notes,
			addresses, contacts, last_purchase_at, created_at, updated_at,
			external_code, salesman_id, price_table_id, payment_method_id,
			suframa, reference_code
		FROM %s.customers 
		WHERE tenant_id = $1 AND salesman_id = $2
		ORDER BY name ASC
		LIMIT $3 OFFSET $4`, schema)

		rows, err = conn.Query(ctx, query, tenantID, salesmanID, limit, offset)
	}

	if err != nil {
		return nil, fmt.Errorf("erro ao buscar clientes por vendedor: %w", err)
	}
	defer rows.Close()

	return r.scanCustomerRows(rows)
}

// FindByPriceTable implementa customer.Repository.FindByPriceTable
func (r *CustomerRepository) FindByPriceTable(ctx context.Context, priceTableID string, limit, offset int) ([]*customer.Customer, error) {
	conn, err := r.db.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("falha ao obter conexão: %w", err)
	}
	defer conn.Release()

	// Obter tenant ID do contexto
	tenantID := pkgtenant.GetTenantID(ctx)
	if tenantID == "" {
		return nil, errors.New("tenant ID não encontrado no contexto")
	}

	// Primeiro, definir o search_path para public para garantir que acessamos os tenants
	_, err = conn.Exec(ctx, "SET search_path TO public")
	if err != nil {
		return nil, fmt.Errorf("falha ao configurar search_path: %w", err)
	}

	// Obter o schema do tenant a partir do tenant_id
	var schema string
	err = conn.QueryRow(ctx, "SELECT schema FROM public.tenants WHERE id = $1", tenantID).Scan(&schema)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("tenant não encontrado")
		}
		return nil, fmt.Errorf("falha ao obter schema do tenant: %w", err)
	}

	// Debug para verificar o schema recuperado
	fmt.Printf("DEBUG FindByPriceTable Customer - Tenant ID: %s, Schema: %s, PriceTable ID: %s\n", tenantID, schema, priceTableID)

	// Obter o branch ID do contexto
	branchID := pkgbranch.GetBranchID(ctx)

	var rows pgx.Rows

	// Construir a query usando o schema específico do tenant
	if branchID != "" {
		// Se tivermos um branch ID, filtrar por tenant_id, price_table_id e branch_id
		fmt.Printf("DEBUG FindByPriceTable Customer - Filtrando por Branch ID: %s\n", branchID)
		query := fmt.Sprintf(`SELECT 
			id, tenant_id, branch_id, person_type, name, trade_name, document,
			state_document, city_document, tax_regime, customer_type, status,
			credit_limit, payment_term, website, observations, fiscal_notes,
			addresses, contacts, last_purchase_at, created_at, updated_at,
			external_code, salesman_id, price_table_id, payment_method_id,
			suframa, reference_code
		FROM %s.customers 
		WHERE tenant_id = $1 AND branch_id = $2 AND price_table_id = $3
		ORDER BY name ASC
		LIMIT $4 OFFSET $5`, schema)

		rows, err = conn.Query(ctx, query, tenantID, branchID, priceTableID, limit, offset)
	} else {
		// Se não tivermos branch ID, filtrar apenas por tenant_id e price_table_id
		fmt.Printf("DEBUG FindByPriceTable Customer - Nenhum Branch ID encontrado, listando por tabela de preço sem filtro de filial\n")
		query := fmt.Sprintf(`SELECT 
			id, tenant_id, branch_id, person_type, name, trade_name, document,
			state_document, city_document, tax_regime, customer_type, status,
			credit_limit, payment_term, website, observations, fiscal_notes,
			addresses, contacts, last_purchase_at, created_at, updated_at,
			external_code, salesman_id, price_table_id, payment_method_id,
			suframa, reference_code
		FROM %s.customers 
		WHERE tenant_id = $1 AND price_table_id = $2
		ORDER BY name ASC
		LIMIT $3 OFFSET $4`, schema)

		rows, err = conn.Query(ctx, query, tenantID, priceTableID, limit, offset)
	}

	if err != nil {
		return nil, fmt.Errorf("erro ao buscar clientes por tabela de preço: %w", err)
	}
	defer rows.Close()

	return r.scanCustomerRows(rows)
}

// FindByPaymentMethod implementa customer.Repository.FindByPaymentMethod
func (r *CustomerRepository) FindByPaymentMethod(ctx context.Context, paymentMethodID string, limit, offset int) ([]*customer.Customer, error) {
	conn, err := r.db.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("falha ao obter conexão: %w", err)
	}
	defer conn.Release()

	// Obter tenant ID do contexto
	tenantID := pkgtenant.GetTenantID(ctx)
	if tenantID == "" {
		return nil, errors.New("tenant ID não encontrado no contexto")
	}

	// Primeiro, definir o search_path para public para garantir que acessamos os tenants
	_, err = conn.Exec(ctx, "SET search_path TO public")
	if err != nil {
		return nil, fmt.Errorf("falha ao configurar search_path: %w", err)
	}

	// Obter o schema do tenant a partir do tenant_id
	var schema string
	err = conn.QueryRow(ctx, "SELECT schema FROM public.tenants WHERE id = $1", tenantID).Scan(&schema)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("tenant não encontrado")
		}
		return nil, fmt.Errorf("falha ao obter schema do tenant: %w", err)
	}

	// Debug para verificar o schema recuperado
	fmt.Printf("DEBUG FindByPaymentMethod Customer - Tenant ID: %s, Schema: %s, PaymentMethod ID: %s\n", tenantID, schema, paymentMethodID)

	// Obter o branch ID do contexto
	branchID := pkgbranch.GetBranchID(ctx)

	var rows pgx.Rows

	// Construir a query usando o schema específico do tenant
	if branchID != "" {
		// Se tivermos um branch ID, filtrar por tenant_id, payment_method_id e branch_id
		fmt.Printf("DEBUG FindByPaymentMethod Customer - Filtrando por Branch ID: %s\n", branchID)
		query := fmt.Sprintf(`SELECT 
			id, tenant_id, branch_id, person_type, name, trade_name, document,
			state_document, city_document, tax_regime, customer_type, status,
			credit_limit, payment_term, website, observations, fiscal_notes,
			addresses, contacts, last_purchase_at, created_at, updated_at,
			external_code, salesman_id, price_table_id, payment_method_id,
			suframa, reference_code
		FROM %s.customers 
		WHERE tenant_id = $1 AND branch_id = $2 AND payment_method_id = $3
		ORDER BY name ASC
		LIMIT $4 OFFSET $5`, schema)

		rows, err = conn.Query(ctx, query, tenantID, branchID, paymentMethodID, limit, offset)
	} else {
		// Se não tivermos branch ID, filtrar apenas por tenant_id e payment_method_id
		fmt.Printf("DEBUG FindByPaymentMethod Customer - Nenhum Branch ID encontrado, listando por método de pagamento sem filtro de filial\n")
		query := fmt.Sprintf(`SELECT 
			id, tenant_id, branch_id, person_type, name, trade_name, document,
			state_document, city_document, tax_regime, customer_type, status,
			credit_limit, payment_term, website, observations, fiscal_notes,
			addresses, contacts, last_purchase_at, created_at, updated_at,
			external_code, salesman_id, price_table_id, payment_method_id,
			suframa, reference_code
		FROM %s.customers 
		WHERE tenant_id = $1 AND payment_method_id = $2
		ORDER BY name ASC
		LIMIT $3 OFFSET $4`, schema)

		rows, err = conn.Query(ctx, query, tenantID, paymentMethodID, limit, offset)
	}

	if err != nil {
		return nil, fmt.Errorf("erro ao buscar clientes por método de pagamento: %w", err)
	}
	defer rows.Close()

	return r.scanCustomerRows(rows)
}

// FindByStatus implementa customer.Repository.FindByStatus
func (r *CustomerRepository) FindByStatus(ctx context.Context, tenantID string, status customer.Status, limit, offset int) ([]*customer.Customer, error) {
	conn, err := r.db.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("falha ao obter conexão: %w", err)
	}
	defer conn.Release()

	// Primeiro, definir o search_path para public para garantir que acessamos os tenants
	_, err = conn.Exec(ctx, "SET search_path TO public")
	if err != nil {
		return nil, fmt.Errorf("falha ao configurar search_path: %w", err)
	}

	// Obter o schema do tenant a partir do tenant_id
	var schema string
	err = conn.QueryRow(ctx, "SELECT schema FROM public.tenants WHERE id = $1", tenantID).Scan(&schema)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("tenant não encontrado")
		}
		return nil, fmt.Errorf("falha ao obter schema do tenant: %w", err)
	}

	// Debug para verificar o schema recuperado
	fmt.Printf("DEBUG FindByStatus Customer - Tenant ID: %s, Schema: %s, Status: %s\n", tenantID, schema, status)

	// Obter o branch ID do contexto
	branchID := pkgbranch.GetBranchID(ctx)

	var rows pgx.Rows
	statusStr := mapCustomerStatus(string(status))

	// Construir a query usando o schema específico do tenant
	if branchID != "" {
		// Se tivermos um branch ID, filtrar por tenant_id, status e branch_id
		fmt.Printf("DEBUG FindByStatus Customer - Filtrando por Branch ID: %s\n", branchID)
		query := fmt.Sprintf(`SELECT 
			id, tenant_id, branch_id, person_type, name, trade_name, document,
			state_document, city_document, tax_regime, customer_type, status,
			credit_limit, payment_term, website, observations, fiscal_notes,
			addresses, contacts, last_purchase_at, created_at, updated_at,
			external_code, salesman_id, price_table_id, payment_method_id,
			suframa, reference_code
		FROM %s.customers 
		WHERE tenant_id = $1 AND branch_id = $2 AND status = $3
		ORDER BY name ASC
		LIMIT $4 OFFSET $5`, schema)

		rows, err = conn.Query(ctx, query, tenantID, branchID, statusStr, limit, offset)
	} else {
		// Se não tivermos branch ID, filtrar apenas por tenant_id e status (comportamento original)
		fmt.Printf("DEBUG FindByStatus Customer - Nenhum Branch ID encontrado, listando por status sem filtro de filial\n")
		query := fmt.Sprintf(`SELECT 
			id, tenant_id, branch_id, person_type, name, trade_name, document,
			state_document, city_document, tax_regime, customer_type, status,
			credit_limit, payment_term, website, observations, fiscal_notes,
			addresses, contacts, last_purchase_at, created_at, updated_at,
			external_code, salesman_id, price_table_id, payment_method_id,
			suframa, reference_code
		FROM %s.customers 
		WHERE tenant_id = $1 AND status = $2
		ORDER BY name ASC
		LIMIT $3 OFFSET $4`, schema)

		rows, err = conn.Query(ctx, query, tenantID, statusStr, limit, offset)
	}

	if err != nil {
		return nil, fmt.Errorf("erro ao buscar clientes por status: %w", err)
	}
	defer rows.Close()

	return r.scanCustomerRows(rows)
}

// FindByTaxRegime implementa customer.Repository.FindByTaxRegime
func (r *CustomerRepository) FindByTaxRegime(ctx context.Context, tenantID string, taxRegime customer.TaxRegime, limit, offset int) ([]*customer.Customer, error) {
	conn, err := r.db.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("falha ao obter conexão: %w", err)
	}
	defer conn.Release()

	// Primeiro, definir o search_path para public para garantir que acessamos os tenants
	_, err = conn.Exec(ctx, "SET search_path TO public")
	if err != nil {
		return nil, fmt.Errorf("falha ao configurar search_path: %w", err)
	}

	// Obter o schema do tenant a partir do tenant_id
	var schema string
	err = conn.QueryRow(ctx, "SELECT schema FROM public.tenants WHERE id = $1", tenantID).Scan(&schema)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("tenant não encontrado")
		}
		return nil, fmt.Errorf("falha ao obter schema do tenant: %w", err)
	}

	// Debug para verificar o schema recuperado
	fmt.Printf("DEBUG FindByTaxRegime Customer - Tenant ID: %s, Schema: %s, TaxRegime: %s\n", tenantID, schema, taxRegime)

	// Obter o branch ID do contexto
	branchID := pkgbranch.GetBranchID(ctx)

	var rows pgx.Rows
	taxRegimeStr := mapTaxRegime(string(taxRegime))

	// Construir a query usando o schema específico do tenant
	if branchID != "" {
		// Se tivermos um branch ID, filtrar por tenant_id, tax_regime e branch_id
		fmt.Printf("DEBUG FindByTaxRegime Customer - Filtrando por Branch ID: %s\n", branchID)
		query := fmt.Sprintf(`SELECT 
			id, tenant_id, branch_id, person_type, name, trade_name, document,
			state_document, city_document, tax_regime, customer_type, status,
			credit_limit, payment_term, website, observations, fiscal_notes,
			addresses, contacts, last_purchase_at, created_at, updated_at,
			external_code, salesman_id, price_table_id, payment_method_id,
			suframa, reference_code
		FROM %s.customers 
		WHERE tenant_id = $1 AND branch_id = $2 AND tax_regime = $3
		ORDER BY name ASC
		LIMIT $4 OFFSET $5`, schema)

		rows, err = conn.Query(ctx, query, tenantID, branchID, taxRegimeStr, limit, offset)
	} else {
		// Se não tivermos branch ID, filtrar apenas por tenant_id e tax_regime (comportamento original)
		fmt.Printf("DEBUG FindByTaxRegime Customer - Nenhum Branch ID encontrado, listando por regime tributário sem filtro de filial\n")
		query := fmt.Sprintf(`SELECT 
			id, tenant_id, branch_id, person_type, name, trade_name, document,
			state_document, city_document, tax_regime, customer_type, status,
			credit_limit, payment_term, website, observations, fiscal_notes,
			addresses, contacts, last_purchase_at, created_at, updated_at,
			external_code, salesman_id, price_table_id, payment_method_id,
			suframa, reference_code
		FROM %s.customers 
		WHERE tenant_id = $1 AND tax_regime = $2
		ORDER BY name ASC
		LIMIT $3 OFFSET $4`, schema)

		rows, err = conn.Query(ctx, query, tenantID, taxRegimeStr, limit, offset)
	}

	if err != nil {
		return nil, fmt.Errorf("erro ao buscar clientes por regime tributário: %w", err)
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
	var customers []*customer.Customer

	for rows.Next() {
		var c customer.Customer
		var addressesJSON, contactsJSON []byte

		// Usar variáveis para valores que podem ser nulos
		var branchID, salesmanID, priceTableID, paymentMethodID, externalCode, suframa, referenceCode sql.NullString
		var lastPurchaseAt sql.NullTime

		err := rows.Scan(
			&c.ID, &c.TenantID, &branchID, &c.PersonType, &c.Name, &c.TradeName,
			&c.Document, &c.StateDocument, &c.CityDocument, &c.TaxRegime,
			&c.CustomerType, &c.Status, &c.CreditLimit, &c.PaymentTerm,
			&c.Website, &c.Observations, &c.FiscalNotes, &addressesJSON,
			&contactsJSON, &lastPurchaseAt, &c.CreatedAt, &c.UpdatedAt,
			&externalCode, &salesmanID, &priceTableID, &paymentMethodID,
			&suframa, &referenceCode)

		if err != nil {
			return nil, fmt.Errorf("erro ao ler cliente: %w", err)
		}

		// Atribuir valores nulos aos campos da estrutura apenas se forem válidos
		if branchID.Valid {
			c.BranchID = branchID.String
		}
		if salesmanID.Valid {
			c.SalesmanID = salesmanID.String
		}
		if priceTableID.Valid {
			c.PriceTableID = priceTableID.String
		}
		if paymentMethodID.Valid {
			c.PaymentMethodID = paymentMethodID.String
		}
		if externalCode.Valid {
			c.ExternalCode = externalCode.String
		}
		if suframa.Valid {
			c.SUFRAMA = suframa.String
		}
		if referenceCode.Valid {
			c.ReferenceCode = referenceCode.String
		}
		if lastPurchaseAt.Valid {
			c.LastPurchaseAt = &lastPurchaseAt.Time
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
		return nil, fmt.Errorf("erro ao ler linhas: %w", err)
	}

	return customers, nil
}
