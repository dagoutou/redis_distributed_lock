package redis_client

import "testing"

func TestNewRedisClient(t *testing.T) {
	client := NewRedisClient("tcp", "127.0.0.1:6379", "")
}
