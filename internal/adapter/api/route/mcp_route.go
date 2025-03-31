package route

import (
	"github.com/gin-gonic/gin"
	"github.com/hugohenrick/erp-supermercado/internal/adapter/api/controller"
	"github.com/hugohenrick/erp-supermercado/pkg/auth"
	"github.com/hugohenrick/erp-supermercado/pkg/logger"
	"github.com/hugohenrick/erp-supermercado/pkg/mcp"
	"github.com/hugohenrick/erp-supermercado/pkg/repository"
)

// ConfigureMCPRoutes configura as rotas do MCP
func ConfigureMCPRoutes(router *gin.RouterGroup, mcpClient *mcp.MCPClient, customerRepo repository.CustomerRepository, logger logger.Logger) {
	mcpController := controller.NewMCPController(mcpClient, customerRepo, logger)

	// Grupo de rotas MCP com autenticação JWT e middleware MCP
	mcpGroup := router.Group("/mcp")
	mcpGroup.Use(auth.JWTAuthMiddleware()) // Primeiro autenticação JWT
	mcpGroup.Use(mcp.MCPMiddleware())      // Depois middleware MCP
	{
		mcpGroup.POST("/message", mcpController.ProcessMessage)
		mcpGroup.GET("/history", mcpController.GetHistory)
		mcpGroup.DELETE("/history", mcpController.DeleteHistory)
	}
}
