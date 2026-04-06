package clickup

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repo struct {
	pool *pgxpool.Pool
}

type Repository interface {
	EnsureSchema(ctx context.Context) error
	UpsertConnection(ctx context.Context, connection Connection) (Connection, error)
	GetConnectionByUserID(ctx context.Context, userID string) (Connection, error)
	MarkConnectionHealth(ctx context.Context, userID string, status string, lastError *string, checkedAt time.Time) error
}

func NewRepo(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

func (r *Repo) EnsureSchema(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `
		create table if not exists clickup_connections (
			user_id uuid primary key references users(id) on delete cascade,
			token_ciphertext text not null,
			token_key_version text not null,
			default_workspace_id text null,
			default_workspace_name text null,
			status text not null default 'connected',
			last_checked_at timestamptz null,
			last_error text null,
			created_at timestamptz not null default now(),
			updated_at timestamptz not null default now()
		)
	`)
	return err
}

func (r *Repo) UpsertConnection(ctx context.Context, connection Connection) (Connection, error) {
	var out Connection

	err := r.pool.QueryRow(ctx, `
		insert into clickup_connections (
			user_id,
			token_ciphertext,
			token_key_version,
			default_workspace_id,
			default_workspace_name,
			status,
			last_checked_at,
			last_error,
			updated_at
		)
		values ($1, $2, $3, $4, $5, $6, $7, $8, now())
		on conflict (user_id) do update
		set
			token_ciphertext = excluded.token_ciphertext,
			token_key_version = excluded.token_key_version,
			default_workspace_id = excluded.default_workspace_id,
			default_workspace_name = excluded.default_workspace_name,
			status = excluded.status,
			last_checked_at = excluded.last_checked_at,
			last_error = excluded.last_error,
			updated_at = now()
		returning
			user_id,
			token_ciphertext,
			token_key_version,
			default_workspace_id,
			default_workspace_name,
			status,
			last_checked_at,
			last_error,
			created_at,
			updated_at
	`,
		connection.UserID,
		connection.TokenCiphertext,
		connection.TokenKeyVersion,
		connection.DefaultWorkspaceID,
		connection.DefaultWorkspaceName,
		connection.Status,
		connection.LastCheckedAt,
		connection.LastError,
	).Scan(
		&out.UserID,
		&out.TokenCiphertext,
		&out.TokenKeyVersion,
		&out.DefaultWorkspaceID,
		&out.DefaultWorkspaceName,
		&out.Status,
		&out.LastCheckedAt,
		&out.LastError,
		&out.CreatedAt,
		&out.UpdatedAt,
	)
	return out, err
}

func (r *Repo) GetConnectionByUserID(ctx context.Context, userID string) (Connection, error) {
	var out Connection

	err := r.pool.QueryRow(ctx, `
		select
			user_id,
			token_ciphertext,
			token_key_version,
			default_workspace_id,
			default_workspace_name,
			status,
			last_checked_at,
			last_error,
			created_at,
			updated_at
		from clickup_connections
		where user_id = $1
	`, userID).Scan(
		&out.UserID,
		&out.TokenCiphertext,
		&out.TokenKeyVersion,
		&out.DefaultWorkspaceID,
		&out.DefaultWorkspaceName,
		&out.Status,
		&out.LastCheckedAt,
		&out.LastError,
		&out.CreatedAt,
		&out.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Connection{}, ErrConnectionNotFound
	}
	return out, err
}

func (r *Repo) MarkConnectionHealth(ctx context.Context, userID string, status string, lastError *string, checkedAt time.Time) error {
	result, err := r.pool.Exec(ctx, `
		update clickup_connections
		set
			status = $2,
			last_error = $3,
			last_checked_at = $4,
			updated_at = now()
		where user_id = $1
	`, userID, status, lastError, checkedAt)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrConnectionNotFound
	}
	return nil
}
