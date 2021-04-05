package full

import (
	"context"
	"fmt"

	"github.com/filecoin-project/go-jsonrpc/auth"
	"github.com/lyswifter/api"
	"github.com/lyswifter/jwt"
)

type FullNodeAPI struct {
	APISecret *jwt.APIAlg
}

func (a *FullNodeAPI) AuthVerify(ctx context.Context, token string) ([]auth.Permission, error) {
	return jwt.AuthVerify(ctx, token, a.APISecret)
}

func (a *FullNodeAPI) AuthNew(ctx context.Context, perms []auth.Permission) ([]byte, error) {
	return jwt.AuthNew(ctx, perms, a.APISecret)
}

func (f *FullNodeAPI) FuncA(ctx context.Context) error {
	fmt.Println("funcA")
	return nil
}

var _ api.FullApi = &FullNodeAPI{}
