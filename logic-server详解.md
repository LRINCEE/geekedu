# Logic Server 详解

这份文档只讲 `logic-server`。目标是让没有后端基础的人也能明白：它是什么、启动后做了什么、每个模块负责什么、业务逻辑是怎么实现的、数据库和 Redis/OSS 是怎么用的。

## 1. Logic Server 是什么

`logic-server` 是这个项目真正处理业务的服务。

如果说 `web-server` 是“前台接待员”，那么 `logic-server` 就是“真正办业务的人”。

它不直接给浏览器提供 HTTP API，而是提供 gRPC 服务给 `web-server` 调用。

整体关系：

```text
前端
  |
  v
web-server
  |
  v gRPC
logic-server
  |
  +--> MySQL
  +--> Redis
  +--> 阿里云 OSS
```

`logic-server` 主要做这些事：

1. 用户注册。
2. 用户登录。
3. 密码 bcrypt 加密和校验。
4. JWT token 生成。
5. 创建课程。
6. 查询课程列表和课程详情。
7. 课程列表 Redis 缓存。
8. 创建订单。
9. 防止重复购买。
10. 校验用户是否购买课程。
11. 初始化视频分片上传。
12. 完成视频分片上传。
13. 生成视频播放预签名 URL。
14. 操作 MySQL、Redis、OSS。

一句话总结：

> `logic-server` 是业务核心层，负责真正的业务规则、数据读写、缓存、文件存储和权限校验。

## 2. Logic Server 用了哪些技术

| 技术 | 作用 |
|---|---|
| Go | 编写服务 |
| gRPC | 对 web-server 提供 RPC 接口 |
| protobuf | 定义 RPC 接口和消息 |
| GORM | 操作 MySQL |
| MySQL | 存用户、课程、视频、订单 |
| Redis | 课程列表缓存、订单分布式锁 |
| singleflight | 防止缓存击穿 |
| 阿里云 OSS SDK | 上传和播放文件 |
| bcrypt | 密码加密和校验 |
| JWT | 登录成功后生成 token |
| Zap | 日志 |
| Viper | 配置读取 |

## 3. logic-server 目录结构

```text
logic-server/
├── main.go
├── config.yaml
├── Dockerfile
├── go.mod
├── service/
│   ├── user_service.go
│   ├── course_service.go
│   ├── order_service.go
│   ├── video_service.go
│   ├── dependencies.go
│   └── adapters.go
├── dao/
│   ├── db.go
│   ├── redis.go
│   ├── user_dao.go
│   ├── course_dao.go
│   ├── order_dao.go
│   └── video_dao.go
├── model/
│   ├── user.go
│   ├── course.go
│   ├── order.go
│   └── video.go
└── oss/
    └── oss_client.go
```

每个目录的作用：

| 目录/文件 | 作用 |
|---|---|
| `main.go` | 启动 gRPC 服务 |
| `service/` | 业务逻辑 |
| `dao/` | 数据库访问 |
| `model/` | 数据表对应的 Go 结构体 |
| `oss/` | 阿里云 OSS 操作 |
| `Dockerfile` | 构建镜像 |

## 4. Logic Server 启动过程

入口文件：

```text
logic-server/main.go
```

启动过程可以分成 8 步。

### 4.1 读取配置

代码：

```go
config.InitConfig("config.yaml")
cfg := config.GetConfig()
```

配置来源包括：

1. `config.yaml`
2. 环境变量
3. 默认值

例如：

```text
DB_HOST
DB_PORT
DB_USER
DB_PASSWORD
REDIS_HOST
OSS_ACCESS_KEY
OSS_SECRET_KEY
JWT_SECRET
GRPC_PORT
```

### 4.2 初始化日志

代码：

```go
logger.InitLogger("dev", "logs/logic-server.log")
```

日志会写到：

```text
logic-server/logs/logic-server.log
```

### 4.3 初始化 MySQL

代码：

```go
dao.InitDB()
```

