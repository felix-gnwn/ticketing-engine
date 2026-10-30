package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/felix-gnwn/ticketing-engine/internal/domain"
)

type ticketRepository struct {
	db *pgxpool.Pool
}

func NewTicketRepository(db *pgxpool.Pool) domain.TicketRepository {
	return &ticketRepository{
		db: db,
	}
}

func (r *ticketRepository) Create(ctx context.Context, ticket *domain.Ticket) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
		INSERT INTO tickets (
			id,
			event_name,
			price,
			quota,
			created_at,
			updated_at
		)
		VALUES (
			$1,
			$2,
			$3,
			$4,
			$5,
			$6
		)
	`
	
	_, err := r.db.Exec(ctx, query,
		ticket.ID,
		ticket.EventName,
		ticket.Price,
		ticket.Quota,
		ticket.CreatedAt,
		ticket.UpdatedAt,
	)

	return err
}

func (r *ticketRepository) FindByID(ctx context.Context, id string) (*domain.Ticket, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	query := `
		SELECT
			id,
			event_name,
			price,
			quota,
			created_at,
			updated_at
		FROM
			tickets
		WHERE
			id = $1
	`

	var ticket domain.Ticket
	err := r.db.QueryRow(ctx, query, id).Scan(
		&ticket.ID,
		&ticket.EventName,
		&ticket.Price,
		&ticket.Quota,
		&ticket.CreatedAt,
		&ticket.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("ticket not found")
		}

		return nil, err
	}

	return &ticket, nil
}

func (r *ticketRepository) GetAvailableTickets(ctx context.Context) ([]*domain.Ticket, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	query := `
		SELECT
			id,
			event_name,
			price,
			quota,
			created_at,
			updated_at
		FROM
			tickets
		WHERE
			quota > 0
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tickets []*domain.Ticket
	for rows.Next() {
		var t domain.Ticket
		err := rows.Scan(
			&t.ID,
			&t.EventName,
			&t.Price,
			&t.Quota,
			&t.CreatedAt,
			&t.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		tickets = append(tickets, &t)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return tickets, nil
}

func (r *ticketRepository) DecreaseQuota(ctx context.Context, id string, quantity int) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	query := `
		UPDATE
			tickets
		SET
			quota = quota - $2,
			updated_at = $3
		WHERE
			id = $1
			AND quota >= $2
	`

	now := time.Now().UTC()

	commandTag, err := r.db.Exec(ctx, query, id, quantity, now)
	if err != nil {
		return err
	}

	if commandTag.RowsAffected() == 0 {
		return domain.ErrTicketUnavailable
	}

	return nil
}