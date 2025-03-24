package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hugohenrick/erp-supermercado/internal/domain/user"
	pkgtenant "github.com/hugohenrick/erp-supermercado/pkg/tenant"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Erros específicos do repositório
var (
	ErrUserNotFound       = errors.New("usuário não encontrado")
	ErrUserDuplicateEmail = errors.New("usuário com mesmo email já existe para este tenant")
	ErrUserDatabaseError  = errors.New("erro de banco de dados")
)

// UserRepository implementa a interface user.Repository usando PostgreSQL
type UserRepository struct {
	db *pgxpool.Pool
}

// NewUserRepository cria uma nova instância de UserRepository
func NewUserRepository(db *pgxpool.Pool) user.Repository {
	return &UserRepository{
		db: db,
	}
}

// Create implementa user.Repository.Create
func (r *UserRepository) Create(ctx context.Context, u *user.User) error {
	conn, err := r.db.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("falha ao obter conexão: %w", err)
	}
	defer conn.Release()

	// Primeiro, definir o search_path para public para garantir que acessamos os tenants
	_, err = conn.Exec(ctx, "SET search_path TO public")
	if err != nil {
		return fmt.Errorf("falha ao configurar search_path: %w", err)
	}

	// Obter o schema do tenant a partir do tenant_id
	var schema string
	err = conn.QueryRow(ctx, "SELECT schema FROM public.tenants WHERE id = $1", u.TenantID).Scan(&schema)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errors.New("tenant não encontrado")
		}
		return fmt.Errorf("falha ao obter schema do tenant: %w", err)
	}

	// Debug para verificar o schema recuperado
	fmt.Printf("DEBUG Create - Tenant ID: %s, Schema: %s\n", u.TenantID, schema)

	// Verificar se já existe um usuário com o mesmo email no mesmo tenant
	// Note que esta verificação precisa ocorrer no schema específico do tenant
	var exists bool
	checkQuery := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s.users WHERE tenant_id = $1 AND email = $2)", schema)
	err = conn.QueryRow(ctx, checkQuery, u.TenantID, u.Email).Scan(&exists)
	if err != nil {
		return fmt.Errorf("falha ao verificar existência do usuário: %w", err)
	}

	if exists {
		return ErrUserDuplicateEmail
	}

	// Preparar branch_id - se for string vazia, enviar NULL para o banco
	var branchID interface{} = nil
	if u.BranchID != "" {
		branchID = u.BranchID
	}

	// Inserir o usuário no schema específico do tenant
	query := fmt.Sprintf(`
		INSERT INTO %s.users (
			id, tenant_id, branch_id, name, email, password, role, status, last_login_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)
	`, schema)

	_, err = conn.Exec(ctx, query,
		u.ID,
		u.TenantID,
		branchID,
		u.Name,
		u.Email,
		u.Password,
		string(u.Role),
		string(u.Status),
		u.LastLoginAt,
		u.CreatedAt,
		u.UpdatedAt,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // Unique violation
			return ErrUserDuplicateEmail
		}
		return fmt.Errorf("falha ao inserir usuário: %w", err)
	}

	return nil
}

// FindByID implementa user.Repository.FindByID
func (r *UserRepository) FindByID(ctx context.Context, id string) (*user.User, error) {
	conn, err := r.db.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("falha ao obter conexão: %w", err)
	}
	defer conn.Release()

	// Obter tenant ID do contexto
	tenantID := pkgtenant.GetTenantID(ctx)
	if tenantID == "" {
		return nil, errors.New("tenant ID não encontrado no contexto")
	}

	query := `
		SELECT 
			id, tenant_id, branch_id, name, email, password, role, status, last_login_at, created_at, updated_at
		FROM 
			users
		WHERE 
			id = $1 AND tenant_id = $2
	`

	u := &user.User{}
	var role, status string
	var lastLoginTime pgtype.Timestamp

	err = conn.QueryRow(ctx, query, id, tenantID).Scan(
		&u.ID,
		&u.TenantID,
		&u.BranchID,
		&u.Name,
		&u.Email,
		&u.Password,
		&role,
		&status,
		&lastLoginTime,
		&u.CreatedAt,
		&u.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("falha ao buscar usuário: %w", err)
	}

	u.Role = user.Role(role)
	u.Status = user.Status(status)
	if lastLoginTime.Valid {
		u.LastLoginAt = lastLoginTime.Time
	}

	return u, nil
}

