package chat

import (
	"context"
)

// Repository define a interface para operações de repositório do histórico de chat
type Repository interface {
	// SaveMessage salva uma nova mensagem no histórico
	SaveMessage(ctx context.Context, message *Message) error

	// GetUserHistory retorna o histórico de mensagens de um usuário
	GetUserHistory(ctx context.Context, userID string, limit, offset int) ([]Message, error)

	// DeleteUserHistory deleta todo o histórico de um usuário
	DeleteUserHistory(ctx context.Context, userID string) error

	// CountUserMessages conta quantas mensagens um usuário tem
	CountUserMessages(ctx context.Context, userID string) (int, error)
}
