package config

import (
	"log"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinioClient 全局 MinIO 客户端
var MinioClient *minio.Client

const (
	// 1. 内部连接地址：Windows 后端程序连接 Linux 虚拟机用的地址
	// 用于后端直接上传、下载原始视频，走局域网速度快且稳定
	MinioInnerEndpoint = "172.20.10.2:9000"

	// 2. 公网访问域名：返回给别人浏览器看视频用的域名
	// 🌟 重点：这里填 cpolar 给你的那个 8080 端口的公网地址 (去掉 http://)
	// 例子：如果 cpolar 地址是 http://magic-girl.cpolar.top，这里就填 magic-girl.cpolar.top
	MinioPublicServer = "1a253b7.r17.cpolar.top"

	// 3. 访问凭证（需与 Linux Docker 中的配置一致）
	MinioAccessKey = "admin"
	MinioSecretKey = "password123"
	MinioBucket    = "videos"
)

// InitMinIO 初始化 MinIO 连接
func InitMinIO() {
	var err error

	// 初始化 SDK 必须使用 MinioInnerEndpoint (172.20.10.2)
	// 因为你的 Go 代码和虚拟机在同一个热点/路由器下，内网直连是最稳妥的
	MinioClient, err = minio.New(MinioInnerEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(MinioAccessKey, MinioSecretKey, ""),
		Secure: false, // 免费版穿透或内网连接通常不使用 HTTPS
	})

	if err != nil {
		log.Fatal("❌ MinIO 连接失败: ", err)
	}

	log.Printf("✅ MinIO 连接成功 (存储节点: %s)", MinioInnerEndpoint)
}
