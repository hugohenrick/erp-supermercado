package controller

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hugohenrick/erp-supermercado/internal/adapter/api/dto"
	"github.com/hugohenrick/erp-supermercado/internal/adapter/repository"
	"github.com/hugohenrick/erp-supermercado/internal/domain/user"
	"github.com/hugohenrick/erp-supermercado/pkg/auth"
)

// AuthController gerencia as requisições relacionadas à autenticação
type AuthController struct {
	userRepository user.Repository
}

// NewAuthController cria uma nova instância de AuthController
func NewAuthController(userRepository user.Repository) *AuthController {
	return &AuthController{
		userRepository: userRepository,
	}
}

// Login autentica um usuário e retorna um token JWT
// @Summary Autentica um usuário
// @Description Verifica as credenciais do usuário e retorna um token JWT
// @Tags auth
// @Accept json
// @Produce json
// @Param login body dto.LoginRequest true "Credenciais de login"
// @Success 200 {object} dto.LoginResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /auth/login [post]
func (c *AuthController) Login(ctx *gin.Context) {
	var request dto.LoginRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "Requisição inválida", err.Error()))
		return
	}

	// Buscar o usuário pelo email
	u, err := c.userRepository.FindByEmail(ctx, request.TenantID, request.Email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			ctx.JSON(http.StatusUnauthorized, dto.NewErrorResponse(http.StatusUnauthorized, "Credenciais inválidas", "Email ou senha incorretos"))
			return
		}
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "Erro ao autenticar usuário", err.Error()))
		return
	}

	// Verificar se o usuário está ativo
	if !u.IsActive() {
		ctx.JSON(http.StatusForbidden, dto.NewErrorResponse(http.StatusForbidden, "Usuário inativo", "Sua conta está desativada ou bloqueada"))
		return
	}

	// Verificar a senha
	if !u.CheckPassword(request.Password) {
		ctx.JSON(http.StatusUnauthorized, dto.NewErrorResponse(http.StatusUnauthorized, "Credenciais inválidas", "Email ou senha incorretos"))
		return
	}

	// Gerar o token JWT
	jwtService, err := auth.NewJWTService()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "Erro ao configurar autenticação", err.Error()))
		return
	}

	token, err := jwtService.GenerateToken(u)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "Erro ao gerar token", err.Error()))
		return
	}

	// Gerar refresh token (usando o mesmo token por simplicidade)
	refreshToken := token

	// Obter duração do token (24h por padrão)
	expirationTime := time.Now().Add(24 * time.Hour)

	// Atualizar o último login do usuário
	err = c.userRepository.UpdateLastLogin(ctx, u.ID)
	if err != nil {
		// Apenas logar o erro, não impedir o login
	}

	// Construir a resposta
	response := dto.LoginResponse{
		User:         dto.ToUserResponse(u),
		AccessToken:  token,
		RefreshToken: refreshToken,
		ExpiresAt:    expirationTime,
	}

	ctx.JSON(http.StatusOK, response)
}

// RefreshToken renova um token JWT
// @Summary Renova um token JWT
// @Description Renova um token JWT existente
// @Tags auth
// @Accept json
// @Produce json
// @Param refresh body dto.RefreshTokenRequest true "Token a ser renovado"
// @Success 200 {object} dto.LoginResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /auth/refresh [post]
func (c *AuthController) RefreshToken(ctx *gin.Context) {
	var request dto.RefreshTokenRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "Requisição inválida", err.Error()))
		return
	}

	// Inicializar o serviço JWT
	jwtService, err := auth.NewJWTService()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "Erro ao configurar autenticação", err.Error()))
		return
	}

	// Renovar o token
	newToken, err := jwtService.RefreshToken(request.RefreshToken)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidToken) {
			ctx.JSON(http.StatusUnauthorized, dto.NewErrorResponse(http.StatusUnauthorized, "Token inválido", err.Error()))
			return
		}
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "Erro ao renovar token", err.Error()))
		return
	}

	// Validar o novo token para obter as claims
	claims, err := jwtService.ValidateToken(newToken)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "Erro ao validar novo token", err.Error()))
		return
	}

	// Buscar o usuário para ter informações atualizadas
	u, err := c.userRepository.FindByID(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			ctx.JSON(http.StatusUnauthorized, dto.NewErrorResponse(http.StatusUnauthorized, "Usuário não encontrado", "O usuário associado ao token não existe mais"))
			return
		}
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "Erro ao buscar usuário", err.Error()))
		return
	}

	// Obter duração do token (24h por padrão)
	expirationTime := time.Now().Add(24 * time.Hour)

	response := dto.LoginResponse{
		User:         dto.ToUserResponse(u),
		AccessToken:  newToken,
		RefreshToken: newToken, // Usar o mesmo token para simplicidade
		ExpiresAt:    expirationTime,
	}

	ctx.JSON(http.StatusOK, response)
}

// Me retorna informações do usuário atual
// @Summary Retorna informações do usuário atual
// @Description Retorna informações do usuário autenticado
// @Tags auth
// @Produce json
// @Security Bearer
// @Success 200 {object} dto.UserResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /auth/me [get]
func (c *AuthController) Me(ctx *gin.Context) {
	// Obter o ID do usuário do contexto (definido pelo middleware de autenticação)
	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, dto.NewErrorResponse(http.StatusUnauthorized, "Não autenticado", ""))
		return
	}

	// Converter userID para string
	userIDStr, ok := userID.(string)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "Erro interno", "Falha ao obter ID do usuário"))
		return
	}

	// Buscar o usuário no repositório
	u, err := c.userRepository.FindByID(ctx, userIDStr)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			ctx.JSON(http.StatusUnauthorized, dto.NewErrorResponse(http.StatusUnauthorized, "Usuário não encontrado", ""))
			return
		}
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "Erro ao buscar usuário", err.Error()))
		return
	}

	ctx.JSON(http.StatusOK, dto.ToUserResponse(u))
}
