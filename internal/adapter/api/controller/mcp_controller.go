package controller

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hugohenrick/erp-supermercado/pkg/domain"
	"github.com/hugohenrick/erp-supermercado/pkg/logger"
	"github.com/hugohenrick/erp-supermercado/pkg/mcp"
	"github.com/hugohenrick/erp-supermercado/pkg/repository"
)

// MCPController handles MCP-related requests
type MCPController struct {
	mcpClient    *mcp.MCPClient
	customerRepo repository.CustomerRepository
	logger       logger.Logger
}

// NewMCPController creates a new MCP controller
func NewMCPController(mcpClient *mcp.MCPClient, customerRepo repository.CustomerRepository, logger logger.Logger) *MCPController {
	return &MCPController{
		mcpClient:    mcpClient,
		customerRepo: customerRepo,
		logger:       logger,
	}
}

type MCPMessageRequest struct {
	Message string `json:"message" binding:"required"`
}

// ProcessMessage godoc
// @Summary Process a message through MCP
// @Description Process a user message and return the response with chat history
// @Tags MCP
// @Accept json
// @Produce json
// @Param message body MCPMessageRequest true "Message to process"
// @Success 200 {object} chat.Message
// @Router /api/v1/mcp/message [post]
func (c *MCPController) ProcessMessage(ctx *gin.Context) {
	var req MCPMessageRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// EMERGENCY FIX: Direct handling of customer creation
	// This bypasses the complexity to ensure customer creation works
	message := req.Message
	lowerMsg := strings.ToLower(message)

	// Check if this looks like a customer creation request
	hasNome := strings.Contains(message, "Nome:") || strings.Contains(message, "nome:")
	hasCPF := strings.Contains(message, "CPF:") || strings.Contains(message, "cpf:")
	hasCustomerKeywords := strings.Contains(lowerMsg, "cadastre") && strings.Contains(lowerMsg, "cliente")

	// Check for customer listing requests
	isListCustomersRequest := (strings.Contains(lowerMsg, "lista") ||
		strings.Contains(lowerMsg, "busca") ||
		strings.Contains(lowerMsg, "mostrar") ||
		strings.Contains(lowerMsg, "exibir") ||
		strings.Contains(lowerMsg, "encontrar") ||
		strings.Contains(lowerMsg, "pesquisar")) &&
		strings.Contains(lowerMsg, "cliente")

	// Get tenant ID from context early as we'll need it for both create and list operations
	tenantID := ctx.GetString("tenant_id")
	if tenantID == "" {
		c.logger.Error("No tenant ID found in context")
		ctx.JSON(http.StatusOK, gin.H{
			"response": "Erro ao identificar o tenant. Por favor, tente novamente.",
		})
		return
	}

	if (hasNome && hasCPF) || (hasCustomerKeywords && hasNome) {
		// This looks like a customer creation request
		// Log detection for debugging
		c.logger.Info("DETECTED CUSTOMER CREATION REQUEST:",
			"message", message)

		// Extract customer data
		nameRegex := regexp.MustCompile(`(?i)Nome\s*:?\s*([^\r\n]+)`)
		nameMatch := nameRegex.FindStringSubmatch(message)
		name := ""
		if len(nameMatch) > 1 {
			name = strings.TrimSpace(nameMatch[1])
		}

		// Extract document (CPF/CNPJ)
		docRegex := regexp.MustCompile(`(?i)(?:CPF|CNPJ|Documento)\s*:?\s*([^\r\n]+)`)
		docMatch := docRegex.FindStringSubmatch(message)
		doc := ""
		if len(docMatch) > 1 {
			doc = strings.TrimSpace(docMatch[1])
		}

		// Extract email
		emailRegex := regexp.MustCompile(`(?i)E-?mail\s*:?\s*([^\r\n]+)`)
		emailMatch := emailRegex.FindStringSubmatch(message)
		email := ""
		if len(emailMatch) > 1 {
			email = strings.TrimSpace(emailMatch[1])
		}

		// Extract phone
		phoneRegex := regexp.MustCompile(`(?i)(?:Telefone|Fone|Celular)\s*:?\s*([^\r\n]+)`)
		phoneMatch := phoneRegex.FindStringSubmatch(message)
		phone := ""
		if len(phoneMatch) > 1 {
			phone = strings.TrimSpace(phoneMatch[1])
		}

		// Extract address
		addrRegex := regexp.MustCompile(`(?i)(?:Endereço|Endereco|Localização|Localizacao)\s*:?\s*([^\r\n]+)`)
		addrMatch := addrRegex.FindStringSubmatch(message)
		addr := ""
		if len(addrMatch) > 1 {
			addr = strings.TrimSpace(addrMatch[1])
		}

		c.logger.Info("Extracted customer details",
			"name", name,
			"doc", doc,
			"email", email,
			"phone", phone,
			"address", addr)

		// Check if we have a name (required)
		if name == "" {
			ctx.JSON(http.StatusOK, gin.H{
				"response": "Para cadastrar um cliente, preciso pelo menos do nome. Por favor, informe o nome completo.",
			})
			return
		}

		// Create customer object
		customer := &domain.Customer{
			ID:        uuid.New().String(),
			Name:      name,
			Document:  doc,
			Email:     email,
			Phone:     phone,
			Address:   addr,
			TenantID:  tenantID,
			Active:    true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		// Save customer to database
		c.logger.Info("Saving customer to database",
			"name", name,
			"tenant_id", tenantID)

		err := c.customerRepo.Create(tenantID, customer)
		if err != nil {
			c.logger.Error("Failed to create customer",
				"error", err,
				"name", name)

			ctx.JSON(http.StatusOK, gin.H{
				"response": fmt.Sprintf("Não foi possível cadastrar o cliente: %v", err),
			})
			return
		}

		// Success!
		c.logger.Info("Customer created successfully",
			"customer_id", customer.ID,
			"name", customer.Name)

		// Format JSON response compatible with client expectations
		ctx.JSON(http.StatusOK, gin.H{
			"response": fmt.Sprintf("✅ Cliente '%s' cadastrado com sucesso! O ID do novo cliente é #%s.", name, customer.ID),
			"history":  []interface{}{}, // Empty history since this is a direct operation
		})
		return
	}

	// If this is a request to list or search for customers
	if isListCustomersRequest {
		c.logger.Info("DETECTED CUSTOMER LISTING REQUEST:",
			"message", message)

		// Extract search parameters
		nameSearchRegex := regexp.MustCompile(`(?i)(?:por|com|de|chamado|nome)\s+(?:nome|chamado)?\s*(?::|é|como|igual a)?\s*["']?([^"'\n,]+)["']?`)
		docSearchRegex := regexp.MustCompile(`(?i)(?:por|com|de)\s+(?:cpf|cnpj|documento)\s*(?::|é|como|igual a)?\s*["']?([^"'\n,]+)["']?`)

		var customers []*domain.Customer
		var err error
		var searchType string
		var searchValue string

		// Check if searching by document (CPF/CNPJ)
		docMatch := docSearchRegex.FindStringSubmatch(message)
		if len(docMatch) > 1 && strings.TrimSpace(docMatch[1]) != "" {
			searchValue = strings.TrimSpace(docMatch[1])
			searchType = "document"
			c.logger.Info("Searching customer by document", "document", searchValue)

			// Search by document
			customer, err := c.customerRepo.FindByDocument(tenantID, searchValue)
			if err == nil && customer != nil {
				customers = []*domain.Customer{customer}
			}
		} else {
			// Check if searching by name
			nameMatch := nameSearchRegex.FindStringSubmatch(message)
			if len(nameMatch) > 1 && strings.TrimSpace(nameMatch[1]) != "" {
				searchValue = strings.TrimSpace(nameMatch[1])
				searchType = "name"
				c.logger.Info("Searching customer by name", "name", searchValue)

				// Search by name
				customers, err = c.customerRepo.FindByName(tenantID, searchValue)
			} else {
				// List all customers if no specific search criteria
				searchType = "all"
				c.logger.Info("Listing all customers")

				// Get all customers
				customers, err = c.customerRepo.FindAll(tenantID)
			}
		}

		if err != nil {
			c.logger.Error("Error searching customers",
				"error", err,
				"searchType", searchType,
				"searchValue", searchValue)

			ctx.JSON(http.StatusOK, gin.H{
				"response": fmt.Sprintf("Ocorreu um erro ao buscar clientes: %v", err),
			})
			return
		}

		// Format the customer list response
		var responseMsg string

		if len(customers) == 0 {
			if searchType == "all" {
				responseMsg = "Não encontrei nenhum cliente cadastrado."
			} else {
				var typeLabel string
				if searchType == "name" {
					typeLabel = "nome"
				} else {
					typeLabel = "documento"
				}
				responseMsg = fmt.Sprintf("Não encontrei nenhum cliente com %s '%s'.",
					typeLabel, searchValue)
			}
		} else if len(customers) == 1 {
			// Single customer found
			customer := customers[0]
			responseMsg = fmt.Sprintf("✅ Cliente encontrado:\n\n"+
				"**ID:** %s\n"+
				"**Nome:** %s\n"+
				"**Documento:** %s\n",
				customer.ID, customer.Name, customer.Document)

			if customer.Email != "" {
				responseMsg += fmt.Sprintf("**Email:** %s\n", customer.Email)
			}

			if customer.Phone != "" {
				responseMsg += fmt.Sprintf("**Telefone:** %s\n", customer.Phone)
			}

			if customer.Address != "" {
				responseMsg += fmt.Sprintf("**Endereço:** %s\n", customer.Address)
			}

			var statusText string
			if customer.Active {
				statusText = "Ativo"
			} else {
				statusText = "Inativo"
			}
			responseMsg += fmt.Sprintf("\n**Status:** %s", statusText)
		} else {
			// Multiple customers found
			responseMsg = fmt.Sprintf("✅ Encontrei %d clientes:\n\n", len(customers))

			for i, customer := range customers {
				if i < 10 { // Limit to 10 results to avoid overly long responses
					responseMsg += fmt.Sprintf("%d. **%s** (%s) - ID: %s\n",
						i+1, customer.Name, customer.Document, customer.ID)
				} else {
					responseMsg += fmt.Sprintf("\n... e mais %d cliente(s).", len(customers)-10)
					break
				}
			}

			responseMsg += "\n\nPara ver detalhes de um cliente específico, peça para buscar pelo nome ou documento."
		}

		// Return the formatted customer list
		ctx.JSON(http.StatusOK, gin.H{
			"response": responseMsg,
			"history":  []interface{}{}, // Empty history since this is a direct operation
		})
		return
	}

	mcpContextData := mcp.GetContextFromRequest(ctx)
	if mcpContextData == nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "MCP context not found"})
		return
	}

	response, err := c.mcpClient.ProcessWithContext(ctx, req.Message, mcpContextData)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	history, err := c.mcpClient.GetChatHistory(ctx, mcpContextData.UserID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Sort history for display in client
	// The history is returned from database newest first,
	// but the client needs oldest first for proper display
	if len(history) > 0 {
		// Reverse the history array for chronological display
		for i, j := 0, len(history)-1; i < j; i, j = i+1, j-1 {
			history[i], history[j] = history[j], history[i]
		}
	}

	// Return the response with history
	ctx.JSON(http.StatusOK, gin.H{
		"response": response,
		"history":  history,
	})
}

