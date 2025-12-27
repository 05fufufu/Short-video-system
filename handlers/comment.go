package handlers

import (
	"net/http"
	"strconv"
	"tiktok-server/config"
	"tiktok-server/models"

	"github.com/gin-gonic/gin"
)

// CommentWithUser 这是一个“聚合结构体”，用于给前端返回包含用户信息的评论
type CommentWithUser struct {
	ID           int64  `json:"id"`
	UserID       int64  `json:"user_id"`
	UserNickname string `json:"user_nickname"`
	UserAvatar   string `json:"user_avatar"`
	Content      string `json:"content"`
	CreateDate   string `json:"create_date"`
}

// CommentAction 处理发布评论请求
// 路由: POST /comment/action
func CommentAction(c *gin.Context) {
	// 1. 获取参数
	videoID, _ := strconv.ParseInt(c.Query("video_id"), 10, 64)
	commentText := c.Query("comment_text")
	userIDStr := c.Query("user_id") // 从参数获取当前操作者ID
	userID, _ := strconv.ParseInt(userIDStr, 10, 64)

	// 2. 基本校验
	if userID <= 0 {
		c.JSON(http.StatusOK, gin.H{"status_code": 1, "status_msg": "请先登录后再施法"})
		return
	}
	if videoID == 0 || commentText == "" {
		c.JSON(http.StatusOK, gin.H{"status_code": 1, "status_msg": "咒语不完整"})
		return
	}

	// 3. 构造评论模型并存入主库 (tiktok_db)
	newComment := models.Comment{
		UserID:  userID,
		VideoID: videoID,
		Content: commentText,
	}

	if err := config.DB.Create(&newComment).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"status_code": 1, "status_msg": "评论记录失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status_code": 0,
		"status_msg":  "咒语发送成功！",
	})
}

// CommentList 获取视频评论列表
// 路由: GET /comment/list
func CommentList(c *gin.Context) {
	videoID := c.Query("video_id")
	var comments []models.Comment

	// 1. 从主库（tiktok_db）查询该视频下的所有原始评论
	if err := config.DB.Where("video_id = ?", videoID).Order("created_at desc").Find(&comments).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"status_code": 1, "status_msg": "无法读取留言板"})
		return
	}

	// 2. 遍历评论，去对应的分片库“补全”用户信息
	finalList := make([]CommentWithUser, 0)
	for _, cmt := range comments {
		// 定义一个临时结构接收用户信息
		var userPart struct {
			Nickname string
			Avatar   string
		}

		// 🌟 核心逻辑：根据评论者 ID 定位到所属的数据库分片
		userDB := config.GetUserDB(cmt.UserID)
		userDB.Table("users").Select("nickname, avatar").Where("id = ?", cmt.UserID).First(&userPart)

		// 在分片库查询该用户的昵称和头像
		err := userDB.Table("users").
			Select("nickname, avatar").
			Where("id = ?", cmt.UserID).
			First(&userPart).Error

		// 如果找不到用户（可能是脏数据），给个默认显示
		if err != nil {
			userPart.Nickname = "已失踪的魔法使"
			userPart.Avatar = "https://via.placeholder.com/40/cccccc/ffffff?text=?"
		}

		// 3. 组装最终结果
		finalList = append(finalList, CommentWithUser{
			ID:           cmt.ID,
			UserID:       cmt.UserID,
			UserNickname: userPart.Nickname,
			UserAvatar:   userPart.Avatar,
			Content:      cmt.Content,
			// 格式化日期为友好格式
			CreateDate: cmt.CreatedAt.Format("2006-01-02 15:04"),
		})
	}

	// 4. 返回聚合后的结果
	c.JSON(http.StatusOK, gin.H{
		"status_code":  0,
		"status_msg":   "success",
		"comment_list": finalList,
	})
}
