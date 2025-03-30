package chat

import (
	"context"

	"github.com/hugohenrick/erp-supermercado/pkg/chat"
)

type Repository interface {
	SaveMessage(ctx context.Context, message *chat.Message) error
	GetUserHistory(ctx context.Context, userID string, limit, offset int) ([]chat.Message, error)
	DeleteUserHistory(ctx context.Context, userID string) error
	CountUserMessages(ctx context.Context, userID string) (int, error)
}
