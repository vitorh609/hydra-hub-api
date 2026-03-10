package account_settings

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	// ErrNotFound padroniza o erro usado quando o registro não existe.
	// O handler traduz isso para HTTP 404.
	ErrNotFound = errors.New("not found")
)

// Repo encapsula o acesso ao banco para o recurso notes.
// Mantemos SQL explícito (sem ORM) para clareza e controle.
type Repo struct {
	pool *pgxpool.Pool
}

func NewRepo(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

func (r *Repo) Create(ctx context.Context, in CreateAccountSettingsInput) (AccountSettings, error) {
	var a AccountSettings

	row := r.pool.QueryRow(ctx, `
		insert into account_settings (email, phone, name, address, avatar_base64, location)
		values ($1, $2, $3, $4, $5, $6)
		returning user_id, email, phone, name, address, avatar_base64, location, created_at, updated_at
	`, in.Email, in.Phone, in.Name, in.Address, in.AvatarBase64, in.Location)

	if err := row.Scan(&a.UserId, &a.Phone, &a.Name, &a.Email, &a.Location, &a.Address, &a.UpdatedAt); err != nil {
		return AccountSettings{}, err
	}

	return a, nil
}

func (r *Repo) List(ctx context.Context) ([]AccountSettings, error) {
	rows, err := r.pool.Query(ctx, `
		select user_id, email, phone, name, address, avatar_base64, location, created_at, updated_at
		from account_settings
		order by created_at desc
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []AccountSettings

	for rows.Next() {
		var a AccountSettings
		if err := rows.Scan(&a.UserId, &a.Phone, &a.Name, &a.Email, &a.Location, &a.Address, &a.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}

	return out, rows.Err()
}

func (r *Repo) GetByID(ctx context.Context, id string) (AccountSettings, error) {
	var a AccountSettings

	err := r.pool.QueryRow(ctx, `
		select user_id, email, phone, name, address, avatar_base64, location, created_at, updated_at
		from account_settings
		where user_id = $1
	`, id).Scan(&a.UserId, &a.Phone, &a.Name, &a.Email, &a.Location, &a.Address, &a.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return AccountSettings{}, ErrNotFound
	}
	return a, err
}

func (r *Repo) Update(ctx context.Context, id string, in UpdateAccountSettingsInput) (AccountSettings, error) {
	var a AccountSettings

	err := r.pool.QueryRow(ctx, `
		update account_settings
		set 
		    avatarBase64 = coalesce($2, avatarBase64),
			name = coalesce($3, name),
			location = coalesce($4, location),
			email = coalesce($5, email),
			phone = coalesce($6, phone),
			address = coalesce($7, address),
			updated_at = $8
		where user_id = $1
		returning user_id, email, phone, name, address, avatarBase64, location, created_at, updated_at
	`,
		id,
		in.Email,
		in.Phone,
		in.Name,
		in.Address,
		in.AvatarBase64,
		in.Location,
		time.Now().UTC(),
	).Scan(&a.UserId, &a.Phone, &a.Name, &a.Email, &a.Location, &a.Address, &a.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return AccountSettings{}, ErrNotFound
	}
	return a, err
}

func (r *Repo) Delete(ctx context.Context, id string) error {
	ct, err := r.pool.Exec(ctx, `delete from account_settings where user_id = $1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
