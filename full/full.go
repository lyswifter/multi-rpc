package full

import (
	"context"

	"github.com/filecoin-project/go-jsonrpc/auth"
	"github.com/gbrlsnchs/jwt/v3"
	"github.com/lyswifter/api"
	"golang.org/x/xerrors"
)

type jwtPayload struct {
	Allow []auth.Permission
}

type APIAlg jwt.HMACSHA

type FullNodeAPI struct {
	APISecret *APIAlg
}

func (a *FullNodeAPI) AuthVerify(ctx context.Context, token string) ([]auth.Permission, error) {
	var payload jwtPayload
	if _, err := jwt.Verify([]byte(token), (*jwt.HMACSHA)(a.APISecret), &payload); err != nil {
		return nil, xerrors.Errorf("JWT Verification failed: %w", err)
	}

	return payload.Allow, nil
}

func (a *FullNodeAPI) AuthNew(ctx context.Context, perms []auth.Permission) ([]byte, error) {
	p := jwtPayload{
		Allow: perms, // TODO: consider checking validity
	}

	return jwt.Sign(&p, (*jwt.HMACSHA)(a.APISecret))
}

func (f *FullNodeAPI) FuncA(ctx context.Context) error {
	return nil
}

var _ api.FullApi = &FullNodeAPI{}
