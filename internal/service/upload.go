package service

import (
	"context"

	v1 "media/api/upload/v1"
	"media/internal/biz"
	"media/internal/data"

	"github.com/go-kratos/kratos/v2/log"
)

type UploadService struct {
	v1.UnimplementedUploadServer

	log *log.Helper
	jwt *data.JwtProcessor
	uc  *biz.MediaUsecase
}

func NewUploadService(logger log.Logger, jwt *data.JwtProcessor, uc *biz.MediaUsecase) *UploadService {
	return &UploadService{
		log: log.NewHelper(logger),
		jwt: jwt,
		uc:  uc,
	}
}

func (s *UploadService) UploadMedia(ctx context.Context, req *v1.UploadMediaRequest) (*v1.UploadMediaReply, error) {
	userId, ok := s.jwt.GetUserIdFromContext(ctx)
	if !ok {
		return nil, v1.ErrorUnauthorized("Unauthorized")
	}

	url, err := s.uc.UploadMedia(ctx, userId, req.Content)
	if err != nil {
		return nil, err
	}

	return &v1.UploadMediaReply{
		Url: url,
	}, nil
}

func (s *UploadService) UploadAvatar(ctx context.Context, req *v1.UploadMediaRequest) (*v1.UploadMediaReply, error) {
	userId, ok := s.jwt.GetUserIdFromContext(ctx)
	if !ok {
		return nil, v1.ErrorUnauthorized("Unauthorized")
	}

	url, err := s.uc.UploadAvatar(ctx, userId, req.Content)
	if err != nil {
		return nil, err
	}

	return &v1.UploadMediaReply{
		Url: url,
	}, nil
}
