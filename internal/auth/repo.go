package auth

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrInvalidCredentials = errors.New("credenciais inválidas")
	ErrInvalidToken       = errors.New("token inválido")
)

type Repo struct {
	pool *pgxpool.Pool
}

func NewRepo(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

func (r *Repo) FindUserByLogin(ctx context.Context, login string) (AuthUser, error) {
	var user AuthUser

	err := r.pool.QueryRow(ctx, `
		select id, name, email, password_hash
		from users
		where lower(email) = lower($1)
	`, login).Scan(&user.ID, &user.Name, &user.Email, &user.PasswordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return AuthUser{}, ErrInvalidCredentials
	}

	return user, err
}

func (r *Repo) CreateSession(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error {
	_, err := r.pool.Exec(ctx, `
		insert into auth_sessions (user_id, token_hash, expires_at, last_activity_at)
		values ($1, $2, $3, now())
	`, userID, tokenHash, expiresAt)
	return err
}

func (r *Repo) ValidateAndTouchSession(ctx context.Context, tokenHash string, now time.Time, expiresAt time.Time) (SessionUser, time.Time, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return SessionUser{}, time.Time{}, err
	}
	defer tx.Rollback(ctx)

	var user SessionUser
	var currentExpiresAt time.Time

	err = tx.QueryRow(ctx, `
		select u.id, u.name, u.email, s.expires_at
		from auth_sessions s
		join users u on u.id = s.user_id
		where s.token_hash = $1
			and s.revoked_at is null
			and s.expires_at > $2
		for update
	`, tokenHash, now).Scan(&user.ID, &user.Name, &user.Email, &currentExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return SessionUser{}, time.Time{}, ErrInvalidToken
	}
	if err != nil {
		return SessionUser{}, time.Time{}, err
	}

	_, err = tx.Exec(ctx, `
		update auth_sessions
		set last_activity_at = $2,
			expires_at = $3
		where token_hash = $1
	`, tokenHash, now, expiresAt)
	if err != nil {
		return SessionUser{}, time.Time{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return SessionUser{}, time.Time{}, err
	}

	return user, expiresAt, nil
}
