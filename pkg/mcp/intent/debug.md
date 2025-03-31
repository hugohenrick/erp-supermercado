# Como corrigir a integração MCP para salvar clientes corretamente

Identificamos que o problema era uma incompatibilidade entre os modelos de domínio utilizados na integração MCP e no repositório de clientes. O modelo simplificado usado pelo MCP (`pkg/domain/Customer`) não é compatível com o modelo interno (`internal/domain/customer/Customer`).

Para corrigir isso, criamos:

1. Um adaptador (`pkg/mcp/intent/adapter/customer_adapter.go`) que converte entre os dois modelos
2. Funções de inicialização (`pkg/mcp/intent/bootstrap.go`) que configuram os handlers corretamente

## Como atualizar a inicialização do MCP

Quando o `MCPHandler` é inicializado no seu código, você precisa alterar o seguinte:

```go
// De:
mcpHandler, err := handlers.NewMCPHandler(
    logger,
    userRepo,
    productRepo,
    customerRepo, // Isso não vai funcionar porque customerRepo não é compatível
)

// Para:
// Primeiro, obtenha o repositório interno
internalCustomerRepo := repositorio_interno_de_clientes // Você já tem isso em algum lugar 

// Em seguida, crie o handler usando nosso inicializador
customerHandler := intent.InitCustomerIntentHandler(logger, internalCustomerRepo)
productHandler := intent.InitProductIntentHandler(logger, productRepo)
userHandler := intent.InitUserIntentHandler(logger, userRepo)

// Crie a instância do MCP
mcpInstance := mcp.NewMCP(logger, apiKey, "claude-3-sonnet-20240229")

// Registre os handlers
mcpInstance.RegisterHandler(userHandler)
mcpInstance.RegisterHandler(productHandler)
mcpInstance.RegisterHandler(customerHandler)

// Crie o MCPHandler
mcpHandler := &handlers.MCPHandler{
    mcp:          mcpInstance,
    logger:       logger,
    userRepo:     userRepo,
    productRepo:  productRepo,
    customerRepo: nil, // Não precisamos mais usar isso diretamente
}
```

## Como testar

Após fazer essas alterações, você deve ser capaz de criar clientes via MCP e eles serão corretamente salvos na base de dados. A mensagem do MCP deve confirmar a criação com o ID real do cliente.

## Explicação técnica

O problema ocorria porque:

1. A interface `repository.CustomerRepository` esperava trabalhar com `pkg/domain/Customer` (modelo simples)
2. A implementação real usava `internal/domain/customer/Customer` (modelo complexo)

Nosso adaptador resolve isso convertendo entre os dois modelos, permitindo que você mantenha a simplicidade no MCP enquanto usa toda a complexidade necessária no repositório interno. 