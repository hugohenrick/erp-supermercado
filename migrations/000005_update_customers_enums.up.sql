-- Vamos primeiro checar se os enum types existem e mantê-los intactos se sim
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'person_type') THEN
        -- Person type já existe, mantê-lo intacto
    ELSE
        -- Criar person_type se não existir
        CREATE TYPE person_type AS ENUM ('PF', 'PJ');
    END IF;
END
$$;

-- Criar um novo tipo para customer_type que corresponda ao front-end
DROP TYPE IF EXISTS customer_type CASCADE;
CREATE TYPE customer_type AS ENUM ('final', 'reseller', 'wholesale');

-- Criar um novo tipo para customer_status que corresponda ao front-end
DROP TYPE IF EXISTS customer_status CASCADE;
CREATE TYPE customer_status AS ENUM ('active', 'inactive', 'blocked');

-- Criar um novo tipo para tax_regime que corresponda ao front-end
DROP TYPE IF EXISTS tax_regime CASCADE;
CREATE TYPE tax_regime AS ENUM ('simples', 'mei', 'presumido', 'real');

-- Alterações necessárias para a tabela customers se ela já existir
-- Isso garante que não haverá problemas se a migração for executada em um novo banco
-- mas também lida com bancos já existentes
DO $$
BEGIN
    -- Verificar se a tabela customers existe
    IF EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = 'customers') THEN
        -- Recria a tabela customers com a estrutura atualizada e preserva os dados
        ALTER TABLE customers
            ALTER COLUMN person_type TYPE VARCHAR(10),
            ALTER COLUMN customer_type TYPE VARCHAR(20),
            ALTER COLUMN status TYPE VARCHAR(20),
            ALTER COLUMN tax_regime TYPE VARCHAR(20);
            
        -- Atualiza os valores existentes para mapear para os novos tipos
        UPDATE customers SET 
            customer_type = 
                CASE customer_type 
                    WHEN 'CUSTOMER' THEN 'final'
                    WHEN 'SUPPLIER' THEN 'reseller'
                    WHEN 'CARRIER' THEN 'wholesale'
                    ELSE 'final'
                END,
            status = 
                CASE status 
                    WHEN 'ACTIVE' THEN 'active'
                    WHEN 'INACTIVE' THEN 'inactive'
                    WHEN 'BLOCKED' THEN 'blocked'
                    ELSE 'active'
                END,
            tax_regime = 
                CASE tax_regime 
                    WHEN 'SIMPLE' THEN 'simples'
                    WHEN 'PRESUMED' THEN 'presumido'
                    WHEN 'REAL' THEN 'real'
                    ELSE 'simples'
                END;
                
        -- Converter colunas para os novos tipos ENUM
        ALTER TABLE customers
            ALTER COLUMN person_type TYPE person_type USING person_type::person_type,
            ALTER COLUMN customer_type TYPE customer_type USING customer_type::customer_type,
            ALTER COLUMN status TYPE customer_status USING status::customer_status,
            ALTER COLUMN tax_regime TYPE tax_regime USING tax_regime::tax_regime;
    END IF;
END
$$; 