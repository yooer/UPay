package rdb

import (
	"context"
	"fmt"
	"os"
	"time"
	"upay_pro/db/sdb"
	"upay_pro/mylog"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

var RDB *redis.Client

func init() {
	// 如果不是工作进程，则不初始化 Redis 客户端
	if os.Getenv("UPAY_IS_WORKER") != "1" {
		return
	}

	// 创建 Redis 客户端
	rdb := redis.NewClient(&redis.Options{
		// 基本连接配置
		Addr:     fmt.Sprintf("%s:%d", sdb.GetSetting().Redishost, sdb.GetSetting().Redisport), // Redis 地址
		Password: sdb.GetSetting().Redispasswd,                                                 // Redis 密码
		DB:       sdb.GetSetting().Redisdb,                                                     // 数据库编号

		// 连接超时设置
		DialTimeout:  10 * time.Second, // 建立连接超时时间
		ReadTimeout:  30 * time.Second, // 读取超时时间
		WriteTimeout: 30 * time.Second, // 写入超时时间

		// 连接池设置
		PoolSize:        10,               // 连接池最大连接数
		MinIdleConns:    5,                // 最小空闲连接数
		PoolTimeout:     4 * time.Second,  // 从连接池获取连接的超时时间
		ConnMaxLifetime: 30 * time.Minute, // 连接的最大存活时间（替代 MaxConnAge）
		ConnMaxIdleTime: 5 * time.Minute,  // 空闲连接超时时间（替代 IdleTimeout）

		// 其他设置
		OnConnect: func(ctx context.Context, cn *redis.Conn) error {
			// 连接建立时的回调函数
			return nil
		},
	})
	ctx := context.Background()
	RDB = rdb
	// defer rdb.Close()  在其他调用时最后关闭
	// 测试连接
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		// redis 连接失败写入日志，但不 panic 挂起
		mylog.Logger.Error("redis 连接失败，系统将限制收款相关功能", zap.Error(err))
	} else {
		// 测试redis是否连接成功 写入日志
		mylog.Logger.Info("redis 连接成功")
	}

	/* 	// 测试访问不存在的键
	   	_, err = rdb.Get(ctx, "520").Result()
	   	if err != nil {
	   		log.Logger.Info("redis 访问不存在的键")
	   	} */

}

// IsConnected 动态检测 Redis 连通性
func IsConnected() bool {
	if RDB == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	_, err := RDB.Ping(ctx).Result()
	return err == nil
}

// Close 优雅关闭 Redis 连接
