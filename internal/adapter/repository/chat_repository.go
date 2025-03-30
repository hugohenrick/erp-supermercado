package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/hugohenrick/erp-supermercado/pkg/chat"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ChatRepository struct {
	db *pgxpool.Pool
}

func NewChatRepository(db *pgxpool.Pool) chat.Repository {
	return &ChatRepository{
		db: db,
	}
}

func (r *ChatRepository) SaveMessage(ctx context.Context, message *chat.Message) error {
	// Extrair tenant_id do contexto
	tenantID := ctx.Value("tenant_id").(string)
	if tenantID == "" {
		return fmt.Errorf("tenant_id não encontrado no contexto")
	}

	// Obter o schema do tenant
	var schema string
	err := r.db.QueryRow(ctx, "SELECT schema FROM public.tenants WHERE id = $1", tenantID).Scan(&schema)
	if err != nil {
		return fmt.Errorf("erro ao obter schema do tenant: %w", err)
	}

	query := fmt.Sprintf(`
		INSERT INTO %s.chat_history (id, tenant_id, user_id, role, content, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, schema)

	// Se o ID da mensagem estiver vazio, gerar um novo
	if message.ID == "" {
		message.ID = uuid.New().String()
	}

	_, err = r.db.Exec(ctx, query,
		message.ID,
		tenantID,
		message.UserID,
		message.Role,
		message.Content,
		message.Timestamp,
	)

	if err != nil {
		return fmt.Errorf("erro ao salvar mensagem: %w", err)
	}

	return nil
}

func (r *ChatRepository) GetUserHistory(ctx context.Context, userID string, limit, offset int) ([]chat.Message, error) {
	// Extrair tenant_id do contexto
	tenantID := ctx.Value("tenant_id").(string)
	if tenantID == "" {
		return nil, fmt.Errorf("tenant_id não encontrado no contexto")
	}

	// Obter o schema do tenant
	var schema string
	err := r.db.QueryRow(ctx, "SELECT schema FROM public.tenants WHERE id = $1", tenantID).Scan(&schema)
	if err != nil {
		return nil, fmt.Errorf("erro ao obter schema do tenant: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT id, role, content, created_at
		FROM %s.chat_history
		WHERE user_id = $1 AND tenant_id = $2
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`, schema)

	rows, err := r.db.Query(ctx, query, userID, tenantID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar histórico: %w", err)
	}
	defer rows.Close()

	var messages []chat.Message
	for rows.Next() {
		var msg chat.Message
		err := rows.Scan(
			&msg.ID,
			&msg.Role,
			&msg.Content,
			&msg.Timestamp,
		)
		if err != nil {
			return nil, fmt.Errorf("erro ao ler mensagem: %w", err)
		}
		msg.UserID = userID
		messages = append(messages, msg)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("erro ao ler linhas: %w", err)
	}

	return messages, nil
}

func (r *ChatRepository) DeleteUserHistory(ctx context.Context, userID string) error {
	// Extrair tenant_id do contexto
	tenantID := ctx.Value("tenant_id").(string)
	if tenantID == "" {
		return fmt.Errorf("tenant_id não encontrado no contexto")
	}

	// Obter o schema do tenant
	var schema string
	err := r.db.QueryRow(ctx, "SELECT schema FROM public.tenants WHERE id = $1", tenantID).Scan(&schema)
	if err != nil {
		return fmt.Errorf("erro ao obter schema do tenant: %w", err)
	}

	query := fmt.Sprintf(`DELETE FROM %s.chat_history WHERE user_id = $1 AND tenant_id = $2`, schema)

	result, err := r.db.Exec(ctx, query, userID, tenantID)
	if err != nil {
		return fmt.Errorf("erro ao deletar histórico: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("nenhuma mensagem encontrada para o usuário")
	}

	return nil
}

func (r *ChatRepository) CountUserMessages(ctx context.Context, userID string) (int, error) {
	// Extrair tenant_id do contexto
	tenantID := ctx.Value("tenant_id").(string)
	if tenantID == "" {
		return 0, fmt.Errorf("tenant_id não encontrado no contexto")
	}

	// Obter o schema do tenant
	var schema string
	err := r.db.QueryRow(ctx, "SELECT schema FROM public.tenants WHERE id = $1", tenantID).Scan(&schema)
	if err != nil {
		return 0, fmt.Errorf("erro ao obter schema do tenant: %w", err)
	}

	query := fmt.Sprintf(`SELECT COUNT(*) FROM %s.chat_history WHERE user_id = $1 AND tenant_id = $2`, schema)

	var count int
	err = r.db.QueryRow(ctx, query, userID, tenantID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("erro ao contar mensagens: %w", err)
	}

	return count, nil
}
