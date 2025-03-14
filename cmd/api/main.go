package main

import (
	"log"

	"github.com/joho/godotenv"
)

func main() {
	// Carregar variáveis de ambiente
	if err := godotenv.Load(); err != nil {
		log.Printf("Aviso: Arquivo .env não encontrado: %v", err)
	}

	// Criar aplicação
	app := NewApp()

	// Iniciar o servidor
	app.Start()
}
