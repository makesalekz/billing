package data

import (
	"context"

	"gitlab.calendaria.team/services/dummy/ent"

	_ "github.com/lib/pq"
)

// DummyRepo
type DummyRepo interface {
	CreateDummy(ctx context.Context, userId int64) (*ent.Dummy, error)
}

type dummyRepo struct {
	db *ent.Client
}

// NewDummyRepo .
func NewDummyRepo(d *Data) DummyRepo {
	return &dummyRepo{
		db: d.db,
	}
}

func (r *dummyRepo) CreateDummy(ctx context.Context, userId int64) (*ent.Dummy, error) {
	return r.db.Dummy.Create().SetUserID(userId).Save(ctx)
}
