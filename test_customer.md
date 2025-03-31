# Testes para Integração do Cadastro de Clientes via MCP

Este documento descreve passo a passo como testar e depurar a integração de cadastro de clientes através do MCP.

## Passo 1: Limpar o histórico

Primeiro, limpe o histórico para garantir um teste limpo.

```
POST /api/v1/mcp/clear-history
```

## Passo 2: Verificar os logs

Habilite os logs de debug:

```bash
# No arquivo .env ou na sua shell
export LOG_LEVEL=debug
# Reinicie a aplicação
make run
```

## Passo 3: Enviar uma Solicitação de Criação de Cliente

Envie uma solicitação explícita de criação de cliente:

```
POST /api/v1/mcp/message
{
  "message": "cadastre um novo cliente: Nome: João Silva, CPF: 123.456.789-00, Email: joao@teste.com, Telefone: (11) 99999-9999"
}
```

## Passo 4: Verificar os Logs

Nos logs, você deve procurar estas entradas específicas:

```
Checking if customer handler can handle message message="cadastre um novo cliente: Nome: João Silva..."
Message contains form-style customer creation
Extracting intent from message
Detected structured customer creation message
Executing intent intent=create_customer
```

Se você não vir essas mensagens, significa que o sistema não está reconhecendo a intenção de criação de cliente.

## Passo 5: Responder à Confirmação

Quando o sistema pedir uma confirmação, responda com:

```
POST /api/v1/mcp/message
{
  "message": "confirme"
}
```

## Passo 6: Verificar os Logs Novamente

Você deve ver:

```
Processing message with context has_intent_session=true
Found active session in MCP session_id=... state=awaiting_confirmation
Injecting session state session_id=... state=awaiting_confirmation
Found active session session_id=... state=awaiting_confirmation intent=create_customer
Executando ação confirmada intent=create_customer
CustomerRepositoryAdapter.Create called id=... name=João Silva
Customer successfully created id=... name=João Silva
```

## Passo 7: Verificar o Banco de Dados

Verifique se o cliente foi realmente criado no banco de dados:

```sql
SELECT * FROM customers WHERE email = 'joao@teste.com';
```

## Problemas Comuns

### 1. Intent não é reconhecido

**Sintoma**: Os logs não mostram "Extracting intent from message"

**Solução**: 
- Verifique se o handler está registrado corretamente
- Verifique se a mensagem segue o formato esperado nos padrões regex

### 2. Sessão não é mantida

**Sintoma**: `has_intent_session=false` quando deveria ser true

**Solução**: 
- Verifique se o sessionID está sendo gerado consistentemente
- Verifique se a sessão está sendo salva no MCP

### 3. Adapter não está sendo chamado

**Sintoma**: Não aparece "CustomerRepositoryAdapter.Create called" nos logs

**Solução**:
- Verifique se o adaptador foi inicializado corretamente
- Verifique se a ação está sendo executada corretamente

## Utilizando o Postman ou Curl

Para facilitar os testes, utilize o Postman ou curl para enviar as requisições:

```bash
# Limpar histórico
curl -X POST http://localhost:8080/api/v1/mcp/clear-history \
  -H "Authorization: Bearer YOUR_TOKEN"

# Enviar mensagem de criação
curl -X POST http://localhost:8080/api/v1/mcp/message \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"message": "cadastre um novo cliente: Nome: João Silva, CPF: 123.456.789-00, Email: joao@teste.com, Telefone: (11) 99999-9999"}'

# Enviar confirmação
curl -X POST http://localhost:8080/api/v1/mcp/message \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"message": "confirme"}'
``` 