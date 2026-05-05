# Web Server 详解

这份文档只讲 `web-server`。目标是让没有后端基础的人也能明白：它是什么、启动后做了什么、每个目录负责什么、一次 HTTP 请求是怎么被处理的。

## 1. Web Server 是什么

`web-server` 是这个项目的 HTTP API 网关。

你可以把它理解成“前台接待员”：

```text
浏览器 / 前端
   |
   v
web-server
   |
   v
logic-server
```

前端不会直接访问 `logic-server`，而是访问 `web-server`。

`web-server` 主要做这些事：

1. 接收前端 HTTP 请求。
2. 判断请求路径应该交给哪个 handler。
3. 校验 JWT，确认用户是否登录。
4. 判断用户是否是管理员。
5. 把 HTTP 请求转换成 gRPC 请求。
6. 调用 `logic-server`。
7. 把 `logic-server` 返回的结果包装成统一 JSON 返回给前端。
8. 记录访问日志。
9. 处理 panic，防止服务崩溃。
10. 提供 Swagger API 文档。
11. 提供 `/health` 健康检查。
12. 对部分接口做限流。

一句话总结：

> `web-server` 不直接处理核心业务和数据库，它主要负责 HTTP 层、鉴权层、路由层和转发层。

## 2. Web Server 用了哪些技术

`web-server` 主要使用这些技术：

| 技术 | 作用 |
|---|---|
| Go | 编写后端服务 |
| Gin | 提供 HTTP API |
| gRPC Client | 调用 logic-server |
| JWT | 登录认证 |
| Zap | 结构化日志 |
| lumberjack | 日志轮转 |
| Viper | 配置读取 |
| Swagger / swaggo | API 文档 |
| x/time/rate | 令牌桶限流 |
| Docker | 容器化运行 |

为什么用 Gin：

Gin 是 Go 里很常见的 Web 框架。它能方便地写路由、中间件、参数解析和 JSON 响应。相比直接用 Go 原生 `net/http`，Gin 更省代码，也更适合项目开发。

为什么 Web Server 只做 HTTP：

因为浏览器更适合访问 HTTP/JSON 接口，而不是直接访问 gRPC。所以项目选择：

```text
对前端：HTTP + JSON
后端内部：gRPC + protobuf
```

## 3. web-server 目录结构

`web-server` 目录大概是：

```text
web-server/
├── main.go
├── config.yaml
├── Dockerfile
├── go.mod
├── router/
│   └── router.go
├── middleware/
│   ├── auth.go
│   ├── cors.go
│   ├── logger.go
│   └── ratelimit.go
├── handler/
│   ├── auth_handler.go
│   ├── course_handler.go
│   ├── order_handler.go
│   └── video_handler.go
├── grpc_client/
│   └── client.go
├── docs/
│   ├── docs.go
│   ├── swagger.json
│   └── swagger.yaml
└── mocks/
```

每个目录的作用：

| 目录/文件 | 作用 |
|---|---|
| `main.go` | Web Server 启动入口 |
| `router/` | 注册所有 HTTP 路由 |
| `middleware/` | 中间件，比如鉴权、日志、限流 |
| `handler/` | 具体处理 HTTP 请求 |
| `grpc_client/` | 初始化 gRPC 客户端 |
| `docs/` | Swagger 文档 |
| `mocks/` | 测试用 mock |
| `Dockerfile` | 构建 Web Server 镜像 |

## 4. Web Server 启动过程

入口文件是：

```text
web-server/main.go
```

启动过程可以分成 6 步。

### 4.1 读取配置

代码：

```go
config.InitConfig("config.yaml")
cfg := config.GetConfig()
```

它会读取 `config.yaml`，也支持环境变量覆盖。例如：

```text
HTTP_PORT
LOGIC_SERVER_ADDR
JWT_SECRET
```

这样本地开发可以用配置文件，Docker 部署可以用环境变量。

### 4.2 初始化日志

代码：

```go
logger.InitLogger("dev", "logs/web-server.log")
defer logger.Log.Sync()
```

项目使用 Zap 记录结构化日志，并写入：

```text
web-server/logs/web-server.log
```

### 4.3 初始化 gRPC Client

代码：

```go
grpc_client.InitGRPCClient()
defer grpc_client.Close()
```

这一步会连接 `logic-server`，然后创建 4 个客户端：

```go
UserClient
CourseClient
VideoClient
OrderClient
```

这些 client 用来调用 logic-server 的业务接口。

