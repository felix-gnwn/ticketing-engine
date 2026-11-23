package postgres_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/felix-gnwn/ticketing-engine/internal/domain"
	"github.com/felix-gnwn/ticketing-engine/internal/repository/postgres"
)

func getTestDBPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	connStr := os.Getenv("TEST_DB_CONNECTION_STRING")
	if connStr == "" {
		t.Skip("Skipping PostgreSQL integration test: TEST_DB_CONNECTION_STRING environment variable is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Skipf("Skipping PostgreSQL integration test: failed to create pool for test DB: %v", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("Skipping PostgreSQL integration test: dedicated test database (ticketing_db_test) is unreachable: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
	})

	return pool
}

func TestOrderRepository_Integration_CreateAndFind(t *testing.T) {
	pool := getTestDBPool(t)
	ctx := context.Background()

	ticketRepo := postgres.NewTicketRepository(pool)
	orderRepo := postgres.NewOrderRepository(pool)

	// 1. Seed User Fixture (required to satisfy Foreign Key constraint fk_orders_user)
	userID := uuid.New().String()
	userQuery := `
		INSERT INTO	users (
			id
			, name
			, email
			, password
			, role
			, created_at
			, updated_at
		)
		VALUES (
			$1
			, $2
			, $3
			, $4
			, $5
			, $6
			, $7
		)
	`
	_, err := pool.Exec(ctx, userQuery, userID, "Integration Test User", "test_"+userID+"@example.com", "hashedpassword", "user", time.Now().UTC(), time.Now().UTC())
	require.NoError(t, err)

	// 2. Seed Ticket Data Fixture
	ticketID := uuid.New().String()
	ticket := &domain.Ticket{
		ID:        ticketID,
		EventName: "Coldplay Live (Integration Test)",
		Price:     1500000,
		Quota:     10,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	err = ticketRepo.Create(ctx, ticket)
	require.NoError(t, err)

	// Clean up test fixtures after test execution
	defer func() {
		_, _ = pool.Exec(ctx, "DELETE FROM orders WHERE ticket_id = $1", ticketID)
		_, _ = pool.Exec(ctx, "DELETE FROM tickets WHERE id = $1", ticketID)
		_, _ = pool.Exec(ctx, "DELETE FROM users WHERE id = $1", userID)
	}()

	// 3. Test Atomic Create Order (SQL Transaction: Decrease Quota + Insert Order)
	orderID := uuid.New().String()
	order := &domain.Order{
		ID:         orderID,
		UserID:     userID,
		TicketID:   ticketID,
		Quantity:   2,
		TotalPrice: 3000000,
		Status:     domain.OrderStatusPending,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}

	err = orderRepo.Create(ctx, order)
	require.NoError(t, err)

	// Verify Ticket Quota Decreased from 10 to 8 in database
	fetchedTicket, err := ticketRepo.FindByID(ctx, ticketID)
	require.NoError(t, err)
	assert.Equal(t, 8, fetchedTicket.Quota, "ticket quota should decrease to 8")

	// 4. Test FindByID
	fetchedOrder, err := orderRepo.FindByID(ctx, orderID)
	require.NoError(t, err)
	assert.Equal(t, orderID, fetchedOrder.ID)
	assert.Equal(t, userID, fetchedOrder.UserID)

	// 5. Test FindByUserID
	userOrders, err := orderRepo.FindByUserID(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, userOrders, 1)

	// 6. Test UpdateStatus
	err = orderRepo.UpdateStatus(ctx, orderID, domain.OrderStatusPaid)
	require.NoError(t, err)

	updatedOrder, err := orderRepo.FindByID(ctx, orderID)
	require.NoError(t, err)
	assert.Equal(t, domain.OrderStatusPaid, updatedOrder.Status)
}

func TestOrderRepository_Integration_CreateOrder_InsufficientQuota(t *testing.T) {
	pool := getTestDBPool(t)
	ctx := context.Background()

	ticketRepo := postgres.NewTicketRepository(pool)
	orderRepo := postgres.NewOrderRepository(pool)

	ticketID := uuid.New().String()
	ticket := &domain.Ticket{
		ID:        ticketID,
		EventName: "Limited Pass (Integration Test)",
		Price:     500000,
		Quota:     1, // Only 1 ticket left
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	err := ticketRepo.Create(ctx, ticket)
	require.NoError(t, err)

	defer func() {
		_, _ = pool.Exec(ctx, "DELETE FROM orders WHERE ticket_id = $1", ticketID)
		_, _ = pool.Exec(ctx, "DELETE FROM tickets WHERE id = $1", ticketID)
	}()

	// Attempt to purchase 5 tickets when only 1 is available
	order := &domain.Order{
		ID:         uuid.New().String(),
		UserID:     uuid.New().String(),
		TicketID:   ticketID,
		Quantity:   5,
		TotalPrice: 2500000,
		Status:     domain.OrderStatusPending,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}

	err = orderRepo.Create(ctx, order)
	assert.ErrorIs(t, err, domain.ErrInsufficientQuota)

	// Quota in database must remain 1 (transaction rolled back)
	fetchedTicket, err := ticketRepo.FindByID(ctx, ticketID)
	require.NoError(t, err)
	assert.Equal(t, 1, fetchedTicket.Quota)
}
