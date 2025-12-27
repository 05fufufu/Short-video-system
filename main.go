package main

import (
	"fmt"
	"tiktok-server/config"
	"tiktok-server/routes"
	"tiktok-server/service"
)

func main() {
	// 1. 初始化所有基础设施 (连 Linux)
	config.InitDB()
	config.InitRedis()
	config.InitMinIO()
	config.InitRabbitMQ()

	// 2. 启动后台 Workers
	service.StartTranscodeWorker()
	service.StartLikeWorker()

	// 3. 启动 Web 服务
	r := routes.InitRouter()
	fmt.Println("🚀 服务启动成功: http://localhost:8080")
	r.Run(":8080")
}