// FindByEmail implementa user.Repository.FindByEmail
func (r *UserRepository) FindByEmail(ctx context.Context, tenantID, email string) (*user.User, error) {
	conn, err := r.db.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("falha ao obter conexão: %w", err)
	}
	defer conn.Release()

	// Obter o schema do tenant a partir do tenant_id
	var schema string
	err = conn.QueryRow(ctx, "SELECT schema FROM public.tenants WHERE id = $1", tenantID).Scan(&schema)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("tenant não encontrado")
		}
		return nil, fmt.Errorf("falha ao obter schema do tenant: %w", err)
	}

	// Construir a query usando o schema específico do tenant
	query := fmt.Sprintf(`
		SELECT 
			id, tenant_id, branch_id, name, email, password, role, status, last_login_at, created_at, updated_at
		FROM 
			%s.users
		WHERE 
			tenant_id = $1 AND email = $2
	`, schema)

	u := &user.User{}
	var role, status string
	var lastLoginTime pgtype.Timestamp

	err = conn.QueryRow(ctx, query, tenantID, email).Scan(
		&u.ID,
		&u.TenantID,
		&u.BranchID,
		&u.Name,
		&u.Email,
		&u.Password,
		&role,
		&status,
		&lastLoginTime,
		&u.CreatedAt,
		&u.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("falha ao buscar usuário por email: %w", err)
	}

	u.Role = user.Role(role)
	u.Status = user.Status(status)
	if lastLoginTime.Valid {
		u.LastLoginAt = lastLoginTime.Time
	}

	return u, nil
}

// FindByEmailAcrossTenants implementa user.Repository.FindByEmailAcrossTenants
func (r *UserRepository) FindByEmailAcrossTenants(ctx context.Context, email string) (*user.User, error) {
	conn, err := r.db.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("falha ao obter conexão: %w", err)
	}
	defer conn.Release()

	// Primeiro, definir o search_path para public para garantir que acessamos os tenants
	_, err = conn.Exec(ctx, "SET search_path TO public")
	if err != nil {
		return nil, fmt.Errorf("falha ao configurar search_path: %w", err)
	}

	// Obter todos os schemas de tenants
	rows, err := conn.Query(ctx, "SELECT id, schema FROM public.tenants")
	if err != nil {
		return nil, fmt.Errorf("falha ao obter schemas dos tenants: %w", err)
	}
	defer rows.Close()

	// Lista para armazenar os schemas e IDs dos tenants
	type tenantInfo struct {
		ID     string
		Schema string
	}
	var tenants []tenantInfo

	// Processar os resultados
	for rows.Next() {
		var t tenantInfo
		if err := rows.Scan(&t.ID, &t.Schema); err != nil {
			return nil, fmt.Errorf("falha ao ler dados do tenant: %w", err)
		}
		tenants = append(tenants, t)

		// Debug: printar o ID e schema do tenant
		fmt.Printf("DEBUG - Tenant ID: %s, Schema: %s\n", t.ID, t.Schema)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erro ao iterar resultados dos tenants: %w", err)
	}

	// Verificar se temos tenants para procurar
	if len(tenants) == 0 {
		return nil, ErrUserNotFound
	}

	// Procurar o usuário em cada schema de tenant
	for _, tenant := range tenants {
		// Garantir que o schema é válido
		if tenant.Schema == "" {
			fmt.Printf("DEBUG - Ignorando tenant %s com schema vazio\n", tenant.ID)
			continue
		}

		// Query para buscar o usuário no schema específico
		query := fmt.Sprintf(`
			SELECT 
				id, tenant_id, branch_id, name, email, password, role, status, last_login_at, created_at, updated_at
			FROM 
				%s.users
			WHERE 
				email = $1
			LIMIT 1
		`, tenant.Schema)

		// Debug: printar a query completa
		fmt.Printf("DEBUG - Executing query: %s with email: %s\n", query, email)

		u := &user.User{}
		var role, status string
		var lastLoginTime pgtype.Timestamp
		var branchID pgtype.Text // Usar pgtype.Text para lidar com NULL

		err = conn.QueryRow(ctx, query, email).Scan(
			&u.ID,
			&u.TenantID,
			&branchID,
			&u.Name,
			&u.Email,
			&u.Password,
			&role,
			&status,
			&lastLoginTime,
			&u.CreatedAt,
			&u.UpdatedAt,
		)

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				// Se não encontrou neste tenant, continuar procurando no próximo
				fmt.Printf("DEBUG - Usuário não encontrado no tenant %s\n", tenant.ID)
				continue
			}
			// Se foi outro erro, logar para debug mas continuar tentando outros tenants
			fmt.Printf("DEBUG - Erro ao buscar usuário em %s: %v\n", tenant.Schema, err)
			continue
		}

		// Encontrou o usuário, converter os campos
		if branchID.Valid {
			u.BranchID = branchID.String
		} else {
			u.BranchID = "" // String vazia se for NULL
		}

		u.Role = user.Role(role)
		u.Status = user.Status(status)
		if lastLoginTime.Valid {
			u.LastLoginAt = lastLoginTime.Time
		}

		fmt.Printf("DEBUG - Usuário encontrado no tenant %s\n", tenant.ID)

		// Retornar o usuário encontrado
		return u, nil
	}

	// Se chegou aqui, o usuário não foi encontrado em nenhum tenant
	return nil, ErrUserNotFound
}

