package redis_lock

import (
	"context"
	"errors"
	"fmt"
	"redis_lock/redis_client"
	"redis_lock/utils"
	"time"
)

const (
	RedisLockKeyPrefix = "REDIS_LOCK_PREFIX_"
	LUADeleteLock      = `
local lockKey = KEYS[1]
local token = ARGV[1]
local getToken = redis.call('get',lockKey)
if (not getToken or getToken ~=token) then
   return 0
   else
   return redis.cal('del',token)
      end
`
)

var ErrorRedisLockFail = errors.New("Failed to obtain lock")

type RedisLock struct {
	client        *redis_client.Client
	token         string
	key           string
	expireSeconds int64

	isBlock             bool  // 是否是阻塞模式
	blockWaitingSeconds int64 // 能阻塞等待的时间
}

func NewRedisLock(key string, client *redis_client.Client) *RedisLock {
	return &RedisLock{
		client: client,
		token:  utils.GetProcessAdnGoroutineIDStr(),
		key:    key,
	}
}

func (r *RedisLock) Lock(ctx context.Context) (err error) {
	// 尝试获取锁
	err = r.tryLock(ctx)
	if err == nil {
		return nil
	}
	// 若是阻塞模式获取锁失败直接返回
	if !r.isBlock {
		return err
	}
	if !IsRetryErr(err) {
		return err
	}

	// 基于阻塞模式轮询取锁
	err = r.blockingLock(ctx)
	return
}

func IsRetryErr(err error) bool {
	return errors.Is(err, ErrorRedisLockFail)
}

func (r *RedisLock) tryLock(ctx context.Context) error {
	// 尝试获取锁，获取锁成功返回1否则获取锁失败
	reply, err := r.client.SetNEx(ctx, r.getLockKey(), r.token, r.expireSeconds)
	if err != nil {
		return err
	}
	if reply != 1 {
		return fmt.Errorf("reply: %d, err: %w", reply, ErrorRedisLockFail)
	}
	return nil
}

func (r *RedisLock) blockingLock(ctx context.Context) error {
	lockExpireTime := time.After(time.Duration(r.blockWaitingSeconds) * time.Second)
	ticker := time.NewTicker(time.Duration(50) * time.Millisecond)
	for range ticker.C {
		select {
		case <-ctx.Done():
			return fmt.Errorf("lock failed, ctx timeout ,error:%w ", ctx.Err())
		case <-lockExpireTime:
			return fmt.Errorf("block waiting time out, err: %w", ErrorRedisLockFail)
		default:
		}
		err := r.tryLock(ctx)
		if err == nil {
			return nil
		}
		if !IsRetryErr(err) {
			return err
		}
	}
	return nil
}

func (r *RedisLock) getLockKey() string {
	return RedisLockKeyPrefix + r.key
}

func (r *RedisLock) UnLock(ctx context.Context) error {
	args := []interface{}{r.getLockKey(), r.token}
	reply, err := r.client.Eval(ctx, LUADeleteLock, 1, args)
	if err != nil {
		return err
	}
	if ret, _ := reply.(int64); ret != 1 {
		return errors.New("can not unlock")
	}
	return nil
}
