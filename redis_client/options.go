package redis_client

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
