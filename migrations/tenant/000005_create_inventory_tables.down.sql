-- Remover índices da tabela de movimentações
DROP INDEX IF EXISTS idx_inventory_movements_created_at;
DROP INDEX IF EXISTS idx_inventory_movements_reference_id;
DROP INDEX IF EXISTS idx_inventory_movements_type;
DROP INDEX IF EXISTS idx_inventory_movements_product_id;
DROP INDEX IF EXISTS idx_inventory_movements_branch_id;
DROP INDEX IF EXISTS idx_inventory_movements_tenant_id;

-- Remover a tabela de movimentações
DROP TABLE IF EXISTS inventory_movements;

-- Remover índices da tabela de inventário
DROP INDEX IF EXISTS idx_inventory_product_id;
DROP INDEX IF EXISTS idx_inventory_branch_id;
DROP INDEX IF EXISTS idx_inventory_tenant_id;

-- Remover a tabela de inventário
DROP TABLE IF EXISTS inventory; 