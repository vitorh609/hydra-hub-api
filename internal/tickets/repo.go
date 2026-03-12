package tickets

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

func (r *Repo) Create(ctx context.Context, in CreateTicketInput) (Ticket, error) {
	var t Ticket

	row := r.pool.QueryRow(ctx, `
		insert into tickets (title, description, due_date)
		values ($1, $2, $3)
		returning id, title, description, status, created_at, updated_at, due_date, cod_ticket
	`, in.Title, in.Description, in.DueDate)

	if err := row.Scan(&t.ID, &t.Title, &t.Description, &t.Status, &t.CreatedAt, &t.UpdatedAt, &t.DueDate, &t.CodTicket); err != nil {
		return Ticket{}, err
	}

	return t, nil
}

func (r *Repo) List(ctx context.Context) ([]Ticket, error) {
	rows, err := r.pool.Query(ctx, `
		select id, title, description, status, created_at, updated_at, due_date, cod_ticket
		from tickets
		order by created_at desc
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Ticket
	for rows.Next() {
		var t Ticket
		if err := rows.Scan(&t.ID, &t.Title, &t.Description, &t.Status, &t.CreatedAt, &t.UpdatedAt, &t.DueDate, &t.CodTicket); err != nil {
			return nil, err
		}
		out = append(out, t)
	}

	return out, rows.Err()
}

func (r *Repo) GetByID(ctx context.Context, id string) (Ticket, error) {
	var t Ticket

	err := r.pool.QueryRow(ctx, `
		select id, title, description, status, created_at, updated_at, due_date, cod_ticket
		from tickets
		where id = $1
	`, id).Scan(&t.ID, &t.Title, &t.Description, &t.Status, &t.CreatedAt, &t.UpdatedAt, &t.DueDate, &t.CodTicket)

	if errors.Is(err, pgx.ErrNoRows) {
		return Ticket{}, ErrNotFound
	}

	return t, err
}

func (r *Repo) Update(ctx context.Context, id string, in UpdateTicketInput) (Ticket, error) {
	var t Ticket

	err := r.pool.QueryRow(ctx, `
		update tickets
		set
			title = coalesce($2, title),
			description = coalesce($3, description),
			status = coalesce($4, status),
			due_date = coalesce($5, due_date),
			updated_at = $6
		where id = $1
		returning id, title, description, status, created_at, updated_at, due_date, cod_ticket
	`,
		id,
		in.Title,
		in.Description,
		in.Status,
		in.DueDate,
		time.Now().UTC(),
	).Scan(&t.ID, &t.Title, &t.Description, &t.Status, &t.CreatedAt, &t.UpdatedAt, &t.DueDate, &t.CodTicket)

	if errors.Is(err, pgx.ErrNoRows) {
		return Ticket{}, ErrNotFound
	}

	return t, err
}

func (r *Repo) Delete(ctx context.Context, id string) error {
	ct, err := r.pool.Exec(ctx, `delete from tickets where id = $1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
