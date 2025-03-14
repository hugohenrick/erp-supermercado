package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	// Carregar variáveis de ambiente
	if err := godotenv.Load(); err != nil {
		log.Printf("Aviso: Arquivo .env não encontrado: %v", err)
	}

	// Criar aplicação
	app, err := NewApp()
	if err != nil {
		log.Fatalf("Falha ao inicializar aplicação: %v", err)
	}
	defer app.Close()

	// Configurar rotas
	basePath := os.Getenv("API_BASE_PATH")
	if basePath == "" {
		basePath = "/api/v1"
	}
	app.SetupRoutes(basePath)

	// Configurar porta
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Inicializar servidor com graceful shutdown
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: app.GetRouter(),
	}

	// Iniciar servidor em goroutine
	go func() {
		log.Printf("Servidor iniciado em http://localhost:%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Falha ao iniciar servidor: %v", err)
		}
	}()

	// Configurar graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Desligando servidor...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Erro ao desligar servidor: %v", err)
	}

	log.Println("Servidor desligado com sucesso")
}