### 4.4 注册路由

代码：

```go
r := router.SetupRouter()
```

所有 HTTP 路由都在：

```text
web-server/router/router.go
```

### 4.5 启动 HTTP 服务

代码：

```go
srv := &http.Server{
    Addr:    ":" + cfg.Server.HttpPort,
    Handler: r,
}
```

默认端口是：

```text
8080
```

也就是访问：

```text
http://localhost:8080
```

### 4.6 优雅停机

代码会监听：

```go
syscall.SIGINT
syscall.SIGTERM
```

当你按 `Ctrl+C` 或容器停止时，它会给服务 5 秒时间处理完已有请求，再关闭。

这叫“优雅停机”。

## 5. 路由是怎么注册的

路由文件：

```text
web-server/router/router.go
```

核心代码逻辑：

```go
r := gin.New()
r.Use(middleware.GinLogger(), middleware.GinRecovery(true))
r.GET("/health", ...)
r.Use(middleware.CORSMiddleware())
r.GET("/swagger/*any", ...)
```

这里做了几件事：

1. 创建 Gin 引擎。
2. 加日志中间件。
3. 加 panic 恢复中间件。
4. 加健康检查接口。
5. 加 CORS 跨域中间件。
6. 加 Swagger 文档路由。

业务路由统一放在：

```text
/api/v1
```

## 6. Web Server 提供哪些接口

### 6.1 公开接口

这些接口不用登录：

```text
POST /api/v1/auth/register
POST /api/v1/auth/login
GET  /api/v1/courses
GET  /api/v1/courses/:course_id
```

含义：

| 接口 | 作用 |
|---|---|
| 注册 | 新用户注册 |
| 登录 | 用户登录并返回 JWT |
| 课程列表 | 游客也可以看课程 |
| 课程详情 | 游客也可以看课程详情 |

### 6.2 需要登录的接口

这些接口需要 JWT：

```text
POST /api/v1/orders
GET  /api/v1/player/:video_id
```

含义：

| 接口 | 作用 |
|---|---|
| 创建订单 | 用户购买课程 |
| 获取播放地址 | 已购买用户获取视频播放 URL |

### 6.3 需要管理员权限的接口

这些接口需要登录，并且 `role == 1`：

```text
POST /api/v1/courses
GET  /api/v1/courses/cover/upload_url
POST /api/v1/courses/:course_id/videos/init
POST /api/v1/courses/:course_id/videos/complete
```

含义：

| 接口 | 作用 |
|---|---|
| 创建课程 | 管理员发布课程 |
| 获取封面上传 URL | 给封面图生成 OSS 上传链接 |
| 初始化视频上传 | 开始 OSS 分片上传 |
| 完成视频上传 | 合并 OSS 分片并写入视频记录 |

## 7. Handler 是什么

`handler` 是“HTTP 请求处理函数”。

例如前端请求：

```text
GET /api/v1/courses
```

路由会找到：

```go
courseHandler.ListCourses
```

这个函数就在：

```text
web-server/handler/course_handler.go
```

handler 主要做 4 件事：

1. 从 HTTP 请求里取参数。
2. 设置超时时间。
3. 调用 gRPC client 请求 logic-server。
4. 把结果返回给前端。

## 8. 登录注册是怎么实现的

相关文件：

```text
web-server/handler/auth_handler.go
```

### 8.1 注册

前端请求：

```text
POST /api/v1/auth/register
```

请求体类似：

```json
{
  "username": "tom",
  "password": "123456"
}
```

Web Server 处理流程：

```text
AuthHandler.Register
  -> ShouldBindJSON 解析 JSON
  -> context.WithTimeout 设置 5 秒超时
  -> 调用 UserClient.Register
  -> logic-server 真正注册用户
  -> 返回 user_id
```

Web Server 不直接写数据库，它只是转发到 logic-server。

### 8.2 登录

前端请求：

```text
POST /api/v1/auth/login
```

请求体：

```json
{
  "username": "admin",
  "password": "admin123"
}
```

Web Server 处理流程：

```text
AuthHandler.Login
  -> 解析 JSON
  -> 调用 UserClient.Login
  -> logic-server 校验密码并生成 JWT
  -> 返回 token、user_id、role
```

## 9. JWT 鉴权是怎么实现的

相关文件：

```text
web-server/middleware/auth.go
common/jwt/jwt.go
```

前端登录成功后，会拿到一个 token。

之后请求需要登录的接口时，要带上：

