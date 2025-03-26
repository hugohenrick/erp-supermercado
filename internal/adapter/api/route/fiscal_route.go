package route

import (
	"github.com/gin-gonic/gin"
	"github.com/hugohenrick/erp-supermercado/internal/adapter/api/controller"
	"github.com/hugohenrick/erp-supermercado/pkg/auth"
)

// SetupFiscalRoutes configura as rotas para o módulo de configurações fiscais
func SetupFiscalRoutes(router *gin.RouterGroup, fiscalController *controller.FiscalController) {
	// Todas as rotas para configurações fiscais requerem autenticação e verificação de tenant
	fiscalRouter := router.Group("/fiscal/configs")
	fiscalRouter.Use(auth.JWTAuthMiddleware())
	{
		// Operações CRUD básicas
		fiscalRouter.GET("", fiscalController.List)
		fiscalRouter.GET("/:id", fiscalController.Get)
		fiscalRouter.POST("", fiscalController.Create)
		fiscalRouter.PUT("/:id", fiscalController.Update)
		fiscalRouter.DELETE("/:id", fiscalController.Delete)

		// Operações por filial
		fiscalRouter.GET("/branch/:branch_id", fiscalController.GetByBranch)
		fiscalRouter.POST("/branch/:branch_id/increment-nfe", fiscalController.IncrementNFeNumber)
		fiscalRouter.POST("/branch/:branch_id/increment-nfce", fiscalController.IncrementNFCeNumber)
		fiscalRouter.POST("/branch/:branch_id/contingency", fiscalController.UpdateContingency)
	}
}
