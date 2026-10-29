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

func newTicketRepository(db *pgxpool.Pool) domain.TicketRepository {
	return &ticketRepository{
		db: db,
	}
}

func (r *ticketRepository) Insert(ctx context.Context, ticket *domain.Ticket) error {
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

	if err != nil {
		return err
	}

	return nil
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