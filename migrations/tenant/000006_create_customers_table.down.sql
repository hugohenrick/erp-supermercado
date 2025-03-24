-- Remover índices
DROP INDEX IF EXISTS idx_customers_payment_method_id;
DROP INDEX IF EXISTS idx_customers_price_table_id;
DROP INDEX IF EXISTS idx_customers_salesman_id;
DROP INDEX IF EXISTS idx_customers_status;
DROP INDEX IF EXISTS idx_customers_customer_type;
DROP INDEX IF EXISTS idx_customers_name;
DROP INDEX IF EXISTS idx_customers_document;
DROP INDEX IF EXISTS idx_customers_branch_id;
DROP INDEX IF EXISTS idx_customers_tenant_id;

-- Remover tabela
DROP TABLE IF EXISTS customers;

-- Remover tipos enumerados
DROP TYPE IF EXISTS tax_regime;
DROP TYPE IF EXISTS customer_status;
DROP TYPE IF EXISTS customer_type;
DROP TYPE IF EXISTS person_type; 