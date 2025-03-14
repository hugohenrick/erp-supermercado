package route

import (
	"github.com/gin-gonic/gin"
	"github.com/hugohenrick/erp-supermercado/internal/adapter/api/controller"
	"github.com/hugohenrick/erp-supermercado/pkg/middleware"
)

// RegisterCustomerRoutes registra as rotas do módulo de clientes
func RegisterCustomerRoutes(r *gin.RouterGroup, customerController *controller.CustomerController) {
	customers := r.Group("/customers")
	customers.Use(middleware.AuthMiddleware())
	{
		customers.POST("", customerController.Create)
		customers.GET("", customerController.List)
		customers.GET("/:id", customerController.Get)
		customers.PUT("/:id", customerController.Update)
		customers.DELETE("/:id", customerController.Delete)
		customers.PATCH("/:id/status", customerController.UpdateStatus)
		customers.GET("/document/:document", customerController.FindByDocument)
		customers.GET("/search", customerController.FindByName)
	}
}
