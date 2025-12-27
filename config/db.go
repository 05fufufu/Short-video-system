package config

import (
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() {
	// 连接 Linux 虚拟机的 MySQL (端口 3307)
	dsn := "root:rootpassword@tcp(172.20.10.2:3307)/tiktok_db?charset=utf8mb4&parseTime=True&loc=Local"

	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("❌ MySQL 连接失败: ", err)
	}
	log.Println("✅ MySQL 连接成功 (172.20.10.2:3307)")
}
