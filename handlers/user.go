package handlers

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"tiktok-server/config"
	"tiktok-server/service"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"golang.org/x/crypto/bcrypt"
)

// UserLoginMap 对应主库的映射表
type UserLoginMap struct {
	Username string `gorm:"primaryKey"`
	UserID   int64
}

func (UserLoginMap) TableName() string { return "user_login_map" }

func Register(c *gin.Context) {
	username := c.Query("username")
	password := c.Query("password")

	// 1. 生成唯一 UserID (时间戳)
	userID := time.Now().UnixNano() / 1e6

	// 2. 密码加密
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	// 3. 写入主库映射表 (解决通过用户名找 ID 的问题)
	loginMap := UserLoginMap{Username: username, UserID: userID}
	if err := config.DB.Create(&loginMap).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"status_code": 1, "status_msg": "用户名已存在"})
		return
	}

	// 4. 写入用户分片库 (tiktok_user_0 或 1)
	db := config.GetUserDB(userID)
	userData := map[string]interface{}{
		"id":         userID,
		"username":   username,
		"password":   string(hashedPassword),
		"nickname":   "魔法使_" + username,
		"avatar":     "https://via.placeholder.com/100/ff9a9e/ffffff?text=User",
		"created_at": time.Now(),
	}

	if err := db.Table("users").Create(&userData).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"status_code": 1, "status_msg": "创建用户信息失败"})
		return
	}

	token, _ := service.GenerateToken(userID)
	c.JSON(http.StatusOK, gin.H{
		"status_code": 0,
		"user_id":     userID,
		"token":       token,
	})
}

func Login(c *gin.Context) {
	username := c.Query("username")
	password := c.Query("password")

	var loginMap UserLoginMap
	if err := config.DB.Where("username = ?", username).First(&loginMap).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"status_code": 1, "status_msg": "用户不存在"})
		return
	}

	var dbPassword string
	config.GetUserDB(loginMap.UserID).Table("users").Select("password").Where("id = ?", loginMap.UserID).Scan(&dbPassword)

	if err := bcrypt.CompareHashAndPassword([]byte(dbPassword), []byte(password)); err != nil {
		c.JSON(http.StatusOK, gin.H{"status_code": 1, "status_msg": "密码错误"})
		return
	}

	token, _ := service.GenerateToken(loginMap.UserID)
	c.JSON(http.StatusOK, gin.H{
		"status_code": 0,
		"user_id":     loginMap.UserID,
		"token":       token,
	})
}

// GetUserInfo 根据用户 ID 查询基本信息（支持分库查询）
func GetUserInfo(c *gin.Context) {
	userIDStr := c.Query("user_id")
	userID, _ := strconv.ParseInt(userIDStr, 10, 64)

	// 1. 根据 ID 定位到对应的分片库
	db := config.GetUserDB(userID)

	// 2. 查询信息
	var user struct {
		ID       int64  `json:"id"`
		Nickname string `json:"nickname"`
		Avatar   string `json:"avatar"`
	}

	// 从分片库的 users 表查
	if err := db.Table("users").Where("id = ?", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"status_code": 1, "status_msg": "找不到该魔法使"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status_code": 0,
		"user":        user,
	})
}

// UpdateAvatar 更新用户头像
func UpdateAvatar(c *gin.Context) {
	// 1. 获取参数
	userIDStr := c.PostForm("user_id")
	userID, _ := strconv.ParseInt(userIDStr, 10, 64)

	file, header, err := c.Request.FormFile("avatar")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status_code": 1, "status_msg": "图片获取失败"})
		return
	}

	// 2. 上传到 MinIO
	ctx := context.Background()
	ext := filepath.Ext(header.Filename)
	objectName := fmt.Sprintf("avatars/%d_%s%s", userID, time.Now().Format("20060102150405"), ext)

	_, err = config.MinioClient.PutObject(ctx, config.MinioBucket, objectName, file, header.Size, minio.PutObjectOptions{
		ContentType: "image/jpeg", // 简单设为jpeg，实际可动态获取
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status_code": 1, "status_msg": "存储魔法失败"})
		return
	}

	// 3. 生成新头像地址 (指向 Linux IP)
	avatarURL := fmt.Sprintf("http://%s/%s/%s", config.MinioEndpoint, config.MinioBucket, objectName)

	// 4. 更新对应的分片库
	db := config.GetUserDB(userID)
	if err := db.Table("users").Where("id = ?", userID).Update("avatar", avatarURL).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status_code": 1, "status_msg": "契约更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status_code": 0,
		"status_msg":  "头像更新成功",
		"avatar_url":  avatarURL,
	})
}
