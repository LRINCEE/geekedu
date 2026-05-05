-- GeekEdu 数据库初始化脚本

CREATE DATABASE IF NOT EXISTS geekedu DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

USE geekedu;

-- 用户表
CREATE TABLE IF NOT EXISTS users (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(64) NOT NULL,
    password VARCHAR(256) NOT NULL COMMENT 'bcrypt hashed',
    role TINYINT NOT NULL DEFAULT 0 COMMENT '0=Student, 1=Admin',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE INDEX idx_username (username)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 课程表
CREATE TABLE IF NOT EXISTS courses (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    title VARCHAR(256) NOT NULL,
    description TEXT,
    cover_key VARCHAR(512) NOT NULL DEFAULT '' COMMENT 'OSS object key for cover image',
    price DECIMAL(10,2) NOT NULL DEFAULT 0.00,
    teacher_id BIGINT UNSIGNED NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_teacher_id (teacher_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 课程视频表
CREATE TABLE IF NOT EXISTS course_videos (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    course_id BIGINT UNSIGNED NOT NULL,
    title VARCHAR(256) NOT NULL,
    video_key VARCHAR(512) NOT NULL COMMENT 'OSS object key for video file',
    duration INT UNSIGNED NOT NULL DEFAULT 0 COMMENT 'duration in seconds',
    sort_order INT NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_course_id (course_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 订单表
CREATE TABLE IF NOT EXISTS orders (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT UNSIGNED NOT NULL,
    course_id BIGINT UNSIGNED NOT NULL,
    price DECIMAL(10,2) NOT NULL DEFAULT 0.00 COMMENT 'price snapshot',
    status TINYINT NOT NULL DEFAULT 1 COMMENT '1=completed',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE INDEX idx_user_course (user_id, course_id),
    INDEX idx_user_id (user_id),
    INDEX idx_course_id (course_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 预设管理员账号: admin / admin123
INSERT INTO users (username, password, role) VALUES
('admin', '$2a$10$8fIj.rO7HlXxnkRyxOCHYuWbOL9IyA3RN3BkNhiexm3JlBSkoGCra', 1);
