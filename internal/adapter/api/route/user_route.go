package route

import (
	"github.com/gin-gonic/gin"
	"github.com/hugohenrick/erp-supermercado/internal/adapter/api/controller"
	"github.com/hugohenrick/erp-supermercado/pkg/auth"
)

// SetupUserRoutes configura as rotas para o módulo de usuários
func SetupUserRoutes(router *gin.RouterGroup, userController *controller.UserController) {
	userRouter := router.Group("/users")
	{
		// Rotas que requerem autenticação e autorização de administrador
		userRouter.Use(auth.JWTAuthMiddleware())
		userRouter.Use(auth.RoleAuthMiddleware("admin"))
		{
			// Operações CRUD básicas
			userRouter.POST("", userController.Create)
			userRouter.GET("", userController.List)
			userRouter.GET("/:id", userController.GetByID)
			userRouter.PUT("/:id", userController.Update)
			userRouter.DELETE("/:id", userController.Delete)

			// Rotas especiais para filtragem e gestão
			userRouter.GET("/branch/:branch_id", userController.ListByBranch)
			userRouter.PATCH("/:id/status/:status", userController.UpdateStatus)
		}

		// Rota para alteração de senha (pode ser usada pelo próprio usuário ou por um admin)
		userRouter.Use(auth.JWTAuthMiddleware())
		userRouter.PATCH("/:id/password", userController.ChangePassword)
	}
}