// FindByBranch implementa user.Repository.FindByBranch
func (r *UserRepository) FindByBranch(ctx context.Context, branchID string, limit, offset int) ([]*user.User, error) {
	conn, err := r.db.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("falha ao obter conexão: %w", err)
	}
	defer conn.Release()

	// Obter tenant ID do contexto
	tenantID := pkgtenant.GetTenantID(ctx)
	if tenantID == "" {
		return nil, errors.New("tenant ID não encontrado no contexto")
	}

	query := `
		SELECT 
			id, tenant_id, branch_id, name, email, password, role, status, last_login_at, created_at, updated_at
		FROM 
			users
		WHERE 
			tenant_id = $1 AND branch_id = $2
		ORDER BY 
			name ASC
		LIMIT $3 OFFSET $4
	`

	rows, err := conn.Query(ctx, query, tenantID, branchID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("falha ao listar usuários por filial: %w", err)
	}
	defer rows.Close()

	return r.scanUserRows(rows)
}

// List implementa user.Repository.List
func (r *UserRepository) List(ctx context.Context, tenantID string, limit, offset int) ([]*user.User, error) {
	conn, err := r.db.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("falha ao obter conexão: %w", err)
	}
	defer conn.Release()

	query := `
		SELECT 
			id, tenant_id, branch_id, name, email, password, role, status, last_login_at, created_at, updated_at
		FROM 
			users
		WHERE 
			tenant_id = $1
		ORDER BY 
			name ASC
		LIMIT $2 OFFSET $3
	`

	rows, err := conn.Query(ctx, query, tenantID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("falha ao listar usuários: %w", err)
	}
	defer rows.Close()

	return r.scanUserRows(rows)
}

// Update implementa user.Repository.Update
func (r *UserRepository) Update(ctx context.Context, u *user.User) error {
	conn, err := r.db.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("falha ao obter conexão: %w", err)
	}
	defer conn.Release()

	// Obter tenant ID do contexto
	tenantID := pkgtenant.GetTenantID(ctx)
	if tenantID == "" {
		return errors.New("tenant ID não encontrado no contexto")
	}

	// Verificar se o usuário pertence ao tenant correto
	if u.TenantID != tenantID {
		return errors.New("usuário não pertence ao tenant atual")
	}

	// Verificar se já existe um usuário com o mesmo email no mesmo tenant (exceto este)
	var exists bool
	err = conn.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE tenant_id = $1 AND email = $2 AND id != $3)", u.TenantID, u.Email, u.ID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("falha ao verificar existência do usuário: %w", err)
	}

	if exists {
		return ErrUserDuplicateEmail
	}

	query := `
		UPDATE users
		SET 
			branch_id = $1,
			name = $2,
			email = $3,
			role = $4,
			status = $5,
			updated_at = $6
		WHERE 
			id = $7 AND tenant_id = $8
	`

	result, err := conn.Exec(ctx, query,
		u.BranchID,
		u.Name,
		u.Email,
		string(u.Role),
		string(u.Status),
		time.Now(),
		u.ID,
		u.TenantID,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // Unique violation
			return ErrUserDuplicateEmail
		}
		return fmt.Errorf("falha ao atualizar usuário: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}

	return nil
}

