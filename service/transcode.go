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
	"time"

	"github.com/minio/minio-go/v7"
)

// 保持结构体一致
type TranscodeMessage struct {
	FileName string `json:"file_name"`
	Title    string `json:"title"`
	AuthorID int64  `json:"author_id"`
	CoverURL string `json:"cover_url"` // 新增
}

type LikeMessage struct {
	UserID  int64 `json:"user_id"`
	VideoID int64 `json:"video_id"`
	NoteID  int64 `json:"note_id"`
	Action  int   `json:"action"`
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

	// 启动点赞 Worker
	likeMsgs, err := config.MQChannel.Consume("like_queue", "", true, false, false, false, nil)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		log.Println("❤️ 点赞 Worker 已启动...")
		for d := range likeMsgs {
			var msg LikeMessage
			json.Unmarshal(d.Body, &msg)
			processLike(msg)
		}
	}()
}

func processLike(msg LikeMessage) {
	// 1. 查是否已存在记录
	var like models.Like
	var err error
	
	if msg.NoteID > 0 {
		err = config.DB.Where("user_id = ? AND note_id = ?", msg.UserID, msg.NoteID).First(&like).Error
	} else {
		// 视频点赞：必须确保 note_id 为 0
		err = config.DB.Where("user_id = ? AND video_id = ? AND note_id = 0", msg.UserID, msg.VideoID).First(&like).Error
	}

	if msg.Action == 1 { // 点赞
		if err != nil { // 不存在则创建
			newLike := models.Like{
				UserID:    msg.UserID,
				VideoID:   msg.VideoID,
				NoteID:    msg.NoteID,
				CreatedAt: time.Now(),
				IsDeleted: 0,
			}
			if createErr := config.DB.Create(&newLike).Error; createErr != nil {
				log.Printf("❌ 点赞写入失败: %v", createErr)
				return
			}
			sendLikeNotification(msg)
		} else { // 存在则恢复
			if updateErr := config.DB.Model(&like).Update("is_deleted", 0).Error; updateErr != nil {
				log.Printf("❌ 点赞恢复失败: %v", updateErr)
				return
			}
			sendLikeNotification(msg)
		}
	} else { // 取消点赞
		if err == nil {
			config.DB.Model(&like).Update("is_deleted", 1)
		}
	}
}

func sendLikeNotification(msg LikeMessage) {
	var authorID int64
	if msg.NoteID > 0 {
		var note models.Note
		if err := config.DB.Select("user_id").First(&note, msg.NoteID).Error; err != nil {
			log.Printf("⚠️ 查不到笔记(ID:%d)作者，无法发送通知: %v", msg.NoteID, err)
			return
		}
		authorID = note.UserID
	} else {
		var video models.Video
		if err := config.DB.Select("author_id").First(&video, msg.VideoID).Error; err != nil {
			log.Printf("⚠️ 查不到视频(ID:%d)作者，无法发送通知: %v", msg.VideoID, err)
			return
		}
		authorID = video.AuthorID
	}

	if authorID != 0 && authorID != msg.UserID {
		notif := models.Notification{
			UserID:     authorID,
			SenderID:   msg.UserID,
			ActionType: 1, // like
			VideoID:    msg.VideoID,
			NoteID:     msg.NoteID,
			CreatedAt:  time.Now(),
			IsRead:     0,
		}
		if err := config.DB.Create(&notif).Error; err != nil {
			log.Printf("❌ 通知创建失败: %v", err)
		}
	}
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
