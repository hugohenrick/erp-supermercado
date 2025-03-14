package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hugohenrick/erp-supermercado/internal/adapter/api/controller"
	"github.com/hugohenrick/erp-supermercado/internal/adapter/api/route"
	"github.com/hugohenrick/erp-supermercado/internal/adapter/repository"
	"github.com/hugohenrick/erp-supermercado/internal/domain/branch"
	"github.com/hugohenrick/erp-supermercado/internal/domain/customer"
	"github.com/hugohenrick/erp-supermercado/internal/domain/tenant"
	"github.com/hugohenrick/erp-supermercado/internal/domain/user"
	"github.com/hugohenrick/erp-supermercado/internal/infrastructure/database"
	"github.com/hugohenrick/erp-supermercado/pkg/logger"
	pkgtenant "github.com/hugohenrick/erp-supermercado/pkg/tenant"
	"github.com/jackc/pgx/v5/pgxpool"
)

// App representa a aplicação
type App struct {
	Router          *gin.Engine
	DB              *pgxpool.Pool
	TenantRepo      tenant.Repository
	BranchRepo      branch.Repository
	UserRepo        user.Repository
	CustomerRepo    customer.Repository
	TenantValidator pkgtenant.TenantValidator
	Logger          logger.Logger
	Server          *http.Server
}

// NewApp cria uma nova instância da aplicação
func NewApp() *App {
	// Inicializar o banco de dados
	pool, err := database.NewPostgresDB()
	if err != nil {
		log.Fatalf("Erro ao conectar ao banco de dados: %v", err)
	}

	// Inicializar logger
	logger := logger.NewLogger()

	// Inicializar repositórios
	tenantRepo := repository.NewTenantRepository(pool)
	branchRepo := repository.NewBranchRepository(pool)
	userRepo := repository.NewUserRepository(pool)
	customerRepo := repository.NewCustomerRepository(pool)

	// Inicializar validador de tenant
	tenantValidator := repository.NewTenantValidator(tenantRepo)

	// Inicializar o router Gin
	router := gin.Default()

	// Configuração do servidor HTTP
	server := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	return &App{
		Router:          router,
		DB:              pool,
		TenantRepo:      tenantRepo,
		BranchRepo:      branchRepo,
		UserRepo:        userRepo,
		CustomerRepo:    customerRepo,
		TenantValidator: tenantValidator,
		Logger:          logger,
		Server:          server,
	}
}

// SetupRoutes configura as rotas da API
func (a *App) SetupRoutes() {
	// Middleware para validação de tenant
	a.Router.Use(pkgtenant.TenantMiddleware(a.TenantValidator))

	// Grupo de rotas para a API v1
	apiV1 := a.Router.Group("/api/v1")

	// Controladores
	tenantController := controller.NewTenantController(a.TenantRepo)
	branchController := controller.NewBranchController(a.BranchRepo)
	authController := controller.NewAuthController(a.UserRepo)
	userController := controller.NewUserController(a.UserRepo)
	customerController := controller.NewCustomerController(a.CustomerRepo, a.Logger)

	// Configurar rotas
	route.SetupTenantRoutes(apiV1, tenantController)
	route.SetupBranchRoutes(apiV1, branchController)
	route.SetupAuthRoutes(apiV1, authController)
	route.SetupUserRoutes(apiV1, userController)
	route.RegisterCustomerRoutes(apiV1, customerController)
}

// Start inicia o servidor HTTP
func (a *App) Start() {
	// Configurar as rotas
	a.SetupRoutes()

	// Canal para capturar sinais do sistema operacional
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Iniciar o servidor em uma goroutine
	go func() {
		log.Println("Servidor iniciado na porta 8080")
		if err := a.Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Erro ao iniciar o servidor: %v", err)
		}
	}()

	// Aguardar sinal para encerramento
	<-quit
	log.Println("Desligando o servidor...")

	// Criar um contexto com timeout para encerramento
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Encerrar o servidor graciosamente
	if err := a.Server.Shutdown(ctx); err != nil {
		log.Fatalf("Erro ao desligar o servidor: %v", err)
	}

	// Fechar a conexão com o banco de dados
	a.DB.Close()
	log.Println("Servidor encerrado")
}
