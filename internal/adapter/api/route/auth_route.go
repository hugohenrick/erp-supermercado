package route

import (
	"github.com/gin-gonic/gin"
	"github.com/hugohenrick/erp-supermercado/internal/adapter/api/controller"
	"github.com/hugohenrick/erp-supermercado/pkg/auth"
)

// SetupAuthRoutes configura as rotas para autenticação
func SetupAuthRoutes(router *gin.RouterGroup, authController *controller.AuthController) {
	authRouter := router.Group("/auth")
	{
		// Rota de login (não requer autenticação)
		authRouter.POST("/login", authController.Login)
		
		// Rota para renovar token (não requer autenticação pois usa o token de refresh)
		authRouter.POST("/refresh-token", authController.RefreshToken)
		
		// Rota para obter informações do usuário logado (requer autenticação)
		authRouter.GET("/me", auth.JWTAuthMiddleware(), authController.Me)
	}
} 