package config

import (
	"log"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var MinioClient *minio.Client

const (
	// 这里必须填 Linux 的 IP，否则 Windows 连不上
	MinioEndpoint  = "172.20.10.2:9000"
	MinioAccessKey = "admin"
	MinioSecretKey = "password123"
	MinioBucket    = "videos"
)

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
