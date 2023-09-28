package barrier_test

import (
	"errors"
	barrier "github.com/xans-me/barier"
	"testing"
	"time"
)

import (
	"context"
	"github.com/go-redis/redis/v8"
)

// MockRedisClient is a mock implementation of redis.Client for testing.
type MockRedisClient struct {
	ExistsResult int64
	SetResult    string
	IncrResult   int64
	ExistsError  error
	SetError     error
	IncrError    error
	ExistsCalled bool
	SetCalled    bool
	IncrCalled   bool
	ExistsKeyArg string
	SetKeyArg    string
	IncrKeyArg   string
}

// Exists is a mock implementation of the Exists function in redis.Client for testing.
func (m *MockRedisClient) Exists(ctx context.Context, key string) (int64, error) {
	m.ExistsCalled = true
	m.ExistsKeyArg = key
	return m.ExistsResult, m.ExistsError
}

// Set is a mock implementation of the Set function in redis.Client for testing.
func (m *MockRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) (string, error) {
	m.SetCalled = true
	m.SetKeyArg = key
	return m.SetResult, m.SetError
}

// Incr is a mock implementation of the Incr function in redis.Client for testing.
func (m *MockRedisClient) Incr(ctx context.Context, key string) (int64, error) {
	m.IncrCalled = true
	m.IncrKeyArg = key
	return m.IncrResult, m.IncrError
}

func TestBarrier_CheckRateLimit(t *testing.T) {
	mockClient := &MockRedisClient{}
	br := &barrier.Barrier{
		Client:  redis.NewClient(&redis.Options{}),
		Expired: 1 * time.Minute,
		Limit:   5,
	}
	req := barrier.ReqCheckLimit{
		URL:      "https://example.com",
		ClientID: "client1",
		UserID:   "user1",
	}

	t.Run("Exists returns error", func(t *testing.T) {
		expectedError := errors.New("redis error")
		mockClient.ExistsError = expectedError

		result := br.CheckRateLimit(context.Background(), req)

		if result {
			t.Error("Expected result to be false")
		}

		if !mockClient.ExistsCalled {
			t.Error("Expected Exists to be called")
		}

		if mockClient.ExistsKeyArg != "rate_limit:client1:user1:https://example.com" {
			t.Error("Expected Exists to be called with the correct key")
		}
	})

	t.Run("Key does not exist", func(t *testing.T) {
		mockClient.ExistsResult = 0
		mockClient.SetResult = "OK"

		result := br.CheckRateLimit(context.Background(), req)

		if !result {
			t.Error("Expected result to be true")
		}

		if !mockClient.ExistsCalled {
			t.Error("Expected Exists to be called")
		}

		if !mockClient.SetCalled {
			t.Error("Expected Set to be called")
		}

		if mockClient.SetKeyArg != "rate_limit:client1:user1:https://example.com" {
			t.Error("Expected Set to be called with the correct key")
		}
	})

	t.Run("Key exists and count is below Limit", func(t *testing.T) {
		mockClient.ExistsResult = 1
		mockClient.IncrResult = 4

		result := br.CheckRateLimit(context.Background(), req)

		if !result {
			t.Error("Expected result to be true")
		}

		if !mockClient.ExistsCalled {
			t.Error("Expected Exists to be called")
		}

		if !mockClient.IncrCalled {
			t.Error("Expected Incr to be called")
		}

		if mockClient.IncrKeyArg != "rate_limit:client1:user1:https://example.com" {
			t.Error("Expected Incr to be called with the correct key")
		}
	})

	t.Run("Key exists and count is above Limit", func(t *testing.T) {
		mockClient.ExistsResult = 1
		mockClient.IncrResult = 6

		result := br.CheckRateLimit(context.Background(), req)

		if result {
			t.Error("Expected result to be false")
		}

		if !mockClient.ExistsCalled {
			t.Error("Expected Exists to be called")
		}

		if !mockClient.IncrCalled {
			t.Error("Expected Incr to be called")
		}

		if mockClient.IncrKeyArg != "rate_limit:client1:user1:https://example.com" {
			t.Error("Expected Incr to be called with the correct key")
		}
	})
}

func TestNewBarrier(t *testing.T) {
	mockClient := &redis.Client{}
	expired := 5 * time.Minute
	limit := 10

	br := barrier.NewBarrier(mockClient, expired, limit)

	if br.Client != mockClient {
		t.Error("Expected Client to be set correctly")
	}

	if br.Expired != expired {
		t.Error("Expected Expired to be set correctly")
	}

	if br.Limit != limit {
		t.Error("Expected Limit to be set correctly")
	}
}
