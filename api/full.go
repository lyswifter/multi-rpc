package api

import (
	"context"

	"github.com/filecoin-project/go-jsonrpc/auth"
)

type FullApi interface {
	AuthVerify(ctx context.Context, token string) ([]auth.Permission, error) //perm:read
	AuthNew(ctx context.Context, perms []auth.Permission) ([]byte, error)    //perm:admin

	FuncA(ctx context.Context, index int) (int, error)
}