// Delete implementa user.Repository.Delete
func (r *UserRepository) Delete(ctx context.Context, id string) error {
	conn, err := r.db.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("falha ao obter conexão: %w", err)
	}
	defer conn.Release()

	// Obter tenant ID do contexto
	tenantID := pkgtenant.GetTenantID(ctx)
	if tenantID == "" {
		return errors.New("tenant ID não encontrado no contexto")
	}

	result, err := conn.Exec(ctx, "DELETE FROM users WHERE id = $1 AND tenant_id = $2", id, tenantID)
	if err != nil {
		return fmt.Errorf("falha ao excluir usuário: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}

	return nil
}

// UpdateStatus implementa user.Repository.UpdateStatus
func (r *UserRepository) UpdateStatus(ctx context.Context, id string, status user.Status) error {
	conn, err := r.db.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("falha ao obter conexão: %w", err)
	}
	defer conn.Release()

	// Obter tenant ID do contexto
	tenantID := pkgtenant.GetTenantID(ctx)
	if tenantID == "" {
		return errors.New("tenant ID não encontrado no contexto")
	}

	query := `
		UPDATE users
		SET 
			status = $1,
			updated_at = $2
		WHERE 
			id = $3 AND tenant_id = $4
	`

	result, err := conn.Exec(ctx, query, string(status), time.Now(), id, tenantID)
	if err != nil {
		return fmt.Errorf("falha ao atualizar status do usuário: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}

	return nil
}

// UpdatePassword implementa user.Repository.UpdatePassword
func (r *UserRepository) UpdatePassword(ctx context.Context, id, hashedPassword string) error {
	conn, err := r.db.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("falha ao obter conexão: %w", err)
	}
	defer conn.Release()

	// Obter tenant ID do contexto
	tenantID := pkgtenant.GetTenantID(ctx)
	if tenantID == "" {
		return errors.New("tenant ID não encontrado no contexto")
	}

	query := `
		UPDATE users
		SET 
			password = $1,
			updated_at = $2
		WHERE 
			id = $3 AND tenant_id = $4
	`

	result, err := conn.Exec(ctx, query, hashedPassword, time.Now(), id, tenantID)
	if err != nil {
		return fmt.Errorf("falha ao atualizar senha do usuário: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}

	return nil
}

// UpdateLastLogin implementa user.Repository.UpdateLastLogin
func (r *UserRepository) UpdateLastLogin(ctx context.Context, id string) error {
	conn, err := r.db.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("falha ao obter conexão: %w", err)
	}
	defer conn.Release()

	// Obter tenant ID do contexto, se disponível
	tenantID := pkgtenant.GetTenantID(ctx)

	var query string
	var args []interface{}

	if tenantID != "" {
		// Se tiver tenant_id no contexto, usar na query
		query = `
			UPDATE users
			SET 
				last_login_at = $1
			WHERE 
				id = $2 AND tenant_id = $3
		`
		args = []interface{}{time.Now(), id, tenantID}
	} else {
		// Se não tiver tenant_id no contexto, buscar apenas pelo ID
		query = `
			UPDATE users
			SET 
				last_login_at = $1
			WHERE 
				id = $2
		`
		args = []interface{}{time.Now(), id}
	}

	result, err := conn.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("falha ao atualizar último login do usuário: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}

	return nil
}

// CountByTenant implementa user.Repository.CountByTenant
func (r *UserRepository) CountByTenant(ctx context.Context, tenantID string) (int, error) {
	conn, err := r.db.Acquire(ctx)
	if err != nil {
		return 0, fmt.Errorf("falha ao obter conexão: %w", err)
	}
	defer conn.Release()

	// Primeiro, definir o search_path para public para garantir que acessamos os tenants
	_, err = conn.Exec(ctx, "SET search_path TO public")
	if err != nil {
		return 0, fmt.Errorf("falha ao configurar search_path: %w", err)
	}

	// Obter o schema do tenant a partir do tenant_id
	var schema string
	err = conn.QueryRow(ctx, "SELECT schema FROM public.tenants WHERE id = $1", tenantID).Scan(&schema)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, errors.New("tenant não encontrado")
		}
		return 0, fmt.Errorf("falha ao obter schema do tenant: %w", err)
	}

	// Debug para verificar o schema recuperado
	fmt.Printf("DEBUG CountByTenant - Tenant ID: %s, Schema: %s\n", tenantID, schema)

	// Contar usuários no schema específico do tenant
	var count int
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s.users WHERE tenant_id = $1", schema)
	err = conn.QueryRow(ctx, query, tenantID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("falha ao contar usuários: %w", err)
	}

	return count, nil
}

// CountByBranch implementa user.Repository.CountByBranch
func (r *UserRepository) CountByBranch(ctx context.Context, branchID string) (int, error) {
	conn, err := r.db.Acquire(ctx)
	if err != nil {
		return 0, fmt.Errorf("falha ao obter conexão: %w", err)
	}
	defer conn.Release()

	// Obter tenant ID do contexto
	tenantID := pkgtenant.GetTenantID(ctx)
	if tenantID == "" {
		return 0, errors.New("tenant ID não encontrado no contexto")
	}

	var count int
	err = conn.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE tenant_id = $1 AND branch_id = $2", tenantID, branchID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("falha ao contar usuários da filial: %w", err)
	}

	return count, nil
}

