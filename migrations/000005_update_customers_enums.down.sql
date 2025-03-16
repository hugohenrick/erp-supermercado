-- Voltar os tipos para os valores antigos se a tabela existir
DO $$
BEGIN
    IF EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = 'customers') THEN
        -- Converter as colunas para VARCHAR temporariamente
        ALTER TABLE customers
            ALTER COLUMN person_type TYPE VARCHAR(10),
            ALTER COLUMN customer_type TYPE VARCHAR(20),
            ALTER COLUMN status TYPE VARCHAR(20),
            ALTER COLUMN tax_regime TYPE VARCHAR(20);
            
        -- Voltar para os valores antigos
        UPDATE customers SET 
            customer_type = 
                CASE customer_type 
                    WHEN 'final' THEN 'CUSTOMER'
                    WHEN 'reseller' THEN 'SUPPLIER'
                    WHEN 'wholesale' THEN 'CARRIER'
                    ELSE 'CUSTOMER'
                END,
            status = 
                CASE status 
                    WHEN 'active' THEN 'ACTIVE'
                    WHEN 'inactive' THEN 'INACTIVE'
                    WHEN 'blocked' THEN 'BLOCKED'
                    ELSE 'ACTIVE'
                END,
            tax_regime = 
                CASE tax_regime 
                    WHEN 'simples' THEN 'SIMPLE'
                    WHEN 'mei' THEN 'SIMPLE'
                    WHEN 'presumido' THEN 'PRESUMED'
                    WHEN 'real' THEN 'REAL'
                    ELSE 'SIMPLE'
                END;
    END IF;
END
$$;

-- Recriar os tipos originais
DROP TYPE IF EXISTS customer_type CASCADE;
CREATE TYPE customer_type AS ENUM ('CUSTOMER', 'SUPPLIER', 'CARRIER');

DROP TYPE IF EXISTS customer_status CASCADE;
CREATE TYPE customer_status AS ENUM ('ACTIVE', 'INACTIVE', 'BLOCKED');

DROP TYPE IF EXISTS tax_regime CASCADE;
CREATE TYPE tax_regime AS ENUM ('SIMPLE', 'PRESUMED', 'REAL');

-- Converter novamente para os tipos ENUM
DO $$
BEGIN
    IF EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = 'customers') THEN
        ALTER TABLE customers
            ALTER COLUMN person_type TYPE person_type USING person_type::person_type,
            ALTER COLUMN customer_type TYPE customer_type USING customer_type::customer_type,
            ALTER COLUMN status TYPE customer_status USING status::customer_status,
            ALTER COLUMN tax_regime TYPE tax_regime USING tax_regime::tax_regime;
    END IF;
END
$$; 