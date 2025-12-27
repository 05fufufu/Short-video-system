package handlers

import (
	"strconv"
	"tiktok-server/config"
	"tiktok-server/models"

	"github.com/gin-gonic/gin"
)

type CommentListResponse struct {
	StatusCode  int              `json:"status_code"`
	StatusMsg   string           `json:"status_msg"`
	CommentList []models.Comment `json:"comment_list"`
}

// 对应数据库模型的简单定义 (可以放在 models/comment.go，为了省事写这就行)
// 注意：需要在 models 文件夹下确认 Comment 结构体是否存在，或者直接在这里定义临时的
// 这里假设你 models 包里没定义，我们定义一个 Response 用的结构
type CommentResponse struct {
	ID         int64  `json:"id"`
	UserID     int64  `json:"user_id"`
	Content    string `json:"content"`
	CreateDate string `json:"create_date"`
}

// CommentAction 发送/删除评论
func CommentAction(c *gin.Context) {
	// 1. 获取参数
	userID := int64(1) // 暂时写死当前用户
	videoID, _ := strconv.ParseInt(c.Query("video_id"), 10, 64)
	actionType, _ := strconv.Atoi(c.Query("action_type")) // 1-发布 2-删除
	content := c.Query("comment_text")

	if videoID == 0 {
		c.JSON(400, gin.H{"status_code": 1, "status_msg": "参数错误"})
		return
	}

	// 2. 发布评论
	if actionType == 1 {
		comment := models.Comment{
			UserID:  userID,
			VideoID: videoID,
			Content: content,
		}
		if err := config.DB.Create(&comment).Error; err != nil {
			c.JSON(500, gin.H{"status_code": 1, "status_msg": "评论失败"})
			return
		}
		c.JSON(200, gin.H{"status_code": 0, "status_msg": "评论成功", "comment": comment})
		return
	}

	// 3. 删除评论 (略，课设通常只演示发布)
	c.JSON(200, gin.H{"status_code": 0, "status_msg": "暂不支持删除"})
}

// CommentList 获取评论列表
func CommentList(c *gin.Context) {
	videoID := c.Query("video_id")
	var comments []models.Comment

	// 倒序查询，最新的在前面
	if err := config.DB.Where("video_id = ?", videoID).Order("created_at desc").Find(&comments).Error; err != nil {
		c.JSON(500, gin.H{"status_code": 1, "status_msg": "查询失败"})
		return
	}

	c.JSON(200, gin.H{
		"status_code":  0,
		"status_msg":   "success",
		"comment_list": comments,
	})
}
