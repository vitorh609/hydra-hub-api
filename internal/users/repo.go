package users

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound = errors.New("not found")
)

type Repo struct {
	pool *pgxpool.Pool
}

func NewRepo(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

func (r *Repo) Create(ctx context.Context, in CreateUserInput) (User, error) {
	var u User

	row := r.pool.QueryRow(ctx, `
		insert into users (name, email, password_hash, phone)
		values ($1, $2, $3, $4)
		returning id, name, email, password_hash, phone, created_at, updated_at
	`, in.Name, in.Email, in.PasswordHash, in.Phone)

	if err := row.Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.Phone, &u.CreatedAt, &u.UpdatedAt); err != nil {
		return User{}, err
	}
	return u, nil
}

func (r *Repo) List(ctx context.Context) ([]User, error) {
	rows, err := r.pool.Query(ctx, `
		select id, name, email, password_hash, phone, created_at, updated_at
		from users
		order by created_at desc
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.Phone, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (r *Repo) GetByID(ctx context.Context, id string) (User, error) {
	var u User
	err := r.pool.QueryRow(ctx, `
		select id, name, email, password_hash, phone, created_at, updated_at
		from users
		where id = $1
	`, id).Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.Phone, &u.CreatedAt, &u.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return u, err
}

func (r *Repo) Update(ctx context.Context, id string, in UpdateUserInput) (User, error) {
	// update parcial: se campo vier nil, mantém o atual
	var u User
	err := r.pool.QueryRow(ctx, `
		update users
		set
			name = coalesce($2, name),
			email = coalesce($3, email),
			password_hash = coalesce($4, password_hash),
			phone = $5,
			updated_at = $6
		where id = $1
		returning id, name, email, password_hash, phone, created_at, updated_at
	`,
		id,
		in.Name,
		in.Email,
		in.PasswordHash,
		in.Phone,
		time.Now().UTC(),
	).Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.Phone, &u.CreatedAt, &u.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return u, err
}

func (r *Repo) Delete(ctx context.Context, id string) error {
	ct, err := r.pool.Exec(ctx, `delete from users where id = $1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
