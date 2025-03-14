CREATE TYPE person_type AS ENUM ('PF', 'PJ');
CREATE TYPE customer_type AS ENUM ('CUSTOMER', 'SUPPLIER', 'CARRIER');
CREATE TYPE customer_status AS ENUM ('ACTIVE', 'INACTIVE', 'BLOCKED');
CREATE TYPE tax_regime AS ENUM ('SIMPLE', 'PRESUMED', 'REAL');

CREATE TABLE IF NOT EXISTS customers (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    branch_id UUID NOT NULL REFERENCES branches(id),
    person_type person_type NOT NULL,
    name VARCHAR(100) NOT NULL,
    trade_name VARCHAR(100),
    document VARCHAR(20) NOT NULL,
    state_document VARCHAR(20),
    city_document VARCHAR(20),
    tax_regime tax_regime NOT NULL,
    customer_type customer_type NOT NULL,
    status customer_status NOT NULL DEFAULT 'ACTIVE',
    credit_limit DECIMAL(15,2) NOT NULL DEFAULT 0,
    payment_term INTEGER NOT NULL DEFAULT 0,
    website VARCHAR(100),
    observations TEXT,
    fiscal_notes TEXT,
    addresses JSONB NOT NULL DEFAULT '[]',
    contacts JSONB NOT NULL DEFAULT '[]',
    last_purchase_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    external_code VARCHAR(50),
    salesman_id UUID REFERENCES users(id),
    price_table_id UUID,
    payment_method_id UUID,
    suframa VARCHAR(20),
    reference_code VARCHAR(50),
    UNIQUE(tenant_id, document)
);

CREATE INDEX idx_customers_tenant_id ON customers(tenant_id);
CREATE INDEX idx_customers_branch_id ON customers(branch_id);
CREATE INDEX idx_customers_document ON customers(document);
CREATE INDEX idx_customers_name ON customers(name);
CREATE INDEX idx_customers_customer_type ON customers(customer_type);
CREATE INDEX idx_customers_status ON customers(status);
CREATE INDEX idx_customers_salesman_id ON customers(salesman_id);
CREATE INDEX idx_customers_price_table_id ON customers(price_table_id);
CREATE INDEX idx_customers_payment_method_id ON customers(payment_method_id); 