package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/hugohenrick/erp-supermercado/internal/api/handlers"
	"github.com/hugohenrick/erp-supermercado/pkg/auth"
)

// SetupProductRoutes configura as rotas de produtos
func SetupProductRoutes(router *gin.RouterGroup, productHandler *handlers.ProductHandler) {
	productRouter := router.Group("/products")
	productRouter.Use(auth.JWTAuthMiddleware())
	{
		// Implementação mínima para fins de exemplo
		productRouter.GET("", func(c *gin.Context) {})
		productRouter.GET("/:id", func(c *gin.Context) {})
		productRouter.POST("", func(c *gin.Context) {})
		productRouter.PUT("/:id", func(c *gin.Context) {})
		productRouter.DELETE("/:id", func(c *gin.Context) {})
	}
}
