package redis_distributed_lock

import (
	"context"
	"go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
	"log"
	"time"
)

func initEtcdLock() {
	client, err := clientv3.New(clientv3.Config{Endpoints: []string{"localhost:2379"}, DialTimeout: time.Second * 3})
	if err != nil {
		log.Fatal("初始化etcd客户端失败！")
	}

	// 创建一个租约为30s
	session, err := concurrency.NewSession(client, concurrency.WithTTL(30))
	if err != nil {
		log.Fatal("初始化租约失败！")
	}

	defer func(session *concurrency.Session) {
		err := session.Close()
		if err != nil {
			log.Fatalf("Session关闭失败:%v\n", err)
		}
	}(session)
	mutex := concurrency.NewMutex(session, LUADeleteLock)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := mutex.TryLock(ctx); err != nil {
		log.Fatalf("加锁失败立即返回:%v\n", err)
		return
	}

	log.Println("加锁成功开始执行业务")
	for i := 1; i <= 10; i++ {
		time.Sleep(time.Second)
		log.Printf("执行 %%%d ...", i*10)
	}

	// 释放锁
	err = mutex.Unlock(context.TODO())
	if err != nil {
		log.Fatalf("释放锁失败:%v\n", err)
		return
	}
	log.Println("释放锁完成")
}
