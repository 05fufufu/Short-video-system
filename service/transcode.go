package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
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
	localRaw := "temp_raw_" + filepath.Base(msg.FileName)
	outputDir := "output_" + strings.TrimSuffix(filepath.Base(msg.FileName), filepath.Ext(msg.FileName))

	// 清理工作
	defer os.Remove(localRaw)
	defer os.RemoveAll(outputDir)

	// 1. 下载原始视频
	err := config.MinioClient.FGetObject(ctx, config.MinioBucket, msg.FileName, localRaw, minio.GetObjectOptions{})
	if err != nil {
		log.Println("❌ 下载失败:", err)
		return
	}

	// 创建输出目录
	os.Mkdir(outputDir, 0755)

	// 2. 转码 - 生成 720P (高清)
	cmdHigh := exec.Command("ffmpeg", "-y", "-i", localRaw, "-vf", "scale=-2:720", "-c:v", "libx264", "-b:v", "1500k", "-c:a", "aac", "-f", "hls", "-hls_list_size", "0", "-hls_time", "5", "-hls_segment_filename", filepath.Join(outputDir, "high_%03d.ts"), filepath.Join(outputDir, "high.m3u8"))
	if err := cmdHigh.Run(); err != nil {
		log.Println("❌ FFmpeg 720P 转码失败:", err)
		return
	}

	// 3. 转码 - 生成 480P (标清)
	cmdLow := exec.Command("ffmpeg", "-y", "-i", localRaw, "-vf", "scale=-2:480", "-c:v", "libx264", "-b:v", "600k", "-c:a", "aac", "-f", "hls", "-hls_list_size", "0", "-hls_time", "5", "-hls_segment_filename", filepath.Join(outputDir, "low_%03d.ts"), filepath.Join(outputDir, "low.m3u8"))
	if err := cmdLow.Run(); err != nil {
		log.Println("❌ FFmpeg 480P 转码失败:", err)
		return
	}

	// 4. 生成 Master Playlist
	masterContent := "#EXTM3U\n#EXT-X-STREAM-INF:BANDWIDTH=1600000,RESOLUTION=1280x720\nhigh.m3u8\n#EXT-X-STREAM-INF:BANDWIDTH=700000,RESOLUTION=854x480\nlow.m3u8"
	os.WriteFile(filepath.Join(outputDir, "master.m3u8"), []byte(masterContent), 0644)

	// 5. 上传所有文件
	// 目标路径前缀: processed/文件名(无后缀)/
	baseName := strings.TrimSuffix(filepath.Base(msg.FileName), filepath.Ext(msg.FileName))
	// 注意：MinIO 路径必须用 /，不能用 filepath.Join (Windows下是反斜杠)
	remotePrefix := "processed/" + baseName + "/"

	files, _ := os.ReadDir(outputDir)
	for _, f := range files {
		localPath := filepath.Join(outputDir, f.Name())
		remotePath := remotePrefix + f.Name()
		
		contentType := "application/octet-stream"
		if strings.HasSuffix(f.Name(), ".m3u8") {
			contentType = "application/x-mpegURL"
		} else if strings.HasSuffix(f.Name(), ".ts") {
			contentType = "video/MP2T"
		}

		_, err := config.MinioClient.FPutObject(ctx, config.MinioBucket, remotePath, localPath, minio.PutObjectOptions{ContentType: contentType})
		if err != nil {
			log.Printf("❌ 上传文件 %s 失败: %v", f.Name(), err)
		}
	}

	// 6. 入库
	playURL := fmt.Sprintf("http://%s/video_file/%s", config.MinioPublicServer, remotePrefix+"master.m3u8")

	video := models.Video{
		AuthorID: msg.AuthorID,
		Title:    msg.Title,
		PlayURL:  playURL,
		CoverURL: msg.CoverURL,
		Status:   1,
	}

	config.DB.Create(&video)
	log.Println("🎉 多清晰度视频处理完成:", msg.Title)
}
