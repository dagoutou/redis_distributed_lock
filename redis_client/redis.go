package redis_client

import (
	"context"
	"github.com/gomodule/redigo/redis"
	"time"
)

const (
	DefaultMaxIdle           = 10
	DefaultMaxActive         = 100
	DefaultIdleTimeoutSecond = 10 // 最大空闲时间10s
)

type Client struct {
	pools *redis.Pool
	ClientOptions
}

func NewRedisClient(network, address, password string, opts ...ClientOption) *Client {
	c := Client{
		ClientOptions: ClientOptions{
			network:  network,
			address:  address,
			password: password,
		},
	}
	for _, opt := range opts {
		opt(&c.ClientOptions)
	}

	repairLock(&c.ClientOptions)

	pools := c.getRedisPools()

	return &Client{
		pools: pools,
	}
}

func repairLock(c *ClientOptions) {
	if c.maxIdle < 0 {
		c.maxIdle = DefaultMaxIdle
	}
	if c.maxActive < 0 {
		c.maxActive = DefaultMaxActive
	}
	if c.idleTimeoutSecond < 0 {
		c.idleTimeoutSecond = DefaultIdleTimeoutSecond
	}
}

func (c *Client) getRedisPools() *redis.Pool {
	return &redis.Pool{
		Dial: func() (redis.Conn, error) {
			conn, err := c.getRedisConn()
			if err != nil {
				return nil, err
			}
			return conn, nil
		},
		TestOnBorrow: func(c redis.Conn, lastUsed time.Time) error {
			_, err := c.Do("PING")
			return err
		},
		MaxIdle:     c.maxIdle,
		MaxActive:   c.maxActive,
		IdleTimeout: time.Duration(c.idleTimeoutSecond) * time.Second,
		Wait:        c.wait,
	}
}

func (c *Client) getRedisConn() (redis.Conn, error) {
	if c.address == "" {
		panic("redis address cant null")
	}
	if c.network == "" {
		panic("redis network cant null")
	}
	var redisOptions []redis.DialOption
	if len(c.password) > 0 {
		redisOptions = append(redisOptions, redis.DialPassword(c.password))
	}

	conn, err := redis.DialContext(context.Background(), c.network, c.address, redisOptions...)
	if err != nil {
		return nil, err
	}
	return conn, nil
}
