package barrier_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
)

// MockRedisClient adalah implementasi palsu dari redis.Client untuk pengujian.
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

// Exists adalah implementasi palsu dari fungsi Exists pada redis.Client untuk pengujian.
func (m *MockRedisClient) Exists(ctx context.Context, key string) (int64, error) {
	m.ExistsCalled = true
	m.ExistsKeyArg = key
	return m.ExistsResult, m.ExistsError
}

// Set adalah implementasi palsu dari fungsi Set pada redis.Client untuk pengujian.
func (m *MockRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) (string, error) {
	m.SetCalled = true
	m.SetKeyArg = key
	return m.SetResult, m.SetError
}

// Incr adalah implementasi palsu dari fungsi Incr pada redis.Client untuk pengujian.
func (m *MockRedisClient) Incr(ctx context.Context, key string) (int64, error) {
	m.IncrCalled = true
	m.IncrKeyArg = key
	return m.IncrResult, m.IncrError
}

func TestBarrier_CheckRateLimit(t *testing.T) {
	mockClient := &MockRedisClient{}
	barrier := &Barrier{
		client:  mockClient,
		expired: 1 * time.Minute,
		limit:   5,
	}
	req := ReqCheckLimit{
		URL:      "http://example.com",
		ClientID: "client1",
		UserID:   "user1",
	}

	t.Run("Exists returns error", func(t *testing.T) {
		expectedError := errors.New("redis error")
		mockClient.ExistsError = expectedError

		result := barrier.CheckRateLimit(context.Background(), req)

		if result {
			t.Error("Expected result to be false")
		}

		if !mockClient.ExistsCalled {
			t.Error("Expected Exists to be called")
		}

		if mockClient.ExistsKeyArg != "rate_limit:client1:user1:http://example.com" {
			t.Error("Expected Exists to be called with the correct key")
		}
	})

	t.Run("Key does not exist", func(t *testing.T) {
		mockClient.ExistsResult = 0
		mockClient.SetResult = "OK"

		result := barrier.CheckRateLimit(context.Background(), req)

		if !result {
			t.Error("Expected result to be true")
		}

		if !mockClient.ExistsCalled {
			t.Error("Expected Exists to be called")
		}

		if !mockClient.SetCalled {
			t.Error("Expected Set to be called")
		}

		if mockClient.SetKeyArg != "rate_limit:client1:user1:http://example.com" {
			t.Error("Expected Set to be called with the correct key")
		}
	})

	t.Run("Key exists and count is below limit", func(t *testing.T) {
		mockClient.ExistsResult = 1
		mockClient.IncrResult = 4

		result := barrier.CheckRateLimit(context.Background(), req)

		if !result {
			t.Error("Expected result to be true")
		}

		if !mockClient.ExistsCalled {
			t.Error("Expected Exists to be called")
		}

		if !mockClient.IncrCalled {
			t.Error("Expected Incr to be called")
		}

		if mockClient.IncrKeyArg != "rate_limit:client1:user1:http://example.com" {
			t.Error("Expected Incr to be called with the correct key")
		}
	})

	t.Run("Key exists and count is above limit", func(t *testing.T) {
		mockClient.ExistsResult = 1
		mockClient.IncrResult = 6

		result := barrier.CheckRateLimit(context.Background(), req)

		if result {
			t.Error("Expected result to be false")
		}

		if !mockClient.ExistsCalled {
			t.Error("Expected Exists to be called")
		}

		if !mockClient.IncrCalled {
			t.Error("Expected Incr to be called")
		}

		if mockClient.IncrKeyArg != "rate_limit:client1:user1:http://example.com" {
			t.Error("Expected Incr to be called with the correct key")
		}
	})
}

func TestNewBarrier(t *testing.T) {
	mockClient := &redis.Client{}
	expired := 5 * time.Minute
	limit := 10

	barrier := NewBarrier(mockClient, expired, limit)

	if barrier.client != mockClient {
		t.Error("Expected client to be set correctly")
	}

	if barrier.expired != expired {
		t.Error("Expected expired to be set correctly")
	}

	if barrier.limit != limit {
		t.Error("Expected limit to be set correctly")
	}
}