对应文件：

```text
logic-server/dao/db.go
```

它会：

1. 拼接 MySQL DSN。
2. 使用 GORM 连接 MySQL。
3. 最多重试 30 次，每次间隔 2 秒。
4. 配置连接池。

连接池参数：

```go
SetMaxIdleConns(10)
SetMaxOpenConns(100)
SetConnMaxLifetime(time.Hour)
```

### 4.4 初始化 Redis

代码：

```go
cacheEnabled := os.Getenv("CACHE_ENABLED") != "false"
if cacheEnabled {
    dao.InitRedis()
}
```

也就是说，如果没有显式设置：

```text
CACHE_ENABLED=false
```

就会启用 Redis。

Redis 初始化失败不会直接让服务崩溃，而是打印 warning。这样 Redis 挂了时，业务可以降级查 MySQL。

### 4.5 初始化 OSS

代码：

```go
ossutil.InitOSSClient()
```

OSS 用来存：

1. 课程封面图。
2. 课程视频。

OSS Bucket 应该是私有的。用户不能直接访问原始文件，必须通过后端生成的签名 URL。

### 4.6 创建 DAO

代码：

```go
courseRepo := dao.NewCourseDao()
videoRepo := dao.NewVideoDao()
orderRepo := dao.NewOrderDao()
userRepo := dao.NewUserDao()
```

DAO 是“数据库访问对象”，专门负责查表、插表。

### 4.7 创建 adapter

代码：

```go
cache = service.NewRedisAdapter(dao.RedisClient)
storage := service.NewOSSAdapter()
pwd := service.NewBcryptPasswordManager()
tokenProvider := service.NewJWTTokenProvider()
```

这些 adapter 把真实基础设施包装成 service 层需要的接口。

好处是 service 层不用直接依赖 Redis Client、OSS SDK、bcrypt、JWT 实现，测试时可以替换成 mock。

### 4.8 注册 gRPC 服务

代码：

```go
pb.RegisterUserServiceServer(...)
pb.RegisterCourseServiceServer(...)
pb.RegisterVideoServiceServer(...)
pb.RegisterOrderServiceServer(...)
```

注册 4 个服务：

```text
UserService
CourseService
VideoService
OrderService
```

默认监听端口：

```text
9090
```

## 5. gRPC 接口是怎么定义的

gRPC 接口定义在：

```text
proto/
```

生成后的 Go 代码在：

```text
common/pb/
```

项目按业务拆成 4 个 proto 文件。

### 5.1 用户服务

文件：

```text
proto/user.proto
```

接口：

```text
Register
Login
```

### 5.2 课程服务

文件：

```text
proto/course.proto
```

接口：

```text
CreateCourse
ListCourses
GetCourse
GetCoverUploadURL
```

### 5.3 订单服务

文件：

```text
proto/order.proto
```

接口：

```text
CreateOrder
CheckPurchase
```

### 5.4 视频服务

文件：

```text
proto/video.proto
```

接口：

```text
InitMultipartUpload
CompleteMultipartUpload
GetVideoPlayURL
```

## 6. service 层是什么

`service/` 是业务逻辑层。

它负责判断：

1. 参数是否合法。
2. 用户是否存在。
3. 密码是否正确。
4. 课程是否存在。
5. 用户是否已购买。
6. 是否允许播放视频。
7. 是否要查缓存。
8. 是否要写数据库。
9. 是否要调用 OSS。

文件对应关系：

| 文件 | 负责业务 |
|---|---|
| `user_service.go` | 注册、登录 |
| `course_service.go` | 课程创建、课程列表、课程详情、封面上传 URL |
| `order_service.go` | 创建订单、检查购买状态 |
| `video_service.go` | 视频上传、播放 URL |
| `dependencies.go` | 定义 service 依赖的接口 |
| `adapters.go` | 把 Redis/OSS/bcrypt/JWT 适配成接口 |

## 7. 为什么 service 层要定义接口

文件：

