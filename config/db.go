package config

import (
	"fmt"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var (
	DB      *gorm.DB   // 业务主库
	UserDBs []*gorm.DB // 用户分片库 [0, 1]
)

func InitDB() {
	// 注意修改为你的 Linux 虚拟机 IP
	dsnFormat := "root:rootpassword@tcp(172.20.10.2:3307)/%s?charset=utf8mb4&parseTime=True&loc=Local"

	var err error
	// 1. 初始化业务主库 (tiktok_db)
	DB, err = gorm.Open(mysql.Open(fmt.Sprintf(dsnFormat, "tiktok_db")), &gorm.Config{})
	if err != nil {
		log.Fatal("❌ 业务主库连接失败: ", err)
	}

	// 2. 初始化 2 个用户分片库 (tiktok_user_0, tiktok_user_1)
	UserDBs = make([]*gorm.DB, 2)
	for i := 0; i < 2; i++ {
		dbName := fmt.Sprintf("tiktok_user_%d", i)
		udb, err := gorm.Open(mysql.Open(fmt.Sprintf(dsnFormat, dbName)), &gorm.Config{})
		if err != nil {
			log.Fatal("❌ 分片库连接失败: ", dbName)
		}
		UserDBs[i] = udb
	}
	log.Println("✅ 分库分表数据库初始化成功")
}

// GetUserDB 根据用户 ID 取模路由到对应的库
func GetUserDB(userID int64) *gorm.DB {
	return UserDBs[int(userID%2)]
}
