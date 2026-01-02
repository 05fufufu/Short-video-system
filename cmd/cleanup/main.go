package main

import (
	"context"
	"fmt"
	"log"
	"tiktok-server/config"

	"github.com/minio/minio-go/v7"
)

func main() {
	// 1. 初始化所有连接
	config.InitDB()
	config.InitMinIO()
	config.InitRedis()
	config.InitRabbitMQ()

	ctx := context.Background()
	fmt.Println("🚀 开始清理所有数据...")

	// 2. 清空 MySQL 表 (需临时关闭外键检查)
	config.DB.Exec("SET FOREIGN_KEY_CHECKS = 0")
	tables := []string{"videos", "notes", "comments", "likes", "notifications"}
	for _, table := range tables {
		if err := config.DB.Exec("TRUNCATE TABLE " + table).Error; err != nil {
			log.Printf("⚠️ 清理表 %s 失败: %v", table, err)
		} else {
			fmt.Printf("✅ 表 %s 已清空\n", table)
		}
	}
	config.DB.Exec("SET FOREIGN_KEY_CHECKS = 1")

	// 3. 清空 MinIO 文件 (视频和封面)
	objectsCh := make(chan minio.ObjectInfo)
	go func() {
		defer close(objectsCh)
		for object := range config.MinioClient.ListObjects(ctx, config.MinioBucket, minio.ListObjectsOptions{Recursive: true}) {
			objectsCh <- object
		}
	}()

	// 批量删除
	opts := minio.RemoveObjectsOptions{GovernanceBypass: true}
	for err := range config.MinioClient.RemoveObjects(ctx, config.MinioBucket, objectsCh, opts) {
		log.Println("⚠️ 删除文件出错:", err)
	}
	fmt.Println("✅ MinIO 存储桶已清空")

	// 4. 清空 Redis 缓存
	config.RDB.FlushDB(ctx)
	fmt.Println("✅ Redis 缓存已清空")

	// 5. 清空 RabbitMQ 队列
	queues := []string{"transcode_queue", "like_queue"}
	for _, q := range queues {
		config.MQChannel.QueuePurge(q, false)
	}
	fmt.Println("✅ 消息队列已清空")

	fmt.Println("\n🎉 清理完成！")
}
