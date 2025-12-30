package config

import (
	"log"

	"github.com/streadway/amqp"
)

var MQConn *amqp.Connection
var MQChannel *amqp.Channel

func InitRabbitMQ() {
	var err error
	// 连接 Linux 的 RabbitMQ
	url := "amqp://guest:guest@172.20.10.6:5672/"
	MQConn, err = amqp.Dial(url)
	if err != nil {
		log.Fatal("❌ RabbitMQ 连接失败: ", err)
	}

	MQChannel, err = MQConn.Channel()
	if err != nil {
		log.Fatal("❌ RabbitMQ Channel 创建失败: ", err)
	}

	// 声明转码队列
	_, err = MQChannel.QueueDeclare("transcode_queue", true, false, false, false, nil)
	if err != nil {
		log.Fatal("❌ 转码队列声明失败: ", err)
	}

	// 声明点赞队列
	_, err = MQChannel.QueueDeclare("like_queue", true, false, false, false, nil)
	if err != nil {
		log.Fatal("❌ 点赞队列声明失败: ", err)
	}

	log.Println("✅ RabbitMQ 连接成功")
}
