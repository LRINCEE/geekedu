# GeekEdu JMeter 压测方案

## 核心接口

| 场景 | 方法 | URL | Header | 请求体 | 需要 Token |
|---|---|---|---|---|---|
| 课程列表 | `GET` | `http://localhost:8080/api/v1/courses?page=${pageNum}&page_size=10` | 无 | 无 | 否 |
| 课程详情 | `GET` | `http://localhost:8080/api/v1/courses/${courseId}` | 无 | 无 | 否 |
| 管理员登录 | `POST` | `http://localhost:8080/api/v1/auth/login` | `Content-Type: application/json` | `login_admin.json` | 否 |
| 学员注册 | `POST` | `http://localhost:8080/api/v1/auth/register` | `Content-Type: application/json` | `register_student_template.json` | 否 |
| 学员登录 | `POST` | `http://localhost:8080/api/v1/auth/login` | `Content-Type: application/json` | `login_student_template.json` | 否 |
| 创建订单 | `POST` | `http://localhost:8080/api/v1/orders` | `Content-Type: application/json` + `Authorization: Bearer <student_token>` | `create_order_template.json` | 是 |
| 播放鉴权 | `GET` | `http://localhost:8080/api/v1/player/${videoId}` | `Authorization: Bearer <student_token>` | 无 | 是 |

## 默认测试数据假设

- `courseId=16954`
- `videoId=1`
- 默认管理员账号：`admin / admin123`

这两个 ID 可以通过 JMeter 属性覆盖：

```powershell
jmeter -n -t tests/jmeter/geekedu.jmx -JcourseId=16954 -JvideoId=1 -l tests/jmeter/result.jtl -e -o tests/jmeter/report
```

## 线程组设计

### 1. Public - List Courses
- 目标：压公开热点读接口
- 默认：`100` 线程，`10s` ramp-up，`60s` 持续时间
- 标签：`Public - List Courses`

### 2. Public - Get Course Detail
- 目标：压单课程详情接口
- 默认：`80` 线程，`10s` ramp-up，`60s` 持续时间
- 标签：`Public - Get Course Detail`

### 3. Auth - Login Admin
- 目标：压登录链路本身
- 默认：`50` 线程，`10s` ramp-up，`60s` 持续时间
- 标签：`Auth - Login Admin`

### 4. Order Flow
- 目标：压“注册 -> 登录 -> 下单”完整下单链路
- 默认：`50` 线程，`10s` ramp-up，`1` 次循环
- 说明：每个线程会生成唯一用户名，避免重复注册/重复购买污染结果
- 关键观察标签：`Order Flow - Create Order`

### 5. Player Flow
- 目标：压“注册 -> 登录 -> 下单 -> 获取播放地址”完整播放授权链路
- 默认：`30` 线程，`10s` ramp-up，`1` 次循环
- 关键观察标签：`Player Flow - Get Play URL`

## 运行命令

```powershell
jmeter -n -t tests/jmeter/geekedu.jmx -l tests/jmeter/result.jtl -e -o tests/jmeter/report
```

## 报告重点

看 `tests/jmeter/report/index.html` 或 `tests/jmeter/result.jtl` 时，重点关注：

- `Public - List Courses`
- `Public - Get Course Detail`
- `Auth - Login Admin`
- `Order Flow - Create Order`
- `Player Flow - Get Play URL`

重点指标：

- Throughput
- Average
- 90th pct
- 95th pct
- 99th pct
- Error %

## 前置条件

压测前必须保证：

1. `web-server`、`logic-server`、`mysql`、`redis` 已启动
2. `courseId` 和 `videoId` 在当前数据库中有效，并且 `videoId` 属于 `courseId`
3. 管理员默认账号仍可用

如果数据库被重置且 `16954 / 1` 不再存在，需要用当前环境中的实际课程/视频 ID 覆盖：

```powershell
jmeter -n -t tests/jmeter/geekedu.jmx -JcourseId=<实际课程ID> -JvideoId=<实际视频ID> -l tests/jmeter/result.jtl -e -o tests/jmeter/report
```
