package api

import (
	"context"

	"github.com/filecoin-project/go-jsonrpc/auth"
)

type FullStruct struct {
	AuthNewInner    func(p0 context.Context, p1 []auth.Permission) ([]byte, error) `perm:"admin"`
	AuthVerifyInner func(p0 context.Context, p1 string) ([]auth.Permission, error) `perm:"read"`

	FuncInner func(ctx context.Context) error `perm:"read"`
}

func (s *FullStruct) AuthNew(p0 context.Context, p1 []auth.Permission) ([]byte, error) {
	return s.AuthNewInner(p0, p1)
}

func (s *FullStruct) AuthVerify(p0 context.Context, p1 string) ([]auth.Permission, error) {
	return s.AuthVerifyInner(p0, p1)
}

func (s *FullStruct) FuncA(ctx context.Context) error {
	return s.FuncInner(ctx)
}

var _ FullApi = new(FullStruct)
