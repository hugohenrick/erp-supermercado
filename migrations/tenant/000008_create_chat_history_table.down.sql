-- Remover índices
DROP INDEX IF EXISTS idx_chat_history_created_at;
DROP INDEX IF EXISTS idx_chat_history_user_id;
DROP INDEX IF EXISTS idx_chat_history_tenant_id;

-- Remover tabela
DROP TABLE IF EXISTS chat_history; 