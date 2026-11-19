package repository_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/felix-gnwn/ticketing-engine/internal/domain"
	repository "github.com/felix-gnwn/ticketing-engine/internal/repository/cache"
	"github.com/felix-gnwn/ticketing-engine/internal/repository/postgres"
)

func getTestRedisClient(t *testing.T) *redis.Client {
	t.Helper()

	redisHost := os.Getenv("REDIS_HOST")
	if redisHost == "" {
		redisHost = "localhost"
	}

	redisPort := os.Getenv("REDIS_PORT")
	if redisPort == "" {
		redisPort = "6379"
	}

	client := redis.NewClient(&redis.Options{
		Addr: redisHost + ":" + redisPort,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		t.Skipf("Skipping Redis integration test: Redis instance is unreachable: %v", err)
	}

	t.Cleanup(func() {
		client.Close()
	})

	return client
}

func getTestDBPoolForCache(t *testing.T) *pgxpool.Pool {
	t.Helper()

	connStr := os.Getenv("TEST_DB_CONNECTION_STRING")
	if connStr == "" {
		t.Skip("Skipping DB integration test for Cache: TEST_DB_CONNECTION_STRING environment variable is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Skipf("Skipping DB integration test for Cache: failed to create pool for test DB: %v", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("Skipping DB integration test for Cache: dedicated test database (ticketing_db_test) is unreachable: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
	})

	return pool
}

func TestCachedTicketRepository_Integration_CacheAsideAndInvalidation(t *testing.T) {
	dbPool := getTestDBPoolForCache(t)
	redisClient := getTestRedisClient(t)
	ctx := context.Background()

	baseRepo := postgres.NewTicketRepository(dbPool)
	cachedRepo := repository.NewCachedTicketRepository(baseRepo, redisClient)

	// Flush target cache key before test execution
	_ = redisClient.Del(ctx, repository.AvailableTicketsCacheKey).Err()

	ticketID := uuid.New().String()
	ticket := &domain.Ticket{
		ID:        ticketID,
		EventName: "VIP Gala (Cache Integration Test)",
		Price:     2500000,
		Quota:     5,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	// 1. Create Ticket -> Triggers Proactive Cache Invalidation
	err := cachedRepo.Create(ctx, ticket)
	require.NoError(t, err)

	defer func() {
		_ = redisClient.Del(ctx, repository.AvailableTicketsCacheKey).Err()
		_, _ = dbPool.Exec(ctx, "DELETE FROM tickets WHERE id = $1", ticketID)
	}()

	exists, err := redisClient.Exists(ctx, repository.AvailableTicketsCacheKey).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(0), exists, "Redis key should be empty after Create")

	// 2. First Call: GetAvailableTickets -> Cache MISS (Fetch from DB & populate Redis)
	ticketsMiss, err := cachedRepo.GetAvailableTickets(ctx)
	require.NoError(t, err)
	assert.True(t, len(ticketsMiss) > 0)

	// Verify Redis key is now populated (Cache Miss Step)
	existsAfterMiss, err := redisClient.Exists(ctx, repository.AvailableTicketsCacheKey).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(1), existsAfterMiss, "Redis key should be populated after Cache Miss")

	// 3. Second Call: GetAvailableTickets -> Cache HIT (Fetch directly from Redis)
	ticketsHit, err := cachedRepo.GetAvailableTickets(ctx)
	require.NoError(t, err)
	assert.Equal(t, len(ticketsMiss), len(ticketsHit))

	// 4. DecreaseQuota -> Triggers Proactive Cache Invalidation
	err = cachedRepo.DecreaseQuota(ctx, ticketID, 1)
	require.NoError(t, err)

	// Verify Redis key is automatically invalidated
	existsAfterMutation, err := redisClient.Exists(ctx, repository.AvailableTicketsCacheKey).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(0), existsAfterMutation, "Redis key should be invalidated after DecreaseQuota mutation")
}
