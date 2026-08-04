package api

import (
	"context"

	"era/services/platform/drive"
)

type ctxKey int

const principalKey ctxKey = 1

func withPrincipal(ctx context.Context, p drive.Principal) context.Context {
	return context.WithValue(ctx, principalKey, p)
}

func principalFrom(ctx context.Context) drive.Principal {
	p, _ := ctx.Value(principalKey).(drive.Principal)
	return p
}
