package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"tiktok-server/config"
	"tiktok-server/models"

	"github.com/minio/minio-go/v7"
)

// 保持结构体一致
type TranscodeMessage struct {
	FileName string `json:"file_name"`
	Title    string `json:"title"`
	AuthorID int64  `json:"author_id"`
	CoverURL string `json:"cover_url"` // 新增
}

func StartTranscodeWorker() {
	msgs, err := config.MQChannel.Consume("transcode_queue", "", true, false, false, false, nil)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		log.Println("🔨 转码 Worker 已启动...")
		for d := range msgs {
			var msg TranscodeMessage
			json.Unmarshal(d.Body, &msg)
			processVideo(msg)
		}
	}()
}

func processVideo(msg TranscodeMessage) {
	ctx := context.Background()
	localRaw := "temp_raw.mp4"
	localOut := "temp_out.mp4"

	// 1. 下载
	err := config.MinioClient.FGetObject(ctx, config.MinioBucket, msg.FileName, localRaw, minio.GetObjectOptions{})
	if err != nil {
		log.Println("下载失败:", err)
		return
	}

	// 2. 转码
	cmd := exec.Command("ffmpeg", "-y", "-i", localRaw, "-vcodec", "libx264", "-s", "640x360", localOut)
	if err := cmd.Run(); err != nil {
		log.Println("❌ FFmpeg 失败:", err)
		return
	}

	// 3. 上传成品
	newObjName := strings.Replace(msg.FileName, "raw/", "processed/", 1)
	config.MinioClient.FPutObject(ctx, config.MinioBucket, newObjName, localOut, minio.PutObjectOptions{ContentType: "video/mp4"})

	// 4. 入库
	playURL := fmt.Sprintf("http://%s/video_file/%s", config.MinioPublicServer, newObjName)

	video := models.Video{
		AuthorID: msg.AuthorID,
		Title:    msg.Title,
		PlayURL:  playURL,
		CoverURL: msg.CoverURL, // 使用前端传来的封面！
		Status:   1,
	}

	config.DB.Create(&video)
	log.Println("🎉 视频处理完成:", msg.Title)

	// 清理
	os.Remove(localRaw)
	os.Remove(localOut)
}
