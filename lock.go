package redis_distributed_lock

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"
)

const (
	RedisLockKeyPrefix = "REDIS_LOCK_PREFIX_"
	WatchDogTimeStep   = 10
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
	LUADelayExpiredTime = `
local lockKey = KEYS[1]
local token = ARGV[1]
local time = ARGV[2]
local getToken = redis.call('get',lockKey)
if (not getToken or getToken ~= token) then 
        return 0
        else
        return redis.cal('expire',lockKey,time)
`
)

var ErrorRedisLockFail = errors.New("Failed to obtain lock")

type RedisLock struct {
	RedisLockOptions
	client LockClient
	token  string
	key    string

	runningWatchDog int32              // 看门狗标识位
	stopWatchDog    context.CancelFunc // 停止看门狗
}

func NewRedisLock(key string, client LockClient, opts ...RedisLockOption) *RedisLock {
	cl := RedisLock{
		client: client,
		token:  GetProcessAdnGoroutineIDStr(),
		key:    key,
	}
	for _, opt := range opts {
		opt(&cl.RedisLockOptions)
	}
	repairLock(&cl.RedisLockOptions)
	return &cl
}

func (r *RedisLock) Lock(ctx context.Context) (err error) {
	defer func() {
		if err != nil {
			return
		}
		r.watchDog(ctx)
	}()
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

func (r *RedisLock) watchDog(ctx context.Context) {
	if !r.watchDogMode {
		return
	}
	// 一次取锁只能开启一次看门狗
	for !atomic.CompareAndSwapInt32(&r.runningWatchDog, 0, 1) {
	}
	ctx, r.stopWatchDog = context.WithCancel(ctx)

	go func() {
		defer func() {
			atomic.CompareAndSwapInt32(&r.runningWatchDog, 1, 0)
		}()
		r.runWatchDog(ctx)
	}()
}

func (r *RedisLock) runWatchDog(ctx context.Context) {
	// 看门狗每10s续约一次
	ticker := time.NewTicker(time.Duration(WatchDogTimeStep) * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// 加10可以防止因为网络延时产生的锁提前释放
		_ = r.delayExpiredTime(ctx, WatchDogTimeStep+10)
	}
}

func (r *RedisLock) delayExpiredTime(ctx context.Context, expireSeconds int64) error {
	args := []interface{}{LUADelayExpiredTime, r.getLockKey(), expireSeconds}
	reply, err := r.client.Eval(ctx, LUADelayExpiredTime, 1, args)
	if err != nil {
		return err
	}
	if rep, _ := reply.(int64); rep != 1 {
		return errors.New("set expired time error")
	}
	return nil
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
