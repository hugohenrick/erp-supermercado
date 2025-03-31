package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/hugohenrick/erp-supermercado/internal/api/handlers"
	"github.com/hugohenrick/erp-supermercado/pkg/auth"
)

// SetupUserRoutes configura as rotas de usuários
func SetupUserRoutes(router *gin.RouterGroup, userHandler *handlers.UserHandler) {
	userRouter := router.Group("/users")
	userRouter.Use(auth.JWTAuthMiddleware())
	{
		// Implementação mínima para fins de exemplo
		userRouter.GET("", func(c *gin.Context) {})
		userRouter.GET("/:id", func(c *gin.Context) {})
		userRouter.POST("", func(c *gin.Context) {})
		userRouter.PUT("/:id", func(c *gin.Context) {})
		userRouter.DELETE("/:id", func(c *gin.Context) {})
	}
}
