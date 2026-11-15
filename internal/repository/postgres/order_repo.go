package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/felix-gnwn/ticketing-engine/internal/domain"
)

type orderRepository struct {
	db *pgxpool.Pool
}

func NewOrderRepository(db *pgxpool.Pool) domain.OrderRepository {
	return &orderRepository{
		db: db,
	}
}

func (r *orderRepository) Create(ctx context.Context, order *domain.Order) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Safe: no-op if transaction is already committed
	updateQuotaQuery := `
		UPDATE tickets
		SET
			quota = quota - $2,
			updated_at = $3
		WHERE
			id = $1 AND quota >= $2
	`
	cmdTag, err := tx.Exec(ctx, updateQuotaQuery, order.TicketID, order.Quantity, order.UpdatedAt)
	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return domain.ErrInsufficientQuota
	}

	insertOrderQuery := `
		INSERT INTO orders (
			id,
			user_id,
			ticket_id,
			quantity,
			total_price,
			status,
			created_at,
			updated_at
		)
		VALUES (
			$1,
			$2,
			$3,
			$4,
			$5,
			$6,
			$7,
			$8
		)
	`
	_, err = tx.Exec(ctx, insertOrderQuery, order.ID, order.UserID, order.TicketID, order.Quantity, order.TotalPrice, order.Status, order.CreatedAt, order.UpdatedAt)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// FindByID retrieves an order by its unique identifier.
func (r *orderRepository) FindByID(ctx context.Context, id string) (*domain.Order, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	query := `
		SELECT
			id,
			user_id,
			ticket_id,
			quantity,
			total_price,
			status,
			created_at,
			updated_at
		FROM
			orders
		WHERE
			id = $1
	`
	var o domain.Order
	err := r.db.QueryRow(ctx, query, id).Scan(
		&o.ID,
		&o.UserID,
		&o.TicketID,
		&o.Quantity,
		&o.TotalPrice,
		&o.Status,
		&o.CreatedAt,
		&o.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrOrderNotFound
		}

		return nil, err
	}

	return &o, nil
}

func (r *orderRepository) FindByUserID(ctx context.Context, userID string) ([]*domain.Order, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	query := `
		SELECT
			id,
			user_id,
			ticket_id,
			quantity,
			total_price,
			status,
			created_at,
			updated_at
		FROM
			orders
		WHERE
			user_id = $1
		ORDER BY
			created_at DESC
	`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []*domain.Order
	for rows.Next() {
		var o domain.Order
		err := rows.Scan(
			&o.ID,
			&o.UserID,
			&o.TicketID,
			&o.Quantity,
			&o.TotalPrice,
			&o.Status,
			&o.CreatedAt,
			&o.UpdatedAt,
		)

		if err != nil {
			return nil, err
		}

		orders = append(orders, &o)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return orders, nil
}

func (r *orderRepository) UpdateStatus(ctx context.Context, id string, status domain.OrderStatus) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	query := `
		UPDATE orders
		SET
			status = $2,
			updated_at = $3
		WHERE
			id = $1
	`

	now := time.Now().UTC()
	cmdTag, err := r.db.Exec(ctx, query, id, status, now)

	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return domain.ErrOrderNotFound
	}

	return nil
}