```text
logic-server/service/dependencies.go
```

里面定义了很多接口：

```go
type CourseRepository interface {}
type VideoRepository interface {}
type OrderRepository interface {}
type UserRepository interface {}
type Cache interface {}
type ObjectStorage interface {}
type PasswordManager interface {}
type TokenProvider interface {}
```

这叫“依赖倒置”。

简单理解：

```text
service 不关心你底层到底是 MySQL 还是 mock 数据库
service 只关心你有没有 CreateCourse、ListCourses 这些方法
```

好处：

1. 业务层更干净。
2. 更容易写单元测试。
3. 以后替换 Redis、OSS、数据库访问方式更方便。

比如测试时不需要真的连 MySQL，可以传一个假的 repository。

## 8. dao 层是什么

`dao/` 是数据库访问层。

DAO 只负责和 MySQL 打交道，不负责复杂业务规则。

文件：

| 文件 | 作用 |
|---|---|
| `db.go` | 初始化 MySQL |
| `redis.go` | 初始化 Redis |
| `user_dao.go` | 操作 users 表 |
| `course_dao.go` | 操作 courses 表 |
| `order_dao.go` | 操作 orders 表 |
| `video_dao.go` | 操作 course_videos 表 |

例如 `CourseDao.ListCourses` 做两件事：

1. 查总数。
2. 查分页课程列表。

```go
d.db.Model(&model.Course{}).Count(&total)
```

然后：

```go
d.db.Select(...).
    Order("created_at DESC").
    Offset(offset).
    Limit(pageSize).
    Find(&courses)
```

注意这里没有 `SELECT *`，只查需要字段，这是性能优化。

## 9. model 层是什么

`model/` 是数据库表和 Go 结构体的映射。

比如 `users` 表对应：

```text
logic-server/model/user.go
```

Go 结构体：

