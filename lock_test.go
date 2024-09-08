package redis_distributed_lock

import (
	"context"
	"sync"
	"testing"
)

func Test_blockLock(t *testing.T) {
	client := NewRedisClient("tcp", "127.0.0.1:6379", "")
	redis_lock1 := NewRedisLock("test_key", client, WithExpireSeconds(100))
	redis_lock2 := NewRedisLock("test_key", client, WithExpireSeconds(4))

	ctx := context.Background()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := redis_lock1.Lock(ctx); err != nil {
			t.Error(err)
			return
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := redis_lock2.Lock(ctx); err != nil {
			t.Error(err)
			return
		}
	}()
	wg.Wait()
	t.Log("执行完成------")
}
