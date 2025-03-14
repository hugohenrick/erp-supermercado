package auth

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/hugohenrick/erp-supermercado/internal/domain/user"
)

// Erros específicos
var (
	ErrInvalidToken  = errors.New("token inválido")
	ErrExpiredToken  = errors.New("token expirado")
	ErrInvalidClaims = errors.New("claims inválidas")
	ErrMissingJWTKey = errors.New("chave secreta JWT não configurada")
)

// JWTClaims representa as claims personalizadas do token JWT
type JWTClaims struct {
	UserID   string `json:"user_id"`
	TenantID string `json:"tenant_id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Role     string `json:"role"`
	BranchID string `json:"branch_id,omitempty"`
	jwt.RegisteredClaims
}

// JWTService implementa serviços relacionados a tokens JWT
type JWTService struct {
	secretKey  []byte
	expiration time.Duration
}

// NewJWTService cria uma nova instância de JWTService
func NewJWTService() (*JWTService, error) {
	secretKey := os.Getenv("JWT_SECRET_KEY")
	if secretKey == "" {
		return nil, ErrMissingJWTKey
	}

	// Duração padrão de 24 horas se não for configurado
	expirationStr := os.Getenv("JWT_EXPIRATION_HOURS")
	expiration := 24 * time.Hour
	if expirationStr != "" {
		expirationHours, err := time.ParseDuration(expirationStr + "h")
		if err == nil {
			expiration = expirationHours
		}
	}

	return &JWTService{
		secretKey:  []byte(secretKey),
		expiration: expiration,
	}, nil
}

// GenerateToken gera um token JWT para o usuário
func (s *JWTService) GenerateToken(u *user.User) (string, error) {
	// Definir o tempo de expiração
	expirationTime := time.Now().Add(s.expiration)

	// Criar as claims
	claims := JWTClaims{
		UserID:   u.ID,
		TenantID: u.TenantID,
		Email:    u.Email,
		Name:     u.Name,
		Role:     string(u.Role),
		BranchID: u.BranchID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "erp-supermercado-api",
			Subject:   u.ID,
		},
	}

	// Criar o token com as claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Assinar o token com a chave secreta
	tokenString, err := token.SignedString(s.secretKey)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ValidateToken valida um token JWT e retorna as claims se for válido
func (s *JWTService) ValidateToken(tokenString string) (*JWTClaims, error) {
	// Definir uma função para analisar as claims
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Verificar o método de assinatura
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return s.secretKey, nil
	})

	if err != nil {
		// Verificar se o erro é de token expirado
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	// Extrair as claims
	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidClaims
	}

	return claims, nil
}

// RefreshToken renova um token JWT
func (s *JWTService) RefreshToken(tokenString string) (string, error) {
	// Validar o token atual
	claims, err := s.ValidateToken(tokenString)
	if err != nil && !errors.Is(err, ErrExpiredToken) {
		// Se o erro não for apenas de expiração, rejeitar a renovação
		return "", err
	}

	// Definir o novo tempo de expiração
	expirationTime := time.Now().Add(s.expiration)
	claims.ExpiresAt = jwt.NewNumericDate(expirationTime)
	claims.IssuedAt = jwt.NewNumericDate(time.Now())
	claims.NotBefore = jwt.NewNumericDate(time.Now())

	// Criar novo token com as claims atualizadas
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Assinar o token com a chave secreta
	newTokenString, err := token.SignedString(s.secretKey)
	if err != nil {
		return "", err
	}

	return newTokenString, nil
}
