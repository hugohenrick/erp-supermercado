package jwt

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	// ErrInvalidToken é retornado quando o token é inválido
	ErrInvalidToken = errors.New("token inválido")
	// ErrExpiredToken é retornado quando o token está expirado
	ErrExpiredToken = errors.New("token expirado")
)

// Claims representa as claims do token JWT
type Claims struct {
	UserID    string `json:"user_id"`
	TenantID  string `json:"tenant_id"`
	BranchID  string `json:"branch_id"`
	ExpiresAt int64  `json:"exp"`
	IssuedAt  int64  `json:"iat"`
}

// GetExpirationTime implementa jwt.Claims
func (c Claims) GetExpirationTime() (*jwt.NumericDate, error) {
	if c.ExpiresAt == 0 {
		return nil, nil
	}
	return jwt.NewNumericDate(time.Unix(c.ExpiresAt, 0)), nil
}

// GetIssuedAt implementa jwt.Claims
func (c Claims) GetIssuedAt() (*jwt.NumericDate, error) {
	if c.IssuedAt == 0 {
		return nil, nil
	}
	return jwt.NewNumericDate(time.Unix(c.IssuedAt, 0)), nil
}

// GetNotBefore implementa jwt.Claims
func (c Claims) GetNotBefore() (*jwt.NumericDate, error) {
	return nil, nil
}

// GetIssuer implementa jwt.Claims
func (c Claims) GetIssuer() (string, error) {
	return "", nil
}

// GetSubject implementa jwt.Claims
func (c Claims) GetSubject() (string, error) {
	return "", nil
}

// GetAudience implementa jwt.Claims
func (c Claims) GetAudience() (jwt.ClaimStrings, error) {
	return nil, nil
}

// Valid implementa jwt.Claims
func (c Claims) Valid() error {
	if c.ExpiresAt < time.Now().Unix() {
		return ErrExpiredToken
	}
	return nil
}

// GenerateToken gera um novo token JWT
func GenerateToken(userID, tenantID, branchID string, expiresIn time.Duration) (string, error) {
	claims := Claims{
		UserID:    userID,
		TenantID:  tenantID,
		BranchID:  branchID,
		ExpiresAt: time.Now().Add(expiresIn).Unix(),
		IssuedAt:  time.Now().Unix(),
	}

	secretKey := os.Getenv("JWT_SECRET")
	if secretKey == "" {
		return "", errors.New("chave secreta JWT não configurada")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secretKey))
}

// ValidateToken valida um token JWT
func ValidateToken(tokenString string) (*Claims, error) {
	secretKey := os.Getenv("JWT_SECRET")
	if secretKey == "" {
		return nil, errors.New("chave secreta JWT não configurada")
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(secretKey), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// RefreshToken gera um novo token JWT com base em um token existente
func RefreshToken(tokenString string) (string, error) {
	claims, err := ValidateToken(tokenString)
	if err != nil && !errors.Is(err, ErrExpiredToken) {
		return "", err
	}

	return GenerateToken(claims.UserID, claims.TenantID, claims.BranchID, 24*time.Hour)
}
