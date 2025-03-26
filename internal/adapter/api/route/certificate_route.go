package route

import (
	"github.com/gin-gonic/gin"
	"github.com/hugohenrick/erp-supermercado/internal/adapter/api/controller"
	"github.com/hugohenrick/erp-supermercado/pkg/auth"
)

// SetupCertificateRoutes configura as rotas para o módulo de certificados digitais
func SetupCertificateRoutes(router *gin.RouterGroup, certificateController *controller.CertificateController) {
	// Todas as rotas para certificados requerem autenticação e verificação de tenant
	certificateRouter := router.Group("/certificates")
	certificateRouter.Use(auth.JWTAuthMiddleware())
	{
		// Operações CRUD básicas
		certificateRouter.GET("", certificateController.List)
		certificateRouter.GET("/:id", certificateController.Get)
		certificateRouter.POST("", certificateController.Create)
		certificateRouter.POST("/upload", certificateController.Upload)
		certificateRouter.PUT("/:id", certificateController.Update)
		certificateRouter.DELETE("/:id", certificateController.Delete)

		// Operações adicionais
		certificateRouter.POST("/:id/activate", certificateController.Activate)
		certificateRouter.POST("/:id/deactivate", certificateController.Deactivate)
		certificateRouter.GET("/expiring", certificateController.ListExpiring)
	}
}
