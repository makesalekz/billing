package data

import (
	"context"

	consul "github.com/go-kratos/consul/registry"
	"github.com/go-kratos/kratos/v2/middleware/auth/jwt"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	jwtv4 "github.com/golang-jwt/jwt/v4"
	"gitlab.calendaria.team/services/dummy/internal/conf"
	users_v1 "gitlab.calendaria.team/services/dummy/third_party/api/users/v1"
)

type Dialer struct {
	conf      *conf.Bootstrap
	discovery *consul.Registry
	jwt       *JwtProcessor
}

// NewJwtProcessor .
func NewDialer(c *Config, jwt *JwtProcessor) (*Dialer, error) {
	return &Dialer{
		conf:      c.Bootstrap,
		discovery: c.GetRegistry(),
		jwt:       jwt,
	}, nil
}

func (d *Dialer) Users(ctx context.Context) (users_v1.UsersClient, error) {
	conn, err := grpc.DialInsecure(
		ctx,
		grpc.WithEndpoint(d.conf.Discovery.Iam),
		grpc.WithDiscovery(d.discovery),
		grpc.WithTimeout(d.conf.Discovery.IamTimeout.AsDuration()),
		grpc.WithMiddleware(
			jwt.Client(func(token *jwtv4.Token) (interface{}, error) {
				return d.jwt.GetSecret(), nil
			}, jwt.WithSigningMethod(jwtv4.SigningMethodHS256), jwt.WithClaims(func() jwtv4.Claims {
				return d.jwt.GetClaimsFromContext(ctx)
			})),
		),
	)
	if err != nil {
		return nil, err
	}
	return users_v1.NewUsersClient(conn), nil
}
