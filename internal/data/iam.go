package data

import (
	"context"

	v1 "gitlab.calendaria.team/services/finance/billing/api/billing/v1"
	"gitlab.calendaria.team/services/finance/billing/internal/conf"
	iam_v1 "gitlab.calendaria.team/services/iam/api/iam/v1"
	"gitlab.calendaria.team/services/utils/v2/dialer"
)

type IamRemote struct {
	dialer dialer.IDialer
}

func NewIamRemote(
	conf *conf.Bootstrap,
	dm dialer.IDialerManager,
) (*IamRemote, func(), error) {
	dialer, err := dm.NewServiceDialer("iam", conf.GetDiscovery().GetIam())
	if err != nil {
		return nil, nil, err
	}

	cleanup := func() {
		dialer.Close()
	}

	return &IamRemote{
		dialer: dialer,
	}, cleanup, nil
}

func (r *IamRemote) getUsersClient(ctx context.Context) (iam_v1.UsersClient, error) {
	conn, err := r.dialer.Connect(ctx)
	if err != nil {
		return nil, v1.ErrorGrpcConnection("can't connect to iam: %s", err.Error())
	}

	return iam_v1.NewUsersClient(conn), nil
}

// --------------------------- Users ---------------------------

func (r *IamRemote) GetUserFull(ctx context.Context, req *iam_v1.GetUserRequest) (*iam_v1.UserFullReply, error) {
	userClient, err := r.getUsersClient(ctx)
	if err != nil {
		return nil, err
	}

	user, err := userClient.GetUserFull(ctx, req)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (r *IamRemote) GetUser(ctx context.Context, req *iam_v1.GetUserRequest) (*iam_v1.UserReply, error) {
	userClient, err := r.getUsersClient(ctx)
	if err != nil {
		return nil, err
	}

	user, err := userClient.GetUser(ctx, req)
	if err != nil {
		return nil, err
	}

	return user, nil
}
