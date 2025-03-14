package tenant

import (
	"context"
)

type contextKey string

const (
	// TenantIDKey é a chave usada para armazenar o tenant ID no contexto
	tenantIDKey contextKey = "tenant_id"
)

// SetTenantIDContext define o tenant ID no contexto
func SetTenantIDContext(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantIDKey, tenantID)
}

// GetTenantIDFromContext obtém o tenant ID do contexto
func GetTenantIDFromContext(ctx context.Context) string {
	if tenantID, ok := ctx.Value(tenantIDKey).(string); ok {
		return tenantID
	}
	return ""
}

// GetTenantID obtém o tenant ID de um contexto do Gin
func GetTenantID(c interface{}) string {
	if gc, ok := c.(interface{ GetString(string) string }); ok {
		return gc.GetString("tenant_id")
	}

	if gc, ok := c.(interface {
		Get(string) (interface{}, bool)
	}); ok {
		if val, exists := gc.Get("tenant_id"); exists {
			if tenantID, ok := val.(string); ok {
				return tenantID
			}
		}
	}

	return ""
}
