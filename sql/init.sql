-- 业务主库
CREATE DATABASE IF NOT EXISTS tiktok_db;
USE tiktok_db;

CREATE TABLE IF NOT EXISTS videos (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    author_id BIGINT,
    play_url VARCHAR(255),
    cover_url VARCHAR(255),
    title VARCHAR(255),
    status INT DEFAULT 0,
    favorite_count INT DEFAULT 0,
    created_at DATETIME
);

CREATE TABLE IF NOT EXISTS notes (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT,
    title VARCHAR(255),
    content TEXT,
    images TEXT,
    created_at DATETIME
);

CREATE TABLE IF NOT EXISTS user_login_map (
    username VARCHAR(255) PRIMARY KEY,
    user_id BIGINT
);

CREATE TABLE IF NOT EXISTS comments (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    video_id BIGINT,
    user_id BIGINT,
    content TEXT,
    created_at DATETIME
);

CREATE TABLE IF NOT EXISTS likes (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT,
    video_id BIGINT,
    created_at DATETIME,
    UNIQUE KEY idx_user_video (user_id, video_id)
);

-- 用户分片库
CREATE DATABASE IF NOT EXISTS tiktok_user_0;
CREATE DATABASE IF NOT EXISTS tiktok_user_1;

USE tiktok_user_0;
CREATE TABLE IF NOT EXISTS users (
    id BIGINT PRIMARY KEY,
    username VARCHAR(255),
    password VARCHAR(255),
    nickname VARCHAR(255),
    avatar VARCHAR(255),
    created_at DATETIME
);

USE tiktok_user_1;
CREATE TABLE IF NOT EXISTS users LIKE tiktok_user_0.users;
