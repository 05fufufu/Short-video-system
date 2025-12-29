[README.md](https://github.com/user-attachments/files/24371180/README.md)
# 🌸 魔法少女的视频网站 (Magic TikTok)

这是一个基于 **Go (Gin)** + **Vue 3** + **Microservices** 架构的短视频平台。
项目包含完整的视频上传、转码（HLS）、弹幕/评论、点赞及用户系统。

## 🛠️ 技术栈

*   **后端**: Go 1.20+, Gin, Gorm
*   **前端**: Vue 3, Element Plus, Hls.js
*   **中间件**: MySQL 8.0, Redis 7.0, RabbitMQ, MinIO (对象存储)
*   **工具**: FFmpeg (视频转码), Docker Compose

## 📋 环境要求

在开始之前，请确保你的电脑已安装：
1.  **Go** (版本 >= 1.20)
2.  **Docker** & **Docker Compose** (用于运行数据库和中间件)
3.  **FFmpeg** (必须添加到系统环境变量 PATH 中，用于视频转码)
4.  **Git**

---

## 🚀 快速开始 (复现实验)

### 1. 克隆项目
```bash
git clone <repository_url>
cd Short-video-system
```

### 2. 启动基础环境
使用 Docker Compose 一键启动 MySQL, Redis, MinIO。
```bash
docker-compose up -d
```
> ⚠️ **注意**: 首次启动可能需要几分钟下载镜像。确保 3307, 6379, 9000, 9001 端口未被占用。

### 3. 配置 MinIO (重要!)
由于 MinIO 是新启动的，你需要手动创建一个存储桶。
1.  浏览器访问: `http://localhost:9001`
2.  账号: `admin` / 密码: `password123`
3.  点击 **Buckets** -> **Create Bucket**。
4.  Bucket Name 输入: `videos` -> 点击 Create。
5.  (可选) 在 **Manage** -> **Access Policy** 中将 Access Policy 设置为 `Public` (虽然代码中有代理接口，但设为 Public 方便调试)。

### 4. 运行后端服务
```bash
# 下载依赖
go mod tidy

# 运行项目
go run main.go
```
如果看到 `🎉 服务启动成功: http://localhost:8080`，说明后端已就绪。

### 5. 访问项目
打开浏览器访问: `http://localhost:8080`

---

## ⚙️ 配置说明

### 切换 局域网/公网 模式
默认配置可能使用了内网穿透域名。如果你在本地或局域网演示，请修改配置：

1.  打开 `config/minio.go`。
2.  找到 `MinioPublicServer` 变量。
3.  **本地/局域网模式**:
    ```go
    // 使用你的局域网 IP (推荐) 或 localhost
    MinioPublicServer = "172.20.10.4:8080" 
    ```
4.  **公网穿透模式 (cpolar)**:
    ```go
    MinioPublicServer = "你的域名.cpolar.top"
    ```
5.  **重启后端服务**使配置生效。

> 💡 前端 `index.html` 已配置为自动适配域名，无需手动修改。

---

## 📂 目录结构
```
.
├── config/          # 数据库、MinIO、Redis 配置
├── handlers/        # 业务逻辑控制层 (Controller)
├── models/          # GORM 数据模型
├── routes/          # 路由注册
├── service/         # 后台服务 (转码 Worker, Token 生成)
├── sql/             # 数据库初始化脚本
├── docker-compose.yml # 中间件编排
├── main.go          # 项目入口
└── index.html       # 前端单页应用
```

## ⚠️ 常见问题

1.  **上传视频后无法播放？**
    *   检查 FFmpeg 是否安装并配置了环境变量。
    *   检查后端控制台是否有转码报错。
    *   确保 MinIO 中 `videos` 桶已创建。

2.  **图片/头像不显示？**
    *   检查 `config/minio.go` 中的 `MinioPublicServer` 是否与你当前访问的浏览器地址栏域名一致。

3.  **数据库连接失败？**
    *   检查 Docker 容器是否正常运行 (`docker ps`)。
    *   默认 MySQL 端口映射为 `3307` (防止与本地 3306 冲突)，请确认代码中连接的是 3307。
