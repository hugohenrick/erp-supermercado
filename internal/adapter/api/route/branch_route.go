package route

import (
	"github.com/gin-gonic/gin"
	"github.com/hugohenrick/erp-supermercado/internal/adapter/api/controller"
	"github.com/hugohenrick/erp-supermercado/pkg/auth"
)

// SetupBranchRoutes configura as rotas para o módulo de filiais
func SetupBranchRoutes(router *gin.RouterGroup, branchController *controller.BranchController) {
	// Todas as rotas para filiais requerem autenticação e verificação de tenant
	branchRouter := router.Group("/branches")
	branchRouter.Use(auth.JWTAuthMiddleware())
	{
		// Operações CRUD básicas
		branchRouter.POST("", branchController.Create)
		branchRouter.GET("", branchController.List)
		branchRouter.GET("/:id", branchController.GetByID)
		branchRouter.GET("/main", branchController.GetMainBranch)
		branchRouter.PUT("/:id", branchController.Update)
		branchRouter.DELETE("/:id", branchController.Delete)
		
		// Operações adicionais
		branchRouter.PATCH("/:id/status/:status", branchController.UpdateStatus)
	}
} 