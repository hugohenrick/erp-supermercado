package main

import (
	"github.com/gin-gonic/gin"
	"github.com/hugohenrick/erp-supermercado/internal/adapter/api/controller"
	"github.com/hugohenrick/erp-supermercado/internal/adapter/repository"
	"github.com/hugohenrick/erp-supermercado/internal/infrastructure/database"
	"github.com/hugohenrick/erp-supermercado/pkg/tenant"
)

// App representa a aplicação e suas dependências
type App struct {
	router           *gin.Engine
	db               *database.PostgresDB
	tenantRepository *repository.PostgresTenantRepository
	branchRepository *repository.PostgresBranchRepository
	tenantValidator  *repository.TenantValidator
	tenantMiddleware gin.HandlerFunc
	tenantController *controller.TenantController
	branchController *controller.BranchController
}

// NewApp cria uma nova instância do aplicativo
func NewApp() (*App, error) {
	// Configurar banco de dados
	config := database.NewPostgresConfigFromEnv()
	db, err := database.NewPostgresDB(config)
	if err != nil {
		return nil, err
	}

	// Criar repositórios
	tenantRepo := repository.NewPostgresTenantRepository(db)
	branchRepo := repository.NewPostgresBranchRepository(db)

	// Criar validador de tenant
	tenantValidator := repository.NewTenantValidator(tenantRepo)

	// Criar extrator de tenant
	tenantExtractor := tenant.NewHeaderTenantExtractor("")

	// Criar middleware de tenant
	tenantMiddleware := tenant.Middleware(tenantExtractor, tenantValidator)

	// Criar controllers
	tenantController := controller.NewTenantController(tenantRepo)
	branchController := controller.NewBranchController(branchRepo)

	// Configurar router com modo correto
	router := gin.Default()

	// Configurar CORS e outros middlewares globais
	router.Use(gin.Recovery())

	return &App{
		router:           router,
		db:               db,
		tenantRepository: tenantRepo,
		branchRepository: branchRepo,
		tenantValidator:  tenantValidator,
		tenantMiddleware: tenantMiddleware,
		tenantController: tenantController,
		branchController: branchController,
	}, nil
}

// SetupRoutes configura as rotas da aplicação
func (a *App) SetupRoutes(basePath string) {
	api := a.router.Group(basePath)

	// Health check
	api.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"version": "1.0.0",
		})
	})

	// Rotas que não precisam de autenticação de tenant
	tenantsRoutes := api.Group("/tenants")
	{
		tenantsRoutes.POST("", a.tenantController.Create)
		tenantsRoutes.GET("", a.tenantController.List)
		tenantsRoutes.GET("/:id", a.tenantController.GetByID)
		tenantsRoutes.GET("/document/:document", a.tenantController.GetByDocument)
		tenantsRoutes.PUT("/:id", a.tenantController.Update)
		tenantsRoutes.DELETE("/:id", a.tenantController.Delete)
		tenantsRoutes.PATCH("/:id/status/:status", a.tenantController.UpdateStatus)
	}

	// Rotas que precisam de autenticação de tenant
	// Usamos o middleware de tenant aqui
	tenantProtectedRoutes := api.Group("")
	tenantProtectedRoutes.Use(a.tenantMiddleware)

	// Rotas para filiais
	branchesRoutes := tenantProtectedRoutes.Group("/branches")
	{
		branchesRoutes.POST("", a.branchController.Create)
		branchesRoutes.GET("", a.branchController.List)
		branchesRoutes.GET("/:id", a.branchController.GetByID)
		branchesRoutes.GET("/main", a.branchController.GetMainBranch)
		branchesRoutes.PUT("/:id", a.branchController.Update)
		branchesRoutes.DELETE("/:id", a.branchController.Delete)
		branchesRoutes.PATCH("/:id/status/:status", a.branchController.UpdateStatus)
	}
}

// GetRouter retorna o router da aplicação
func (a *App) GetRouter() *gin.Engine {
	return a.router
}

// Close libera os recursos da aplicação
func (a *App) Close() {
	if a.db != nil {
		a.db.Close()
	}
}
