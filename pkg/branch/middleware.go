package branch

import (
	"context"

	"github.com/gin-gonic/gin"
)

// branchIDKey é a chave usada para armazenar o branch_id no contexto
type branchIDKey struct{}

// BranchIDKeyType retorna uma nova instância de branchIDKey para uso em outros pacotes
func BranchIDKeyType() branchIDKey {
	return branchIDKey{}
}

// BranchMiddleware cria um middleware para capturar o cabeçalho branch-id
func BranchMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		branchID := c.GetHeader("branch-id")
		if branchID != "" {
			// Armazenar o branch_id no contexto com a chave branchIDKey
			c.Set("branch_id", branchID)
			// Também colocar no contexto padrão para ser acessível por funções que usam context.Context
			ctx := context.WithValue(c.Request.Context(), branchIDKey{}, branchID)
			c.Request = c.Request.WithContext(ctx)
		}
		c.Next()
	}
}

// GetBranchID recupera o branch_id do contexto, se existir
func GetBranchID(ctx context.Context) string {
	if branchID, ok := ctx.Value(branchIDKey{}).(string); ok {
		return branchID
	}
	return ""
}
