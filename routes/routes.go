package routes

import (
	"tiktok-server/handlers"

	"github.com/gin-gonic/gin"
)

func InitRouter() *gin.Engine {
	r := gin.Default()

	// 🌟 核心：跨域中间件（必须放在所有路由之前！）
	r.Use(func(c *gin.Context) {
		// 允许所有来源（开发环境最省事的方法）
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		// 允许的 Header
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, token")
		// 允许的方法
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		// 处理浏览器的预检请求 (OPTIONS)
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// 设置上传限制
	r.MaxMultipartMemory = 100 << 20

	// --- 路由注册 ---
	// 用户模块
	r.POST("/user/register", handlers.Register)
	r.POST("/user/login", handlers.Login)
	r.POST("/user/update_avatar", handlers.UpdateAvatar)
	r.GET("/user/info", handlers.GetUserInfo) // 获取用户信息接口

	// 视频模块
	r.GET("/feed", handlers.FeedAction)
	r.POST("/publish/action", handlers.PublishAction)
	r.POST("/publish/delete", handlers.DeleteAction)

	// 互动模块
	r.POST("/favorite/action", handlers.FavoriteAction)
	r.POST("/comment/action", handlers.CommentAction)
	r.GET("/comment/list", handlers.CommentList)

	// 托管前端
	r.StaticFile("/", "./index.html")

	return r
}
