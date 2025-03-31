package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/hugohenrick/erp-supermercado/internal/api/handlers"
	"github.com/hugohenrick/erp-supermercado/pkg/auth"
)

// SetupCustomerRoutes configura as rotas de clientes
func SetupCustomerRoutes(router *gin.RouterGroup, customerHandler *handlers.CustomerHandler) {
	customerRouter := router.Group("/customers")
	customerRouter.Use(auth.JWTAuthMiddleware())
	{
		// Implementação mínima para fins de exemplo
		customerRouter.GET("", func(c *gin.Context) {})
		customerRouter.GET("/:id", func(c *gin.Context) {})
		customerRouter.POST("", func(c *gin.Context) {})
		customerRouter.PUT("/:id", func(c *gin.Context) {})
		customerRouter.DELETE("/:id", func(c *gin.Context) {})
	}
}