```go
type User struct {
    ID        uint64
    Username  string
    Password  string
    Role      int8
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

GORM 通过结构体标签知道字段类型、索引和约束。

例如：

```go
Username string `gorm:"type:varchar(64);uniqueIndex:idx_username;not null"`
```

表示：

```text
username 是 varchar(64)
不能为空
有唯一索引 idx_username
```

## 10. 数据库表是怎么设计的

建表文件：

```text
deploy/mysql/init.sql
```

项目有 4 张核心表。

### 10.1 users 用户表

存用户账号和密码。

字段：

```text
id
username
password
role
created_at
updated_at
```

关键点：

```text
username 有唯一索引
password 存 bcrypt hash
role: 0 学员，1 管理员
```

### 10.2 courses 课程表

存课程信息。

字段：

```text
id
title
description
cover_key
price
teacher_id
created_at
updated_at
```

`cover_key` 是 OSS 中封面图的 object key，不是图片本身。

### 10.3 course_videos 视频表

存课程下面的视频。

字段：

```text
id
course_id
title
video_key
duration
sort_order
created_at
```

`video_key` 是 OSS 中视频文件的 object key。

### 10.4 orders 订单表

存用户购买记录。

字段：

```text
id
user_id
course_id
price
status
created_at
```

最重要的索引：

```sql
UNIQUE INDEX idx_user_course (user_id, course_id)
```

作用：

```text
同一个用户不能重复购买同一门课
```

## 11. 用户注册是怎么实现的

文件：

```text
logic-server/service/user_service.go
```

方法：

```go
Register
```

流程：

```text
1. 判断 username/password 是否为空
2. 根据 username 查询用户是否已存在
3. 如果已存在，返回 username already exists
4. 用 bcrypt 加密密码
5. 插入 users 表
6. 返回 user_id
```

为什么要 bcrypt：

不能把用户密码明文存进数据库。bcrypt 是专门用于密码哈希的算法，比普通 MD5/SHA 更适合密码存储。

## 12. 用户登录是怎么实现的

文件：

```text
logic-server/service/user_service.go
```

方法：

```go
Login
```

流程：

```text
1. 判断 username/password 是否为空
2. 根据 username 查用户
3. 如果用户不存在，返回 invalid credentials
4. 用 bcrypt 比对密码
5. 密码正确，生成 JWT
6. 返回 token、user_id、role
```

JWT 生成在：

```text
common/jwt/jwt.go
```

Token 里放：

```text
user_id
role
exp
iat
issuer
```

## 13. 课程列表是怎么实现的

文件：

```text
logic-server/service/course_service.go
```

方法：

```go
ListCourses
```

这是项目后端性能优化的核心。

完整流程：

```text
1. 修正 page/pageSize
2. 生成 Redis cache key
3. 查 Redis
4. 命中则 proto.Unmarshal 后直接返回
5. 未命中则进入 singleflight
6. 查 MySQL
7. 生成封面签名 URL
8. proto.Marshal 后写 Redis
9. 返回结果
```

cache key：

```text
cache:courses:page:{page}:size:{pageSize}
```

pageSize 最大限制：

```text
100
```

为什么限制 pageSize：

防止用户一次请求过多数据，造成大 key、大响应和数据库压力。

## 14. 课程列表缓存怎么做的

缓存模式是 Cache-Aside。

读流程：

```text
先读 Redis
Redis 没有再查 MySQL
查到后写回 Redis
```

写流程：

```text
创建课程成功后删除课程列表缓存
```

缓存内容不是 JSON，而是 protobuf：

```go
proto.Marshal(resp)
```

读取时：

```go
proto.Unmarshal(cachedData, &resp)
```

为什么用 protobuf：

1. 体积更小。
2. 序列化更快。
3. Logic Server 本来就是 protobuf response。
4. 减少 JSON 反射开销。

缺点：

Redis 里看不懂，不如 JSON 方便调试。

## 15. singleflight 是怎么用的

项目使用：

```go
courseListGroup singleflight.Group
```

作用是防止缓存击穿。

什么是缓存击穿：

```text
一个热门课程列表 key 过期
同时来了 1000 个请求
如果没有保护，1000 个请求都会查 MySQL
```

singleflight 的效果：

```text
同一个 key 只有 1 个请求查 MySQL
其他请求等待并共享结果
```

这能保护 MySQL。

## 16. TTL 随机抖动是怎么做的

课程列表缓存过期时间：

```go
expiration := 5*time.Minute + time.Duration(rand.Intn(60))*time.Second
```

意思是：

```text
缓存 5 分钟到 5 分 59 秒
```

为什么加随机时间：

如果所有缓存都是固定 5 分钟，它们可能同时过期，大量请求一起打到数据库。这叫缓存雪崩。

加随机时间后，过期时间被打散。

## 17. 创建课程是怎么实现的

方法：

```go
CreateCourse
```

流程：

```text
1. 判断 title 是否为空
2. 判断 price 是否小于 0
3. 构造 Course model
4. 写入 courses 表
5. 删除课程列表缓存
6. 返回 course_id
```

为什么创建课程后要删缓存：

如果不删缓存，用户看到的课程列表可能还是旧的，新课程不会马上出现。

当前删除方式：

```text
删除 cache:courses: 前缀的 key
```

## 18. 获取课程详情是怎么实现的

方法：

```go
GetCourse
```

流程：

```text
1. 校验 course_id
2. 查 courses 表
3. 查 course_videos 表
4. 给封面生成签名 URL
5. 返回课程和视频列表
```

这里是两次查询：

1. 查课程。
2. 查视频。

因为只查单个课程，所以没有明显 N+1 问题。

## 19. 订单创建是怎么实现的

文件：

```text
logic-server/service/order_service.go
```

方法：

```go
CreateOrder
```

流程：

```text
1. 校验 user_id 和 course_id
2. 生成 Redis 锁 key
3. 用 UUID 作为锁 value
4. Redis SetNX 尝试加锁，TTL 10 秒
5. 如果锁已存在，返回订单处理中
6. 如果 Redis 异常，降级依赖 MySQL 唯一索引
7. 查课程是否存在
8. 查用户是否已购买
9. 创建订单
10. defer 里用 Lua 释放锁
```

锁 key：

```text
lock:order:user:{user_id}:course:{course_id}
```

锁 value：

```text
UUID
```

为什么 value 用 UUID：

防止误删别人的锁。

## 20. 为什么订单还需要 MySQL 唯一索引

Redis 锁不是 100% 可靠。

例如：

1. Redis 主从切换可能丢锁。
2. 锁可能 10 秒后过期。
3. Redis 可能不可用。
4. 代码可能绕过锁。

所以数据库必须兜底。

订单表有：

```sql
UNIQUE INDEX idx_user_course (user_id, course_id)
```

这能保证：

```text
同一个用户对同一门课最多只有一条订单
```

Redis 锁负责减少并发冲突，MySQL 唯一索引负责最终正确性。

## 21. Lua 释放锁为什么重要

释放锁脚本：

```lua
if redis.call("get", KEYS[1]) == ARGV[1] then
    return redis.call("del", KEYS[1])
