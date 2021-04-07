package api

import (
	"context"

	"github.com/filecoin-project/go-jsonrpc/auth"
)

type FullStruct struct {
	Internal struct {
		AuthNew    func(p0 context.Context, p1 []auth.Permission) ([]byte, error) `perm:"admin"`
		AuthVerify func(p0 context.Context, p1 string) ([]auth.Permission, error) `perm:"read"`

		FuncA func(ctx context.Context, index int) (int, error) `perm:"read"`
	}
}

func (s *FullStruct) AuthNew(p0 context.Context, p1 []auth.Permission) ([]byte, error) {
	return s.Internal.AuthNew(p0, p1)
}

func (s *FullStruct) AuthVerify(p0 context.Context, p1 string) ([]auth.Permission, error) {
	return s.Internal.AuthVerify(p0, p1)
}

func (s *FullStruct) FuncA(ctx context.Context, index int) (int, error) {
	return s.Internal.FuncA(ctx, index)
}

var _ FullApi = new(FullStruct)
