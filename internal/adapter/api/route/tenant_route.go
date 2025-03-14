package route

import (
	"github.com/gin-gonic/gin"
	"github.com/hugohenrick/erp-supermercado/internal/adapter/api/controller"
)

// SetupTenantRoutes configura as rotas para o módulo de tenants
func SetupTenantRoutes(router *gin.RouterGroup, tenantController *controller.TenantController) {
	tenantRouter := router.Group("/tenants")
	{
		// Operações CRUD básicas
		tenantRouter.POST("", tenantController.Create)
		tenantRouter.GET("", tenantController.List)
		tenantRouter.GET("/:id", tenantController.GetByID)
		tenantRouter.GET("/document/:document", tenantController.GetByDocument)
		tenantRouter.PUT("/:id", tenantController.Update)
		tenantRouter.DELETE("/:id", tenantController.Delete)
		
		// Operações adicionais
		tenantRouter.PATCH("/:id/status/:status", tenantController.UpdateStatus)
	}
}
 