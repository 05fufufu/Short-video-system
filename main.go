package main

import (
	"fmt"
	"tiktok-server/config"
	"tiktok-server/handlers"

	"github.com/gin-gonic/gin"
)

func main() {
	// 1. 初始化配置
	config.InitDB()
	config.InitMinIO()

	// 2. 初始化路由
	r := gin.Default()
	r.MaxMultipartMemory = 100 << 20 // 100MB

	// 3. 注册路由
	// 把逻辑都移到了 handlers 包里，这里只负责分配路径
	r.POST("/publish/action", handlers.PublishAction)

	// 4. 启动
	fmt.Println("🚀 服务已启动: http://localhost:8080")
	r.Run(":8080")
}
