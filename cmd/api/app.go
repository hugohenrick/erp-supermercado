package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/hugohenrick/erp-supermercado/internal/adapter/api/controller"
	"github.com/hugohenrick/erp-supermercado/internal/adapter/api/route"
	"github.com/hugohenrick/erp-supermercado/internal/adapter/repository"
	"github.com/hugohenrick/erp-supermercado/internal/domain/branch"
	"github.com/hugohenrick/erp-supermercado/internal/domain/certificate"
	"github.com/hugohenrick/erp-supermercado/internal/domain/chat"
	"github.com/hugohenrick/erp-supermercado/internal/domain/customer"
	"github.com/hugohenrick/erp-supermercado/internal/domain/fiscal"
	"github.com/hugohenrick/erp-supermercado/internal/domain/tenant"
	"github.com/hugohenrick/erp-supermercado/internal/domain/user"
	"github.com/hugohenrick/erp-supermercado/internal/infrastructure/database"
	pkgbranch "github.com/hugohenrick/erp-supermercado/pkg/branch"
	"github.com/hugohenrick/erp-supermercado/pkg/logger"
	"github.com/hugohenrick/erp-supermercado/pkg/mcp"
	pkgtenant "github.com/hugohenrick/erp-supermercado/pkg/tenant"
	"github.com/jackc/pgx/v5/pgxpool"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	// Importação necessária para inicializar a documentação Swagger
	_ "github.com/hugohenrick/erp-supermercado/docs"
)

// App representa a aplicação
type App struct {
	Router           *gin.Engine
	DB               *pgxpool.Pool
	TenantRepo       tenant.Repository
	BranchRepo       branch.Repository
	UserRepo         user.Repository
	CustomerRepo     customer.Repository
	CertificateRepo  certificate.Repository
	FiscalConfigRepo fiscal.Repository
	ChatRepo         chat.Repository
	TenantValidator  pkgtenant.TenantValidator
	Logger           logger.Logger
	MCPClient        *mcp.MCPClient
	Server           *http.Server
}

// NewApp cria uma nova instância da aplicação
func NewApp() *App {
	// Load environment variables
	// Inicializar o banco de dados
	pool, err := database.NewPostgresDB()
	if err != nil {
		log.Fatalf("Erro ao conectar ao banco de dados: %v", err)
	}

	// Inicializar logger
	logger := logger.NewLogger()
	// Initialize services
	// Inicializar repositórios
	tenantRepo := repository.NewTenantRepository(pool)
	branchRepo := repository.NewBranchRepository(pool)
	userRepo := repository.NewUserRepository(pool)
	customerRepo := repository.NewCustomerRepository(pool)
	certificateRepo := repository.NewCertificateRepository(pool)
	fiscalConfigRepo := repository.NewFiscalRepository(pool)
	chatRepo := repository.NewChatRepository(pool)
	// Initialize controllers
	// Inicializar validador de tenant
	tenantValidator := repository.NewTenantValidator(tenantRepo)
	// Initialize router
	// Inicializar MCP client
	mcpClient, err := mcp.NewMCPClient(logger, chatRepo)
	if err != nil {
		log.Fatalf("Erro ao inicializar MCP client: %v", err)
	}

	// Inicializar o router Gin
	router := gin.Default()

	// Add CORS middleware
	// Configuração do CORS
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Tenant-ID", "tenant-id", "branch-id"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Initialize routes
	// Adicionar middleware MCP após o CORS
	router.Use(mcp.MCPMiddleware())

	// Obter a porta da API das variáveis de ambiente
	apiPort := os.Getenv("API_PORT")
	if apiPort == "" {
		apiPort = "8084" // Valor padrão se não estiver definido
	}

	// Configuração do servidor HTTP
	server := &http.Server{
		Addr:    ":" + apiPort,
		Handler: router,
	}
	return &App{
		Router:           router,
		DB:               pool,
		TenantRepo:       tenantRepo,
		BranchRepo:       branchRepo,
		UserRepo:         userRepo,
		CustomerRepo:     customerRepo,
		CertificateRepo:  certificateRepo,
		FiscalConfigRepo: fiscalConfigRepo,
		ChatRepo:         chatRepo,
		TenantValidator:  tenantValidator,
		Logger:           logger,
		MCPClient:        mcpClient,
		Server:           server,
	}
}

// SetupRoutes configura as rotas da API
func (a *App) SetupRoutes() {
	// Configurar Swagger
	a.Router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Grupo base de rotas com prefixo /api/v1
	apiV1 := a.Router.Group("/api/v1")

	// Middleware para validação de tenant aplicado apenas nas rotas da API
	apiV1.Use(pkgtenant.TenantMiddleware(a.TenantValidator))

	// Middleware para capturar o branch_id do cabeçalho
	apiV1.Use(pkgbranch.BranchMiddleware())

	// Criar instâncias dos controladores
	tenantController := controller.NewTenantController(a.TenantRepo, a.DB)
	branchController := controller.NewBranchController(a.BranchRepo)
	authController := controller.NewAuthController(a.UserRepo)
	userController := controller.NewUserController(a.UserRepo)
	customerController := controller.NewCustomerController(a.CustomerRepo, a.Logger)
	certificateController := controller.NewCertificateController(a.CertificateRepo, a.Logger)
	fiscalController := controller.NewFiscalController(a.FiscalConfigRepo, a.Logger)

	// Configurar rotas para cada módulo
	route.SetupTenantRoutes(apiV1, tenantController)
	route.SetupBranchRoutes(apiV1, branchController)
	route.SetupAuthRoutes(apiV1, authController)
	route.SetupUserRoutes(apiV1, userController)
	route.RegisterCustomerRoutes(apiV1, customerController)
	route.SetupSetupRoutes(apiV1, userController)
	route.SetupCertificateRoutes(apiV1, certificateController)
	route.SetupFiscalRoutes(apiV1, fiscalController)
	route.ConfigureMCPRoutes(apiV1, a.MCPClient)

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
		log.Println("Servidor iniciado na porta 8084")
		log.Println("Documentação Swagger disponível em: http://localhost:8084/swagger/index.html")
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