// Exists implementa user.Repository.Exists
func (r *UserRepository) Exists(ctx context.Context, id string) (bool, error) {
	conn, err := r.db.Acquire(ctx)
	if err != nil {
		return false, fmt.Errorf("falha ao obter conexão: %w", err)
	}
	defer conn.Release()

	// Obter tenant ID do contexto
	tenantID := pkgtenant.GetTenantID(ctx)
	if tenantID == "" {
		return false, errors.New("tenant ID não encontrado no contexto")
	}

	var exists bool
	err = conn.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE id = $1 AND tenant_id = $2)", id, tenantID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("falha ao verificar existência do usuário: %w", err)
	}

	return exists, nil
}

// scanUserRows é um método auxiliar para processar resultados de consultas que retornam múltiplos usuários
func (r *UserRepository) scanUserRows(rows pgx.Rows) ([]*user.User, error) {
	var users []*user.User

	for rows.Next() {
		u := &user.User{}
		var role, status string
		var lastLoginTime pgtype.Timestamp

		err := rows.Scan(
			&u.ID,
			&u.TenantID,
			&u.BranchID,
			&u.Name,
			&u.Email,
			&u.Password,
			&role,
			&status,
			&lastLoginTime,
			&u.CreatedAt,
			&u.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("falha ao ler usuário: %w", err)
		}

		u.Role = user.Role(role)
		u.Status = user.Status(status)
		if lastLoginTime.Valid {
			u.LastLoginAt = lastLoginTime.Time
		}

		users = append(users, u)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erro ao iterar resultados: %w", err)
	}

	return users, nil
}

// TenantExists implementa user.Repository.TenantExists
func (r *UserRepository) TenantExists(ctx context.Context, tenantID string) (bool, error) {
	conn, err := r.db.Acquire(ctx)
	if err != nil {
		return false, fmt.Errorf("falha ao obter conexão: %w", err)
	}
	defer conn.Release()

	// Primeiro, configurar o search_path para garantir que estamos consultando o schema public
	_, err = conn.Exec(ctx, "SET search_path TO public")
	if err != nil {
		return false, fmt.Errorf("falha ao configurar search_path: %w", err)
	}

	// Verificar se o tenant existe especificando explicitamente o schema public
	var exists bool
	err = conn.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM public.tenants WHERE id = $1)", tenantID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("falha ao verificar existência do tenant: %w", err)
	}

	return exists, nil
}
