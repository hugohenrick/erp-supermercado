-- Adicionar colunas faltantes à tabela de clientes existente
DO $$
BEGIN
    -- Verificar se a coluna person_type existe
    IF NOT EXISTS (
        SELECT 1 
        FROM information_schema.columns 
        WHERE table_name = 'customers' AND column_name = 'person_type'
    ) THEN
        -- Criar tipo person_type se não existir
        IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'person_type') THEN
            CREATE TYPE person_type AS ENUM ('PF', 'PJ');
        END IF;
        
        -- Adicionar coluna person_type
        ALTER TABLE customers ADD COLUMN person_type person_type NOT NULL DEFAULT 'PF';
    END IF;

    -- Verificar se a coluna trade_name existe
    IF NOT EXISTS (
        SELECT 1 
        FROM information_schema.columns 
        WHERE table_name = 'customers' AND column_name = 'trade_name'
    ) THEN
        -- Adicionar coluna trade_name
        ALTER TABLE customers ADD COLUMN trade_name VARCHAR(255);
    END IF;

    -- Verificar se a coluna state_document existe
    IF NOT EXISTS (
        SELECT 1 
        FROM information_schema.columns 
        WHERE table_name = 'customers' AND column_name = 'state_document'
    ) THEN
        -- Adicionar coluna state_document
        ALTER TABLE customers ADD COLUMN state_document VARCHAR(50);
    END IF;
    
    -- Verificar se a coluna city_document existe
    IF NOT EXISTS (
        SELECT 1 
        FROM information_schema.columns 
        WHERE table_name = 'customers' AND column_name = 'city_document'
    ) THEN
        -- Adicionar coluna city_document
        ALTER TABLE customers ADD COLUMN city_document VARCHAR(50);
    END IF;
    
    -- Verificar se a coluna customer_type existe
    IF NOT EXISTS (
        SELECT 1 
        FROM information_schema.columns 
        WHERE table_name = 'customers' AND column_name = 'customer_type'
    ) THEN
        -- Criar tipo customer_type se não existir
        IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'customer_type') THEN
            CREATE TYPE customer_type AS ENUM ('final', 'reseller', 'wholesale');
        END IF;
        
        -- Adicionar coluna customer_type
        ALTER TABLE customers ADD COLUMN customer_type customer_type NOT NULL DEFAULT 'final';
    END IF;
    
    -- Verificar se a coluna status existe e renomear se necessário
    IF EXISTS (
        SELECT 1 
        FROM information_schema.columns 
        WHERE table_name = 'customers' AND column_name = 'active' AND NOT EXISTS (
            SELECT 1 
            FROM information_schema.columns 
            WHERE table_name = 'customers' AND column_name = 'status'
        )
    ) THEN
        -- Criar tipo customer_status se não existir
        IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'customer_status') THEN
            CREATE TYPE customer_status AS ENUM ('active', 'inactive', 'blocked');
        END IF;
        
        -- Adicionar coluna status temporária
        ALTER TABLE customers ADD COLUMN status customer_status;
        
        -- Atualizar valores baseados na coluna active
        UPDATE customers SET status = CASE WHEN active THEN 'active'::customer_status ELSE 'inactive'::customer_status END;
        
        -- Definir NOT NULL na coluna status
        ALTER TABLE customers ALTER COLUMN status SET NOT NULL;
    END IF;
    
    -- Verificar se a coluna tax_regime existe
    IF NOT EXISTS (
        SELECT 1 
        FROM information_schema.columns 
        WHERE table_name = 'customers' AND column_name = 'tax_regime'
    ) THEN
        -- Criar tipo tax_regime se não existir
        IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'tax_regime') THEN
            CREATE TYPE tax_regime AS ENUM ('simples', 'mei', 'presumido', 'real');
        END IF;
        
        -- Adicionar coluna tax_regime
        ALTER TABLE customers ADD COLUMN tax_regime tax_regime NOT NULL DEFAULT 'simples';
    END IF;
    
    -- Verificar se a coluna credit_limit existe
    IF NOT EXISTS (
        SELECT 1 
        FROM information_schema.columns 
        WHERE table_name = 'customers' AND column_name = 'credit_limit'
    ) THEN
        -- Adicionar coluna credit_limit
        ALTER TABLE customers ADD COLUMN credit_limit DECIMAL(15,2) NOT NULL DEFAULT 0;
    END IF;
    
    -- Verificar se a coluna payment_term existe
    IF NOT EXISTS (
        SELECT 1 
        FROM information_schema.columns 
        WHERE table_name = 'customers' AND column_name = 'payment_term'
    ) THEN
        -- Adicionar coluna payment_term
        ALTER TABLE customers ADD COLUMN payment_term INTEGER NOT NULL DEFAULT 0;
    END IF;
    
    -- Verificar se a coluna website existe
    IF NOT EXISTS (
        SELECT 1 
        FROM information_schema.columns 
        WHERE table_name = 'customers' AND column_name = 'website'
    ) THEN
        -- Adicionar coluna website
        ALTER TABLE customers ADD COLUMN website VARCHAR(255);
    END IF;
    
    -- Renomear coluna notes para observations, se existir
    IF EXISTS (
        SELECT 1 
        FROM information_schema.columns 
        WHERE table_name = 'customers' AND column_name = 'notes' AND NOT EXISTS (
            SELECT 1 
            FROM information_schema.columns 
            WHERE table_name = 'customers' AND column_name = 'observations'
        )
    ) THEN
        -- Renomear coluna notes para observations
        ALTER TABLE customers RENAME COLUMN notes TO observations;
    END IF;
    
    -- Verificar se a coluna fiscal_notes existe
    IF NOT EXISTS (
        SELECT 1 
        FROM information_schema.columns 
        WHERE table_name = 'customers' AND column_name = 'fiscal_notes'
    ) THEN
        -- Adicionar coluna fiscal_notes
        ALTER TABLE customers ADD COLUMN fiscal_notes TEXT;
    END IF;
    
    -- Verificar se a coluna last_purchase_at existe
    IF NOT EXISTS (
        SELECT 1 
        FROM information_schema.columns 
        WHERE table_name = 'customers' AND column_name = 'last_purchase_at'
    ) THEN
        -- Adicionar coluna last_purchase_at
        ALTER TABLE customers ADD COLUMN last_purchase_at TIMESTAMP;
    END IF;
    
    -- Verificar se a coluna external_code existe
    IF NOT EXISTS (
        SELECT 1 
        FROM information_schema.columns 
        WHERE table_name = 'customers' AND column_name = 'external_code'
    ) THEN
        -- Adicionar coluna external_code
        ALTER TABLE customers ADD COLUMN external_code VARCHAR(50);
    END IF;
    
    -- Verificar se a coluna salesman_id existe
    IF NOT EXISTS (
        SELECT 1 
        FROM information_schema.columns 
        WHERE table_name = 'customers' AND column_name = 'salesman_id'
    ) THEN
        -- Adicionar coluna salesman_id
        ALTER TABLE customers ADD COLUMN salesman_id UUID REFERENCES users(id);
    END IF;
    
    -- Verificar se a coluna price_table_id existe
    IF NOT EXISTS (
        SELECT 1 
        FROM information_schema.columns 
        WHERE table_name = 'customers' AND column_name = 'price_table_id'
    ) THEN
        -- Adicionar coluna price_table_id
        ALTER TABLE customers ADD COLUMN price_table_id UUID;
    END IF;
    
    -- Verificar se a coluna payment_method_id existe
    IF NOT EXISTS (
        SELECT 1 
        FROM information_schema.columns 
        WHERE table_name = 'customers' AND column_name = 'payment_method_id'
    ) THEN
        -- Adicionar coluna payment_method_id
        ALTER TABLE customers ADD COLUMN payment_method_id UUID;
    END IF;
    
    -- Verificar se a coluna suframa existe
    IF NOT EXISTS (
        SELECT 1 
        FROM information_schema.columns 
        WHERE table_name = 'customers' AND column_name = 'suframa'
    ) THEN
        -- Adicionar coluna suframa
        ALTER TABLE customers ADD COLUMN suframa VARCHAR(20);
    END IF;
    
    -- Verificar se a coluna reference_code existe
    IF NOT EXISTS (
        SELECT 1 
        FROM information_schema.columns 
        WHERE table_name = 'customers' AND column_name = 'reference_code'
    ) THEN
        -- Adicionar coluna reference_code
        ALTER TABLE customers ADD COLUMN reference_code VARCHAR(50);
    END IF;
    
    -- Verificar se a coluna addresses existe
    IF NOT EXISTS (
        SELECT 1 
        FROM information_schema.columns 
        WHERE table_name = 'customers' AND column_name = 'addresses'
    ) THEN
        -- Adicionar coluna addresses (JSONB array)
        ALTER TABLE customers ADD COLUMN addresses JSONB NOT NULL DEFAULT '[]';
        
        -- Converter endereços antigos para o formato JSONB, se street, city etc estiverem presentes
        DO $addresses$
        BEGIN
            IF EXISTS (
                SELECT 1 
                FROM information_schema.columns 
                WHERE table_name = 'customers' AND column_name IN ('street', 'city')
            ) THEN
                UPDATE customers 
                SET addresses = (
                    CASE WHEN street IS NOT NULL OR city IS NOT NULL THEN
                        json_build_array(
                            json_build_object(
                                'street', COALESCE(street, ''),
                                'number', COALESCE(number, ''),
                                'complement', COALESCE(complement, ''),
                                'district', COALESCE(district, ''),
                                'city', COALESCE(city, ''),
                                'state', COALESCE(state, ''),
                                'zip_code', COALESCE(zip_code, ''),
                                'country', COALESCE(country, 'Brasil'),
                                'address_type', 'commercial',
                                'main_address', true,
                                'delivery_address', true
                            )
                        )
                    ELSE
                        '[]'
                    END
                )::JSONB;
            END IF;
        END $addresses$;
    END IF;
    
    -- Verificar se a coluna contacts existe
    IF NOT EXISTS (
        SELECT 1 
        FROM information_schema.columns 
        WHERE table_name = 'customers' AND column_name = 'contacts'
    ) THEN
        -- Adicionar coluna contacts (JSONB array)
        ALTER TABLE customers ADD COLUMN contacts JSONB NOT NULL DEFAULT '[]';
        
        -- Converter contatos antigos para o formato JSONB, se email, phone estiverem presentes
        DO $contacts$
        BEGIN
            IF EXISTS (
                SELECT 1 
                FROM information_schema.columns 
                WHERE table_name = 'customers' AND column_name IN ('email', 'phone')
            ) THEN
                UPDATE customers 
                SET contacts = (
                    CASE WHEN email IS NOT NULL OR phone IS NOT NULL THEN
                        json_build_array(
                            json_build_object(
                                'name', name,
                                'email', COALESCE(email, ''),
                                'phone', COALESCE(phone, ''),
                                'mobile_phone', '',
                                'department', '',
                                'position', '',
                                'main_contact', true
                            )
                        )
                    ELSE
                        '[]'
                    END
                )::JSONB;
            END IF;
        END $contacts$;
    END IF;
    
    -- Remover colunas antigas que foram migradas, se necessário
    -- (Manter comentado para evitar perda de dados - remover manualmente depois)
    /* 
    ALTER TABLE customers DROP COLUMN IF EXISTS street;
    ALTER TABLE customers DROP COLUMN IF EXISTS number;
    ALTER TABLE customers DROP COLUMN IF EXISTS complement;
    ALTER TABLE customers DROP COLUMN IF EXISTS district;
    ALTER TABLE customers DROP COLUMN IF EXISTS city;
    ALTER TABLE customers DROP COLUMN IF EXISTS state;
    ALTER TABLE customers DROP COLUMN IF EXISTS zip_code;
    ALTER TABLE customers DROP COLUMN IF EXISTS country;
    ALTER TABLE customers DROP COLUMN IF EXISTS email;
    ALTER TABLE customers DROP COLUMN IF EXISTS phone;
    ALTER TABLE customers DROP COLUMN IF EXISTS active;
    */
    
END
$$; 