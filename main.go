package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// ================= 全局变量 =================
var (
	DB          *gorm.DB
	MinioClient *minio.Client
)

// ================= 配置信息 =================
const (
	// MinIO 配置
	MinioEndpoint  = "127.0.0.1:9000" // 注意端口是 9000
	MinioAccessKey = "admin"
	MinioSecretKey = "password123"
	MinioBucket    = "videos" // 确保存储桶名字和你创建的一样

	// MySQL 配置 (注意端口 3307)
	DSN = "root:rootpassword@tcp(127.0.0.1:3307)/tiktok_db?charset=utf8mb4&parseTime=True&loc=Local"
)

// ================= 数据库模型 =================
type Video struct {
	ID        int64     `json:"id"`
	AuthorID  int64     `json:"author_id"`
	PlayURL   string    `json:"play_url"`
	CoverURL  string    `json:"cover_url"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}

// ================= 初始化函数 =================
func initDB() {
	var err error
	DB, err = gorm.Open(mysql.Open(DSN), &gorm.Config{})
	if err != nil {
		log.Fatal("❌ 数据库连接失败: ", err)
	}
	fmt.Println("✅ MySQL 连接成功！")
}

func initMinIO() {
	var err error
	MinioClient, err = minio.New(MinioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(MinioAccessKey, MinioSecretKey, ""),
		Secure: false, // 本地没 HTTPS，必须关掉
	})
	if err != nil {
		log.Fatal("❌ MinIO 连接失败: ", err)
	}
	fmt.Println("✅ MinIO 连接成功！")
}

// ================= 上传接口逻辑 =================
func uploadHandler(c *gin.Context) {
	// 1. 获取表单中的文件
	file, header, err := c.Request.FormFile("data") // Postman 里的 key 必须叫 "data"
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件上传失败"})
		return
	}

	// 2. 生成唯一文件名 (防止重名覆盖)
	// 格式: 1735282222_filename.mp4
	filename := fmt.Sprintf("%d_%s", time.Now().Unix(), header.Filename)

	// 3. 上传到 MinIO
	ctx := context.Background()
	info, err := MinioClient.PutObject(ctx, MinioBucket, filename, file, header.Size, minio.PutObjectOptions{
		ContentType: "video/mp4",
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "MinIO 上传失败: " + err.Error()})
		return
	}

	// 4. 生成访问 URL
	// http://localhost:9000/videos/xxx.mp4
	playURL := fmt.Sprintf("http://%s/%s/%s", MinioEndpoint, MinioBucket, filename)
	coverURL := "http://localhost:9000/images/default.jpg" // 暂时写死封面，后面再做截图

	// 5. 保存元数据到 MySQL
	newVideo := Video{
		AuthorID: 1, // 暂时写死作者ID
		PlayURL:  playURL,
		CoverURL: coverURL,
		Title:    c.PostForm("title"), // 获取标题
	}

	if err := DB.Create(&newVideo).Error; err != nil {
		//把具体的 err 打印出来
		c.JSON(http.StatusInternalServerError, gin.H{"error": "数据库保存失败: " + err.Error()})
		return
	}

	// 6. 返回成功响应
	c.JSON(http.StatusOK, gin.H{
		"message":   "上传成功！",
		"video_url": playURL,
		"size":      info.Size,
		"db_id":     newVideo.ID,
	})
}

func main() {
	initDB()
	initMinIO()

	r := gin.Default()

	// 限制上传大小 (默认为 32MB，短视频需要调大，比如 100MB)
	r.MaxMultipartMemory = 100 << 20

	// 注册上传路由
	r.POST("/publish/action", uploadHandler)

	fmt.Println("🚀 服务已启动: http://localhost:8080")
	r.Run(":8080")
}
