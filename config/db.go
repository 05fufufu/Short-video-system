package config

import (
	"fmt"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/plugin/dbresolver"
)

var (
	DB      *gorm.DB   // 业务主库
	UserDBs []*gorm.DB // 用户分片库 [0, 1]
)

func InitDB() {
	// 注意修改为你的 Linux 虚拟机 IP
	// 端口占位符 %s, 数据库名占位符 %s
	dsnFormat := "root:rootpassword@tcp(172.20.10.3:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local"

	var err error
	// 1. 初始化业务主库 (tiktok_db)
	masterDSN := fmt.Sprintf(dsnFormat, "3307", "tiktok_db")
	slaveDSN := fmt.Sprintf(dsnFormat, "3308", "tiktok_db")

	DB, err = gorm.Open(mysql.Open(masterDSN), &gorm.Config{})
	if err != nil {
		log.Fatal("❌ 业务主库连接失败: ", err)
	}

	// 配置读写分离
	err = DB.Use(dbresolver.Register(dbresolver.Config{
		Sources:  []gorm.Dialector{mysql.Open(masterDSN)},
		Replicas: []gorm.Dialector{mysql.Open(slaveDSN)},
		Policy:   dbresolver.RandomPolicy{},
	}))
	if err != nil {
		log.Println("⚠️ 读写分离配置失败 (可能是从库未启动): ", err)
	}

	// 2. 初始化 2 个用户分片库 (tiktok_user_0, tiktok_user_1)
	UserDBs = make([]*gorm.DB, 2)
	for i := 0; i < 2; i++ {
		dbName := fmt.Sprintf("tiktok_user_%d", i)
		mDSN := fmt.Sprintf(dsnFormat, "3307", dbName)
		sDSN := fmt.Sprintf(dsnFormat, "3308", dbName)

		udb, err := gorm.Open(mysql.Open(mDSN), &gorm.Config{})
		if err != nil {
			log.Fatal("❌ 分片库连接失败: ", dbName)
		}

		// 为分片库也配置读写分离
		udb.Use(dbresolver.Register(dbresolver.Config{
			Sources:  []gorm.Dialector{mysql.Open(mDSN)},
			Replicas: []gorm.Dialector{mysql.Open(sDSN)},
			Policy:   dbresolver.RandomPolicy{},
		}))

		UserDBs[i] = udb
	}
	log.Println("✅ 分库分表 + 读写分离 初始化成功")
}

// GetUserDB 根据用户 ID 取模路由到对应的库
func GetUserDB(userID int64) *gorm.DB {
	return UserDBs[int(userID%2)]
}
