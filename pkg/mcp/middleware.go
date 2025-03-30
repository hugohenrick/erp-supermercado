package mcp

import (
	"github.com/gin-gonic/gin"
	"github.com/hugohenrick/erp-supermercado/pkg/auth"
)

// MCPMiddleware adds MCP context to requests
func MCPMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user information from the context
		userID, tenantID, _, _, role, _ := auth.GetCurrentUser(c)

		// Store user context data directly
		c.Set("user_id", userID)
		c.Set("tenant_id", tenantID)
		c.Set("role", role)

		c.Next()
	}
}
