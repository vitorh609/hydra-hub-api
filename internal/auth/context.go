package auth

import "context"

type contextKey string

const userContextKey contextKey = "auth_user"

func withUser(ctx context.Context, user SessionUser) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

func UserFromContext(ctx context.Context) (SessionUser, bool) {
	user, ok := ctx.Value(userContextKey).(SessionUser)
	return user, ok
}