```text
Authorization: Bearer <token>
```

`AuthMiddleware` 会做这些事：

1. 读取 `Authorization` 请求头。
2. 判断是否是 `Bearer token` 格式。
3. 调用 `jwt.ParseToken` 解析 token。
4. 解析成功后，把 `user_id` 和 `role` 放进 Gin context。

代码逻辑：

```go
c.Set("user_id", claims.UserID)
c.Set("role", claims.Role)
```

后面的 handler 就可以从 context 里取当前用户。

例如创建订单时：

```go
userID, _ := c.Get("user_id")
```

## 10. 管理员权限是怎么判断的

还是在：

```text
web-server/middleware/auth.go
```

管理员中间件：

```go
func AdminMiddleware() gin.HandlerFunc
```

逻辑很简单：

```go
role, exists := c.Get("role")
if !exists || role.(int32) != 1 {
    response.Error(c, errcode.ErrForbidden)
    c.Abort()
    return
}
```

也就是说：

```text
role == 1 表示管理员
```

当前角色体系比较简单，只有学生和管理员。如果以后有讲师、运营，需要升级成 RBAC 权限模型。

## 11. 课程列表请求是怎么走的

这是最适合新手理解项目的一条链路。

前端请求：

```text
GET /api/v1/courses?page=1&page_size=10
```

完整流程：

```text
1. 请求进入 web-server
2. router.go 匹配到 GET /courses
3. 调用 CourseHandler.ListCourses
4. 从 query 参数取 page 和 page_size
5. 创建 5 秒超时 context
6. 调用 h.courseClient.ListCourses
7. gRPC 请求发送到 logic-server
8. logic-server 返回课程列表
9. web-server 用 response.Success 包装 JSON
10. 返回给前端
```

Web Server 代码重点：

```go
page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
```

然后调用 gRPC：

```go
resp, err := h.courseClient.ListCourses(ctx, &pb.ListCoursesRequest{
    Page:     int32(page),
    PageSize: int32(pageSize),
})
```

注意：真正查 Redis 和 MySQL 的不是 Web Server，而是 Logic Server。

## 12. 创建课程是怎么走的

创建课程接口：

```text
POST /api/v1/courses
```

需要管理员权限。

Web Server 主要做：

1. 校验登录。
2. 校验管理员权限。
3. 解析课程参数。
4. 如果有封面文件，则先拿 OSS 上传 URL 并上传封面。
5. 调用 gRPC `CreateCourse`。

课程请求支持 JSON，也支持 multipart form。

JSON 示例：

```json
{
  "title": "Go 微服务课程",
  "description": "课程介绍",
  "price": 99.9,
  "cover_key": "covers/xxx.jpg"
}
```

Web Server 会把当前管理员的 user_id 作为 teacher_id：

```go
teacherID, _ := c.Get("user_id")
```

然后调用：

```go
h.courseClient.CreateCourse(...)
```

## 13. 购买课程是怎么走的

接口：

```text
POST /api/v1/orders
```

需要登录。

请求体：

```json
{
  "course_id": 1001
}
```

Web Server 处理流程：

```text
OrderHandler.CreateOrder
  -> 解析 course_id
  -> 从 JWT context 取 user_id
  -> 调用 OrderClient.CreateOrder
  -> logic-server 处理购买逻辑
  -> 返回 order_id
```

Web Server 自己不判断是否重复购买，也不查课程价格。这些都在 Logic Server 里做。

## 14. 获取播放地址是怎么走的

接口：

```text
GET /api/v1/player/:video_id
```

需要登录。

流程：

```text
VideoHandler.GetPlayURL
  -> 从 path 取 video_id
  -> 从 JWT context 取 user_id
  -> 调用 VideoClient.GetVideoPlayURL
  -> logic-server 检查用户是否购买课程
  -> 已购买则生成 OSS 签名播放 URL
  -> 返回 play_url
```

这个设计可以保护视频：

```text
用户不能直接访问 OSS 原始视频地址
必须先经过后端鉴权
```

## 15. 视频上传接口是怎么走的

视频上传分两步。

### 15.1 初始化分片上传

接口：

```text
POST /api/v1/courses/:course_id/videos/init
```

请求：

```json
{
  "title": "第一节",
  "filename": "lesson1.mp4",
  "part_count": 3
}
```

Web Server 调用：

```go
videoClient.InitMultipartUpload
```

Logic Server 返回：

