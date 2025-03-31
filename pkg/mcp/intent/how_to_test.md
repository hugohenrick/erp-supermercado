# Testando a Funcionalidade de Cadastro de Cliente via MCP

Após as alterações para resolver o problema de não salvar clientes no banco de dados, é importante testar corretamente a implementação.

## Como Testar

1. **Limpe o histórico do MCP**: É importante iniciar com um histórico limpo para evitar interferência de conversas anteriores
   ```
   POST /api/v1/mcp/clear-history
   ```

2. **Envie uma mensagem para criar um cliente**: Use uma instrução clara
   ```
   POST /api/v1/mcp/message
   {
     "message": "Cadastre um novo cliente chamado João Silva com CPF 123.456.789-00, email joao@exemplo.com e telefone (11) 99999-9999"
   }
   ```

3. **Observe os logs**: Você deve ver mensagens como:
   ```
   Intent detected intent=create_customer
   Created confirmation flow session_id=1921522d-2c64-420d-8cac-9fdc75388d95:917c7804-f8d6-4e9c-a78d-77a3137591eb
   Saved session state session_id=... state=awaiting_confirmation
   ```

4. **Envie a confirmação**: Um simples "confirmado" ou "correto" 
   ```
   POST /api/v1/mcp/message
   {
     "message": "confirmado"
   }
   ```

5. **Observe os logs novamente**: Agora você deve ver:
   ```
   Found active session in MCP session_id=... state=awaiting_confirmation
   Injecting session state session_id=... state=awaiting_confirmation
   Found active session session_id=... state=awaiting_confirmation intent=create_customer
   Executando ação confirmada intent=create_customer
   CustomerRepositoryAdapter.Create called id=... name=João Silva
   ```

6. **Verifique o banco de dados**: Você deve ver o cliente criado no banco de dados
   ```sql
   SELECT * FROM customers WHERE name = 'João Silva';
   ```

## Possíveis Problemas e Soluções

1. **Sessão não está sendo mantida**: Se você vir `has_active_session=false` quando envia a confirmação, significa que a sessão está sendo perdida entre requisições
   - Solução: Verifique se o MCPHandler está sendo recriado entre requisições ou se há múltiplas instâncias do serviço

2. **Intenção não é detectada**: Se você vir `No intent detected in message` ao enviar a confirmação
   - Solução: Verifique os padrões de confirmação no `processContinuation` para garantir que sua resposta está sendo reconhecida

3. **Erro na adaptação do modelo**: Se você vir erros como "Failed to create internal customer"
   - Solução: Verifique os logs detalhados da adaptação para entender qual campo está causando problemas

## Fluxo de Logs Esperado

```
Processing message session_id=... has_active_session=false
No active session found, detecting intent
Intent detected intent=create_customer confidence=0.8
Created confirmation flow session_id=... intent=create_customer state=awaiting_confirmation
Saved session state session_id=... state=awaiting_confirmation

Processing message session_id=... has_active_session=true
Found active session in MCP session_id=... state=awaiting_confirmation intent=create_customer
Injecting session state session_id=... state=awaiting_confirmation
Found active session session_id=... state=awaiting_confirmation intent=create_customer
Executando ação confirmada intent=create_customer message=confirmado
CustomerRepositoryAdapter.Create called id=... name=...
Creating internal customer with data tenant_id=... name=...
Customer successfully created id=... name=...
```

Se você vir este fluxo de logs completo, a funcionalidade está operando corretamente. 