package jwt

import (
	"context"

	"github.com/filecoin-project/go-jsonrpc/auth"
	"github.com/gbrlsnchs/jwt/v3"
	"golang.org/x/xerrors"
)

type jwtPayload struct {
	Allow []auth.Permission
}

type APIAlg jwt.HMACSHA

func AuthVerify(ctx context.Context, token string, alg *APIAlg) ([]auth.Permission, error) {
	var payload jwtPayload
	if _, err := jwt.Verify([]byte(token), (*jwt.HMACSHA)(alg), &payload); err != nil {
		return nil, xerrors.Errorf("JWT Verification failed: %w", err)
	}

	return payload.Allow, nil
}

func AuthNew(ctx context.Context, perms []auth.Permission, alg *APIAlg) ([]byte, error) {
	p := jwtPayload{
		Allow: perms, // TODO: consider checking validity
	}

	return jwt.Sign(&p, (*jwt.HMACSHA)(alg))
}
