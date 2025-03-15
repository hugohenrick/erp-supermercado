package route

import (
	"github.com/gin-gonic/gin"
	"github.com/hugohenrick/erp-supermercado/internal/adapter/api/controller"
)

// SetupSetupRoutes configura as rotas para configuração inicial do sistema
func SetupSetupRoutes(router *gin.RouterGroup, userController *controller.UserController) {
	setupRouter := router.Group("/setup")
	{
		// Rota para criar o primeiro usuário administrador de um tenant
		// Esta rota não requer autenticação, apenas o cabeçalho tenant-id
		setupRouter.POST("/admin", userController.CreateAdminUser)
	}
}
