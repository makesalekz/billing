package data

import (
	"context"
	"fmt"
	"strconv"

	"github.com/go-kratos/kratos/v2/middleware/auth/jwt"
	jwtv4 "github.com/golang-jwt/jwt/v4"
)

type JwtProcessor struct {
	jwtSecret []byte
}

// NewJwtProcessor .
func NewJwtProcessor(c *Config) (*JwtProcessor, error) {
	secret, err := c.ReadGlobalSecretsFor(context.Background(), "jwt")
	if err != nil {
		return nil, fmt.Errorf("JWT_SECRET not found, error: %w", err)
	}

	return &JwtProcessor{
		jwtSecret: []byte(secret["data"].(string)),
	}, nil
}

func (j *JwtProcessor) GetSecret() []byte {
	return j.jwtSecret
}

func (j *JwtProcessor) GetUserIdFromContext(ctx context.Context) (int64, bool) {
	claims := j.GetClaimsFromContext(ctx)
	if claims == nil {
		return 0, false
	}

	userId, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil {
		return 0, false
	}

	return userId, true
}

func (j *JwtProcessor) GetClaimsFromContext(ctx context.Context) *jwtv4.RegisteredClaims {
	token, ok := jwt.FromContext(ctx)
	if !ok {
		return nil
	}

	claims, ok := token.(*jwtv4.RegisteredClaims)
	if !ok {
		return nil
	}

	return claims
}
