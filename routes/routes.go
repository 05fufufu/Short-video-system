package routes

import (
	"tiktok-server/handlers"

	"github.com/gin-gonic/gin"
)

func InitRouter() *gin.Engine {
	r := gin.Default()

	// CORS 跨域
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	r.MaxMultipartMemory = 100 << 20
	r.StaticFile("/", "./index.html") // 直接托管前端

	r.POST("/publish/action", handlers.PublishAction)
	r.GET("/feed", handlers.FeedAction)
	r.POST("/favorite/action", handlers.FavoriteAction)
	// 评论接口
	r.POST("/comment/action", handlers.CommentAction)
	r.GET("/comment/list", handlers.CommentList)
	return r
}
