package config

import (
	"log"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// 全局变量，其他包直接用 config.DB 或 config.MinioClient 访问
var (
	DB          *gorm.DB
	MinioClient *minio.Client
)

// 常量配置 (以后可以从配置文件读取)
const (
	MinioEndpoint  = "127.0.0.1:9000"
	MinioAccessKey = "admin"
	MinioSecretKey = "password123"
	MinioBucket    = "videos"

	// 注意端口 3307
	MysqlDSN = "root:rootpassword@tcp(127.0.0.1:3307)/tiktok_db?charset=utf8mb4&parseTime=True&loc=Local"
)

func InitDB() {
	var err error
	DB, err = gorm.Open(mysql.Open(MysqlDSN), &gorm.Config{})
	if err != nil {
		log.Fatal("❌ MySQL 连接失败: ", err)
	}
	log.Println("✅ MySQL 连接成功")
}

func InitMinIO() {
	var err error
	MinioClient, err = minio.New(MinioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(MinioAccessKey, MinioSecretKey, ""),
		Secure: false,
	})
	if err != nil {
		log.Fatal("❌ MinIO 连接失败: ", err)
	}
	log.Println("✅ MinIO 连接成功")
}