else
    return 0
end
```

为什么不用先 GET 再 DEL：

因为两步不是原子的。

可能出现：

```text
A 拿到锁
A 卡住，锁过期
B 拿到新锁
A 恢复执行，直接 DEL
A 把 B 的锁删了
```

Lua 可以保证判断和删除一次性完成。

## 22. 检查是否购买是怎么实现的

方法：

```go
CheckPurchase
```

DAO 层查询：

```go
Where("user_id = ? AND course_id = ?", userID, courseID).Count(&count)
```

如果 count > 0，说明已购买。

这个逻辑用于：

1. 创建订单前判断是否重复购买。
2. 播放视频前判断是否有权限。

## 23. 视频播放 URL 是怎么实现的

文件：

```text
logic-server/service/video_service.go
```

方法：

```go
GetVideoPlayURL
```

流程：

```text
1. 校验 video_id 和 user_id
2. 查视频记录
3. 根据视频查 course_id
4. 查订单表，确认用户是否购买该课程
5. 未购买则返回 course not purchased
6. 已购买则调用 OSS 生成签名 URL
7. 返回 play_url
```

签名 URL 有效期：

```text
3600 秒
```

这样做的意义：

```text
视频文件在 OSS 私有 Bucket
用户必须通过后端校验后才能拿到临时播放地址
```

## 24. 视频分片上传是怎么实现的

视频上传有两个核心方法。

### 24.1 初始化分片上传

方法：

```go
InitMultipartUpload
```

流程：

```text
1. 校验 course_id、title、filename、part_count
2. 生成 OSS object_key
3. 调用 OSS InitiateMultipartUpload
4. 得到 upload_id
5. 为每个 part 生成预签名 PUT URL
6. 返回 upload_id、object_key、upload_urls
```

part_count 最大限制：

```text
1000
```

### 24.2 完成分片上传

方法：

```go
CompleteMultipartUpload
```

流程：

```text
1. 校验 course_id、upload_id、object_key、title、parts
2. 检查 object_key 是否属于当前课程
3. 检查每个 part_number 和 etag
4. 调用 OSS CompleteMultipartUpload
5. 写入 course_videos 表
6. 返回 video_id
```

为什么要检查 object_key 前缀：

防止用户拿别的课程的 object_key 来完成上传。

## 25. OSS 模块做了什么

文件：

```text
logic-server/oss/oss_client.go
```

它封装了阿里云 OSS SDK。

主要函数：

| 函数 | 作用 |
|---|---|
| `InitOSSClient` | 初始化 OSS client 和 bucket |
| `GenerateSignedURL` | 生成 GET 播放地址 |
| `GeneratePresignedPutURL` | 生成 PUT 上传地址 |
| `InitiateMultipartUpload` | 初始化分片上传 |
| `GeneratePresignedPartURL` | 生成某个分片的上传 URL |
| `CompleteMultipartUpload` | 合并分片 |
| `GenerateCoverKey` | 生成封面 object key |
| `GenerateVideoKey` | 生成视频 object key |

OSS 中真正保存的是文件。

MySQL 里只保存：

```text
cover_key
video_key
```

## 26. 错误处理是怎么做的

错误码定义在：

```text
common/errcode/errcode.go
```

例如：

```text
ErrInvalidParams
ErrUnauthorized
ErrForbidden
ErrUsernameExists
ErrCourseNotFound
ErrVideoNotFound
ErrNotPurchased
ErrAlreadyPurchased
ErrInternal
ErrOSS
```

Logic Server 返回错误时，不是简单返回字符串，而是：

```go
errcode.ErrInvalidParams.ToGRPCError()
```

它会把业务错误转换成 gRPC status，并通过 `errdetails.ErrorInfo` 携带：

```text
业务 code
HTTP status
message
```

Web Server 收到后再转回 HTTP JSON。

这样比字符串匹配更稳定。

## 27. Redis Adapter 做了什么

文件：

```text
logic-server/service/adapters.go
```

`RedisAdapter` 把 `go-redis` 客户端包装成项目自己的 `Cache` 接口。

实现的方法：

```go
Get
Set
SetNX
Eval
DeletePrefix
```

这样 service 层不需要知道 Redis 的具体 API。

## 28. 为什么 Redis 失败时还能运行

Redis 是缓存和辅助锁，不是权威数据。

权威数据在 MySQL。

所以 Redis 如果连接失败，项目会尽量降级：

1. 课程列表缓存失败，就直接查 MySQL。
2. 订单锁失败，就依赖 MySQL 唯一索引兜底。

这样可用性更好。

缺点是：

Redis 故障时，MySQL 压力会变大。

## 29. Logic Server 的 Dockerfile 做了什么

文件：

```text
logic-server/Dockerfile
```

也是多阶段构建。

第一阶段：

```text
golang:1.25-alpine
```

下载依赖并编译。

第二阶段：

```text
alpine:3.19
```

只放可执行文件和配置文件。

默认暴露端口：

```text
9090
```

## 30. Logic Server 怎么测试

测试文件在：

```text
logic-server/service/*_test.go
```

测试重点：

1. 用户登录注册。
2. 课程列表缓存。
3. singleflight 行为。
4. 创建课程。
5. 获取课程。
6. 创建订单。
7. 并发订单。
8. Redis 锁失败时降级。

项目使用：

```text
testify/assert
sqlmock
手写 mock repository
手写 mock cache
手写 mock storage
```

因为 service 层依赖接口，所以测试不一定要真的连 MySQL、Redis、OSS。

## 31. 新手应该怎么读 Logic Server

推荐顺序：

1. 看 `logic-server/main.go`  
   明白服务怎么启动。

2. 看 `logic-server/service/dependencies.go`  
   明白 service 依赖哪些能力。

3. 看 `logic-server/model/*.go`  
   明白数据库表对应哪些结构体。

4. 看 `deploy/mysql/init.sql`  
   明白真实建表语句。

5. 看 `logic-server/service/user_service.go`  
   明白注册登录。

6. 看 `logic-server/service/course_service.go`  
   明白课程列表、缓存、singleflight。

7. 看 `logic-server/service/order_service.go`  
   明白购买课程、Redis 锁、MySQL 唯一索引。

8. 看 `logic-server/service/video_service.go`  
   明白视频上传和播放鉴权。

9. 看 `logic-server/oss/oss_client.go`  
   明白 OSS 预签名 URL。

## 32. Logic Server 一句话总结

`logic-server` 是这个项目的业务核心。它通过 gRPC 接收 `web-server` 的请求，负责用户、课程、订单、视频等业务逻辑，并通过 MySQL 保存正式数据，通过 Redis 提升性能和控制并发，通过 OSS 存储和保护视频文件。

