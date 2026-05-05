# GeekEdu - 在线视频学习平台

基于微服务架构的在线视频学习平台，采用前后端分离模式。核心业务为利用阿里云 OSS 和微服务权限控制，实现付费视频内容的保护与分发。

## 技术栈

- **后端**: Go (Gin + gRPC), GORM, JWT
- **前端**: React 18, TypeScript, Ant Design 5, Vite
- **存储**: MySQL 8.0, 阿里云 OSS
- **运维**: Docker, Docker Compose, Nginx

## 系统架构

```
浏览器 → Nginx(:80) → 静态资源 / 反代 /api
                          ↓
                    Web Server(:8080) [Gin, JWT鉴权]
                          ↓ gRPC
                    Logic Server(:9090) [业务逻辑, GORM]
                      ↓           ↓
                  MySQL(:3306)  阿里云 OSS(私有Bucket)
```

## 项目结构

```
geekedu/
├── web-server/          # HTTP API 网关 (Gin)
├── logic-server/        # 业务逻辑服务 (gRPC)
├── common/              # 公共组件 (Config, JWT, 错误码, 响应)
├── proto/               # Protobuf 定义文件
├── frontend/            # 前端 React 应用
├── deploy/              # 部署配置
│   ├── docker-compose.yaml
│   ├── .env             # 环境变量 (OSS Key 等)
│   ├── mysql/init.sql   # 数据库初始化
│   └── nginx/nginx.conf
└── README.md
```

## 快速启动

### 1. 配置 OSS

编辑 `deploy/.env`，填入真实的阿里云 OSS 配置：

```
OSS_ACCESS_KEY=你的AccessKey
OSS_SECRET_KEY=你的SecretKey
OSS_ENDPOINT=oss-cn-hangzhou.aliyuncs.com
OSS_BUCKET=你的Bucket名称
JWT_SECRET=自定义JWT密钥
```

### 2. 一键启动

```bash
cd deploy
docker-compose up --build
```

### 3. 访问

- 前端: http://localhost
- API: http://localhost:8080

### 默认管理员账号

- 用户名: `admin`
- 密码: `admin123`

## 功能说明

### 用户角色

| 角色 | 说明 |
|------|------|
| 学员 (Student) | 注册账号，浏览课程，购买课程，观看已购视频 |
| 管理员 (Admin) | 发布课程，上传视频，拥有学员所有功能 |

### API 接口

| 方法 | 路径 | 描述 | 权限 |
|------|------|------|------|
| POST | /api/v1/auth/register | 用户注册 | 公开 |
| POST | /api/v1/auth/login | 用户登录 | 公开 |
| GET | /api/v1/courses | 课程列表 | 公开 |
| GET | /api/v1/courses/:id | 课程详情 | 公开 |
| POST | /api/v1/courses | 创建课程 | 管理员 |
| POST | /api/v1/orders | 购买课程 | 登录 |
| GET | /api/v1/player/:video_id | 获取播放地址 | 登录+已购买 |
| POST | /api/v1/courses/:id/videos/init | 初始化分片上传 | 管理员 |
| POST | /api/v1/courses/:id/videos/complete | 完成分片上传 | 管理员 |

### 核心流程：视频播放鉴权

1. 学员请求播放视频 → 携带 JWT Token
2. Web Server 验证 JWT，提取用户 ID
3. Logic Server 查询订单表，验证该用户是否已购买对应课程
4. 已购买 → 调用 OSS SDK 生成预签名 URL（有效期 3600 秒）
5. 前端使用签名 URL 播放视频
6. 直接访问 OSS 原始地址返回 Access Denied

### 分片上传流程

1. 管理员选择文件 → 前端计算分片数（每片 5MB）
2. 调用后端初始化分片上传 → 获取各分片的预签名上传 URL
3. 前端逐片 PUT 到 OSS → 收集每片 ETag
4. 调用后端完成合并 → 视频记录写入数据库

## 数据库设计

| 表名 | 说明 |
|------|------|
| users | 用户表 (username, password, role) |
| courses | 课程表 (title, description, cover_key, price, teacher_id) |
| course_videos | 视频表 (course_id, title, video_key) |
| orders | 订单表 (user_id, course_id, price, status) |

## 配置管理

所有敏感配置通过环境变量注入，支持 config.yaml 本地开发配置：

- `OSS_ACCESS_KEY` / `OSS_SECRET_KEY`: 阿里云 OSS 凭证
- `DB_HOST` / `DB_PASSWORD`: 数据库连接
- `JWT_SECRET`: JWT 签名密钥
- `LOGIC_SERVER_ADDR`: Logic Server gRPC 地址
