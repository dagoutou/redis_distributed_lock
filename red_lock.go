package redis_distributed_lock

import (
	"context"
	"errors"
	"time"
)

type RedLock struct {
	locks       []RedisLock
	expiredTime int // 分布式锁过期时间
	nodeTimeOut int // redis集群中单一redis的超时时间
}

type SingleNodeConf struct {
	Network  string
	Address  string
	Password string
	Opts     []ClientOption
}

func NewRedLock(key string, expiredTime, nodeTimeOut int, configs []SingleNodeConf) (*RedLock, error) {
	if len(configs) < 3 {
		return nil, errors.New("red lock can't use redis less 3")
	}
	if expiredTime > 0 && (len(configs)*nodeTimeOut > expiredTime) {
		return nil, errors.New("nodeTimeOut too long")
	}
	r := RedLock{}
	for _, conf := range configs {
		client := NewRedisClient(conf.Network, conf.Address, conf.Password, conf.Opts...)
		r.locks = append(r.locks, *NewRedisLock(key, client))
	}
	return &r, nil
}

func (r *RedLock) Lock(ctx context.Context) error {
	var successCount int
	for _, lock := range r.locks {
		startTime := time.Now()
		setp := time.Since(startTime)
		err := lock.Lock(ctx)
		if err == nil && setp <= time.Duration(r.nodeTimeOut) {
			successCount++
		}
	}
	if successCount < len(r.locks)>>1+1 {
		return errors.New("lock error")
	}
	return nil
}

func (r *RedLock) UnLock(ctx context.Context) error {
	var erro error
	for _, lock := range r.locks {
		if err := lock.UnLock(ctx); err != nil {
			erro = err
		}
	}
	return erro
}
