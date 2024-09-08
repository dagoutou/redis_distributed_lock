package redis_distributed_lock

const DefaultLockExpireSeconds = 30

type ClientOptions struct {
	maxIdle           int
	maxActive         int
	wait              bool
	idleTimeoutSecond int

	// 必填参数
	network  string
	address  string
	password string
}

type ClientOption func(r *ClientOptions)

func WithMaxIdle(maxIdle int) ClientOption {
	return func(r *ClientOptions) {
		r.maxIdle = maxIdle
	}
}

func WithMaxActive(maxActive int) ClientOption {
	return func(r *ClientOptions) {
		r.maxActive = maxActive
	}
}

func WithWait(wait bool) ClientOption {
	return func(r *ClientOptions) {
		r.wait = wait
	}
}

func WithIdleTimeSecond(idleTimeoutSecond int) ClientOption {
	return func(r *ClientOptions) {
		r.idleTimeoutSecond = idleTimeoutSecond
	}
}

type RedisLockOptions struct {
	expireSeconds       int64
	isBlock             bool  // 是否是阻塞模式
	blockWaitingSeconds int64 // 能阻塞等待的时间
	watchDogMode        bool  // 是否为看门狗模式
}

type RedisLockOption func(opt *RedisLockOptions)

func WithExpireSeconds(expireSeconds int64) RedisLockOption {
	return func(opt *RedisLockOptions) {
		opt.expireSeconds = expireSeconds
	}
}

func repairLock(r *RedisLockOptions) {
	if r.isBlock && r.blockWaitingSeconds <= 0 {
		r.blockWaitingSeconds = 5
	}
	// 如果不设置锁过期时间自动启动看门狗
	if r.expireSeconds > 0 {
		return
	}
	r.expireSeconds = DefaultLockExpireSeconds
	r.watchDogMode = true
}
