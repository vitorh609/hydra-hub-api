package notes

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

func (r *Repo) Create(ctx context.Context, in CreateNoteInput) (Note, error) {
	var n Note

	row := r.pool.QueryRow(ctx, `
		insert into notes (title, body_text, color)
		values ($1, $2, $3)
		returning id, title, body_text, color, created_at, updated_at
	`, in.Title, in.BodyText, in.Color)

	if err := row.Scan(&n.ID, &n.Title, &n.BodyText, &n.Color, &n.CreatedAt, &n.UpdatedAt); err != nil {
		return Note{}, err
	}

	return n, nil
}

func (r *Repo) List(ctx context.Context) ([]Note, error) {
	rows, err := r.pool.Query(ctx, `
		select id, title, body_text, color, created_at, updated_at
		from notes
		order by created_at desc
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Note
	for rows.Next() {
		var n Note
		if err := rows.Scan(&n.ID, &n.Title, &n.BodyText, &n.Color, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, n)
	}

	return out, rows.Err()
}

func (r *Repo) GetByID(ctx context.Context, id string) (Note, error) {
	var n Note

	err := r.pool.QueryRow(ctx, `
		select id, title, body_text, color, created_at, updated_at
		from notes
		where id = $1
	`, id).Scan(&n.ID, &n.Title, &n.BodyText, &n.Color, &n.CreatedAt, &n.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return Note{}, ErrNotFound
	}

	return n, err
}

func (r *Repo) Update(ctx context.Context, id string, in UpdateNoteInput) (Note, error) {
	// Update parcial: cada campo só é alterado se vier no JSON (não-nil).
	// `updated_at` é atualizado na query por consistência; se houver trigger no banco,
	// ele também vai ajustar o valor antes de persistir.
	var n Note

	err := r.pool.QueryRow(ctx, `
		update notes
		set
			title = coalesce($2, title),
			body_text = coalesce($3, body_text),
			color = coalesce($4, color),
			updated_at = $5
		where id = $1
		returning id, title, body_text, color, created_at, updated_at
	`,
		id,
		in.Title,
		in.BodyText,
		in.Color,
		time.Now().UTC(),
	).Scan(&n.ID, &n.Title, &n.BodyText, &n.Color, &n.CreatedAt, &n.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return Note{}, ErrNotFound
	}

	return n, err
}

func (r *Repo) Delete(ctx context.Context, id string) error {
	ct, err := r.pool.Exec(ctx, `delete from notes where id = $1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
