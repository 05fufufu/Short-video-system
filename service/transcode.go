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

	// 2. 转码 (HLS 切片)
	// ffmpeg -i input.mp4 -c:v libx264 -c:a aac -strict -2 -f hls -hls_list_size 0 -hls_time 10 output.m3u8
	cmd := exec.Command("ffmpeg", "-y", "-i", localRaw, "-c:v", "libx264", "-c:a", "aac", "-strict", "-2", "-f", "hls", "-hls_list_size", "0", "-hls_time", "5", "output.m3u8")
	if err := cmd.Run(); err != nil {
		log.Println("❌ FFmpeg HLS 转码失败:", err)
		return
	}

	// 3. 上传成品 (m3u8 + ts)
	// 先上传 m3u8
	m3u8Name := strings.Replace(msg.FileName, "raw/", "processed/", 1) + ".m3u8"
	config.MinioClient.FPutObject(ctx, config.MinioBucket, m3u8Name, "output.m3u8", minio.PutObjectOptions{ContentType: "application/x-mpegURL"})

	// 上传所有 ts 切片
	files, _ := os.ReadDir(".")
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".ts") {
			tsName := "processed/" + f.Name()
			config.MinioClient.FPutObject(ctx, config.MinioBucket, tsName, f.Name(), minio.PutObjectOptions{ContentType: "video/MP2T"})
			os.Remove(f.Name()) // 上传完删除本地 ts
		}
	}

	// 4. 入库
	playURL := fmt.Sprintf("http://%s/video_file/%s", config.MinioPublicServer, m3u8Name)

	video := models.Video{
		AuthorID: msg.AuthorID,
		Title:    msg.Title,
		PlayURL:  playURL,
		CoverURL: msg.CoverURL,
		Status:   1,
	}

	config.DB.Create(&video)
	log.Println("🎉 HLS 视频处理完成:", msg.Title)

	// 清理
	os.Remove(localRaw)
	os.Remove(localOut)
	os.Remove("output.m3u8")
}
