package barrier

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

type Barrier struct {
	client  *redis.Client
	expired time.Duration
	limit   int
}

type ReqCheckLimit struct {
	URL      string
	ClientID string
	UserID   string
}

func (br *Barrier) CheckRateLimit(ctx context.Context, req ReqCheckLimit) bool {
	key := fmt.Sprintf("rate_limit:%s:%s:%s",
		req.ClientID, req.UserID, req.URL)

	exists, err := br.client.Exists(ctx, key).Result()
	if err != nil {
		// handle error if needed
		return false
	}

	if exists == 0 {
		_, err := br.client.Set(ctx, key, 1, br.expired*time.Minute).Result()
		if err != nil {
			// handle error if needed
			return false
		}
		return true
	}

	count, err := br.client.Incr(ctx, key).Result()
	if err != nil {
		// handle error if needed
		return false
	}

	if count > int64(br.limit) {
		return false
	}

	return true
}

func NewBarrier(client *redis.Client, expired time.Duration, limit int) *Barrier {
    return &Barrier{client: client, expired: expired, limit: limit}
}
