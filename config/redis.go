package config

import (
	"context"
	"log"

	"github.com/go-redis/redis/v8"
)

var RDB *redis.Client
var Ctx = context.Background()

func InitRedis() {
	RDB = redis.NewClient(&redis.Options{
		Addr:     "172.20.10.6:6379", // Linux IP
		Password: "",
		DB:       0,
	})

	_, err := RDB.Ping(Ctx).Result()
	if err != nil {
		log.Fatal("❌ Redis 连接失败: ", err)
	}
	log.Println("✅ Redis 连接成功")
}
