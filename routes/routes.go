package routes

import (
	"tiktok-server/handlers"

	"github.com/gin-gonic/gin"
)

func InitRouter() *gin.Engine {
	r := gin.Default()

	// 🌟 核心：跨域中间件（必须放在所有路由之前！）
	r.Use(func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if origin != "" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		}
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
	r.MaxMultipartMemory = 500 << 20

	// --- 路由注册 ---
	// 用户模块
	r.POST("/user/register", handlers.Register)
	r.POST("/user/login", handlers.Login)
	r.POST("/user/update_avatar", handlers.UpdateAvatar)
	r.POST("/user/update_background", handlers.UpdateBackgroundImage) // 新增背景更新接口
	r.GET("/user/info", handlers.GetUserInfo) // 获取用户信息接口

	// 视频模块
	r.GET("/feed", handlers.FeedAction)
	r.GET("/search", handlers.SearchAction) // 新增搜索接口
	r.POST("/publish/action", handlers.PublishAction)
	r.POST("/publish/delete", handlers.DeleteAction)
	r.GET("/publish/list", handlers.PublishList)
	// 注册一个代理接口，专门负责把外网请求转发给内网 MinIO
	r.GET("/video_file/*filepath", handlers.ProxyVideo)

	// 笔记模块
	r.POST("/note/publish", handlers.PublishNote)
	r.POST("/note/delete", handlers.DeleteNote)

	// 互动模块
	r.POST("/favorite/action", handlers.FavoriteAction)
	r.GET("/favorite/status", handlers.FavoriteStatus)
	r.GET("/user/likes_received", handlers.ReceivedLikesList)
	r.POST("/comment/action", handlers.CommentAction)
	r.GET("/comment/list", handlers.CommentList)
	r.GET("/notification/list", handlers.NotificationList)
	// 1. 托管背景图片 (让外网能访问到你本地的 bg.jpg)
	r.StaticFile("/bg.jpg", "./bg.jpg")

	// 托管前端
	r.StaticFile("/", "./index.html")

	return r
}