// GetHistory godoc
// @Summary Get chat history
// @Description Get the chat history for the current user
// @Tags MCP
// @Produce json
// @Success 200 {array} chat.Message
// @Router /api/v1/mcp/history [get]
func (c *MCPController) GetHistory(ctx *gin.Context) {
	mcpContextData := mcp.GetContextFromRequest(ctx)
	if mcpContextData == nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "MCP context not found"})
		return
	}

	history, err := c.mcpClient.GetChatHistory(ctx, mcpContextData.UserID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Sort history for display in client
	// The history is returned from database newest first,
	// but the client needs oldest first for proper display
	if len(history) > 0 {
		// Reverse the history array for chronological display
		for i, j := 0, len(history)-1; i < j; i, j = i+1, j-1 {
			history[i], history[j] = history[j], history[i]
		}
	}

	ctx.JSON(http.StatusOK, gin.H{
		"history": history,
	})
}

// DeleteHistory godoc
// @Summary Delete chat history
// @Description Delete the chat history for the current user
// @Tags MCP
// @Success 200 {object} string
// @Router /api/v1/mcp/history [delete]
func (c *MCPController) DeleteHistory(ctx *gin.Context) {
	mcpContextData := mcp.GetContextFromRequest(ctx)
	if mcpContextData == nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "MCP context not found"})
		return
	}

	err := c.mcpClient.DeleteChatHistory(ctx, mcpContextData.UserID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Chat history deleted successfully",
	})
}