```json
{
  "upload_id": "...",
  "object_key": "...",
  "upload_urls": ["...", "...", "..."]
}
```

前端拿到这些 URL 后，直接把每个分片上传到 OSS。

### 15.2 完成分片上传

接口：

```text
POST /api/v1/courses/:course_id/videos/complete
```

请求：

```json
{
  "upload_id": "...",
  "object_key": "...",
  "title": "第一节",
  "parts": [
    {
      "part_number": 1,
      "etag": "..."
    }
  ]
}
```

Web Server 调用：

```go
videoClient.CompleteMultipartUpload
```

Logic Server 会让 OSS 合并分片，并保存视频记录。

## 16. Web Server 如何调用 Logic Server

相关文件：

```text
web-server/grpc_client/client.go
```

初始化时创建一个连接：

```go
conn, err = grpc.NewClient(addr,
    grpc.WithTransportCredentials(insecure.NewCredentials()),
)
```

然后创建多个业务 client：

```go
UserClient = pb.NewUserServiceClient(conn)
CourseClient = pb.NewCourseServiceClient(conn)
VideoClient = pb.NewVideoServiceClient(conn)
OrderClient = pb.NewOrderServiceClient(conn)
```

为什么一个连接可以给多个 client 用：

gRPC 的 `ClientConn` 是并发安全的，底层是 HTTP/2，可以多路复用。多个请求可以共享同一个连接。

当前不足：

生产环境不应该使用 insecure 明文连接，应该使用 TLS 或 mTLS。

## 17. 错误是怎么返回给前端的

Web Server 统一用：

```text
common/response
common/errcode
```

Logic Server 返回的是 gRPC error。

Web Server 会调用：

```go
errcode.FromGRPCError(err)
```

把 gRPC error 转成业务错误码。

最后返回统一 JSON：

```json
{
  "code": 10002,
  "msg": "unauthorized",
  "data": null
}
```

成功响应类似：

```json
{
  "code": 0,
  "msg": "success",
  "data": {}
}
```

## 18. 日志是怎么做的

相关文件：

```text
web-server/middleware/logger.go
common/logger/logger.go
```

每个 HTTP 请求都会记录：

```text
status
method
path
query
ip
user-agent
errors
latency
```

如果发生 panic，`GinRecovery` 会捕获错误，记录堆栈，并返回 500，避免服务直接崩溃。

日志写到：

```text
web-server/logs/web-server.log
```

并且有日志轮转。

## 19. 限流是怎么做的

相关文件：

```text
web-server/middleware/ratelimit.go
```

使用的是令牌桶算法。

简单理解：

```text
桶里有令牌，请求来了拿一个令牌
拿到令牌就放行
拿不到就返回 429
```

当前是单机限流，也就是只对当前 Web Server 实例有效。如果以后 Web Server 多实例，应该改成 Redis 分布式限流。

## 20. Swagger 是做什么的

Swagger 是 API 文档。

访问：

```text
http://localhost:8080/swagger/index.html
```

可以看到接口说明、请求参数、响应结构。

相关文件：

```text
web-server/docs/swagger.yaml
web-server/docs/swagger.json
```

## 21. Web Server 的 Dockerfile 做了什么

文件：

```text
web-server/Dockerfile
```

它是多阶段构建：

第一阶段：

```text
golang:1.25-alpine
```

用于下载依赖、编译 Go 二进制。

第二阶段：

```text
alpine:3.19
```

只放编译好的 `web-server` 可执行文件和配置文件。

这样最终镜像更小。

## 22. 新手应该怎么读 Web Server

推荐顺序：

1. 看 `web-server/main.go`  
   明白服务怎么启动。

2. 看 `web-server/router/router.go`  
   明白有哪些接口。

3. 看 `web-server/middleware/auth.go`  
   明白登录态怎么判断。

4. 看 `web-server/handler/auth_handler.go`  
   明白登录注册怎么调用 logic-server。

5. 看 `web-server/handler/course_handler.go`  
   明白课程列表和创建课程。

6. 看 `web-server/handler/order_handler.go`  
   明白购买课程。

7. 看 `web-server/handler/video_handler.go`  
   明白视频播放和上传。

## 23. Web Server 一句话总结

`web-server` 是这个项目的 HTTP 入口层。它不直接操作数据库，也不直接处理复杂业务，而是负责接收前端请求、做鉴权和参数处理，然后通过 gRPC 把请求转发给 `logic-server`，最后把结果以统一 JSON 格式返回给前端。

