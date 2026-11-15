package repository

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/felix-gnwn/ticketing-engine/internal/domain"
)

const (
	AvailableTicketsCacheKey = "tickets:available"
	AvailableTicketsCacheTTL = 5 * time.Minute
)

type cachedTicketRepository struct {
	repo  domain.TicketRepository
	redis *redis.Client
}

func NewCachedTicketRepository(repo domain.TicketRepository, redisClient *redis.Client) domain.TicketRepository {
	return &cachedTicketRepository{
		repo:  repo,
		redis: redisClient,
	}
}

func (r *cachedTicketRepository) Create(ctx context.Context, ticket *domain.Ticket) error {
	if err := r.repo.Create(ctx, ticket); err != nil {
		return err
	}

	// Proactive Cache Invalidation
	r.invalidateAvailableTicketsCache(ctx)
	return nil
}

func (r *cachedTicketRepository) FindByID(ctx context.Context, id string) (*domain.Ticket, error) {
	// Delegate directly to PostgreSQL repository
	return r.repo.FindByID(ctx, id)
}

func (r *cachedTicketRepository) GetAvailableTickets(ctx context.Context) ([]*domain.Ticket, error) {
	// 1. Try reading from Redis Cache (Cache Hit)
	cachedData, err := r.redis.Get(ctx, AvailableTicketsCacheKey).Result()
	if err == nil {
		var tickets []*domain.Ticket
		if err := json.Unmarshal([]byte(cachedData), &tickets); err == nil {
			slog.Debug("Redis Cache HIT: GetAvailableTickets", "key", AvailableTicketsCacheKey)
			return tickets, nil
		}
	}

	// 2. Cache Miss: Read from PostgreSQL Source of Truth
	slog.Debug("Redis Cache MISS: GetAvailableTickets fetching from PostgreSQL", "key", AvailableTicketsCacheKey)
	tickets, err := r.repo.GetAvailableTickets(ctx)
	if err != nil {
		return nil, err
	}

	// 3. Save to Redis Cache with TTL (5 minutes)
	if bytes, err := json.Marshal(tickets); err == nil {
		if setErr := r.redis.Set(ctx, AvailableTicketsCacheKey, bytes, AvailableTicketsCacheTTL).Err(); setErr != nil {
			slog.Warn("Failed to write available tickets to Redis cache", "error", setErr)
		}
	}

	return tickets, nil
}

func (r *cachedTicketRepository) DecreaseQuota(ctx context.Context, id string, quantity int) error {
	if err := r.repo.DecreaseQuota(ctx, id, quantity); err != nil {
		return err
	}

	// Proactive Cache Invalidation after stock mutation
	r.invalidateAvailableTicketsCache(ctx)
	return nil
}

func (r *cachedTicketRepository) invalidateAvailableTicketsCache(ctx context.Context) {
	if err := r.redis.Del(ctx, AvailableTicketsCacheKey).Err(); err != nil {
		slog.Warn("Failed to invalidate available tickets cache", "key", AvailableTicketsCacheKey, "error", err)
	} else {
		slog.Debug("Redis Cache INVALIDATED: Available tickets cache cleared", "key", AvailableTicketsCacheKey)
	}
}
