package redis_distributed_lock

import (
	"context"
	"errors"
	"github.com/gomodule/redigo/redis"
	"strings"
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

type LockClient interface {
	SetNEx(ctx context.Context, key, value string, expireSeconds int64) (int64, error)
	Eval(ctx context.Context, src string, keyCount int, keysAndArgs []interface{}) (interface{}, error)
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

	repairClient(&c.ClientOptions)

	pools := c.getRedisPools()

	return &Client{
		pools: pools,
	}
}

func repairClient(c *ClientOptions) {
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
				panic(err)
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
	if _, err = conn.Do("SELECT", 0); err != nil {
		conn.Close()
		return nil, err
	}
	return conn, nil
}

func (c *Client) Get(ctx context.Context, key string) (string, error) {
	if key == "" {
		return "", errors.New("redis GET key can't be empty")
	}
	conn, err := c.pools.GetContext(ctx)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	return redis.String(conn.Do("GET", key))
}

func (c *Client) Set(ctx context.Context, key string) (int, error) {
	if key == "" {
		return 0, errors.New("redis SET key can't be empty")
	}
	conn, err := c.pools.GetContext(ctx)
	if err != nil {
		return 0, err
	}
	defer conn.Close()
	return redis.Int(conn.Do("SET", key))
}

func (c *Client) SetNEx(ctx context.Context, key, value string, expiredSeconds int64) (int64, error) {
	if key == "" || value == "" {
		return -1, errors.New("redis SETNEX key or value can't be empty")
	}
	conn, err := c.pools.GetContext(ctx)
	if err != nil {
		return -1, err
	}
	defer conn.Close()
	reply, err := conn.Do("SET", key, value, "EX", expiredSeconds, "NX")
	if err != nil {
		return -1, err
	}
	if replyStr, ok := reply.(string); ok && strings.ToLower(replyStr) == "ok" {
		return 1, nil
	}
	return redis.Int64(reply, err)
}

func (c *Client) Eval(ctx context.Context, src string, keyCount int, keysAndArgs []interface{}) (interface{}, error) {
	args := make([]interface{}, 2+len(keysAndArgs))
	args[0] = src
	args[1] = keyCount
	copy(args[2:], keysAndArgs)
	conn, err := c.pools.GetContext(ctx)
	if err != nil {
		return -1, err
	}
	defer conn.Close()
	return conn.Do("EVAL", args...)
}
