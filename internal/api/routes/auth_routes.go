package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/hugohenrick/erp-supermercado/internal/api/handlers"
)

// SetupAuthRoutes configura as rotas de autenticação
func SetupAuthRoutes(router *gin.RouterGroup, authHandler *handlers.AuthHandler) {
	authRouter := router.Group("/auth")
	{
		// Implementação mínima para fins de exemplo
		authRouter.POST("/login", func(c *gin.Context) {})
		authRouter.POST("/refresh", func(c *gin.Context) {})
		authRouter.POST("/logout", func(c *gin.Context) {})
	}
}
