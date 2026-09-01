# SUSE OAA Backend

SUSE OAA 后端服务，基于 **Go + Gin + GORM + MySQL + Redis**，面向协会账号、组织架构、公告以及招新 / 换届业务。


## 当前进度

项目目前已完成以下核心模块：

- **认证与账号安全**：注册、登录、刷新 Token、登出、修改密码、发送邮箱验证码、验证码重置密码
- **用户模块**：当前用户信息、用户列表、修改用户名、批量修改部门 / 职位
- **基础数据**：部门列表 / 新建 / 更新，职位列表 / 新建 / 更新
- **公告模块**：创建、更新、推送、删除、查看当前公告、历史公告和按权限获取公告列表
- **招新 / 换届模块**：周期管理、申请表提交 / 更新 / 查询、志愿下拉数据、申请列表查询、面试官管理

## 接口一览

所有接口前缀为 `/v2`。除注册、登录、刷新 Token、发送验证码、验证码重置密码外，其余接口需要在请求头携带：

```http
Authorization: Bearer <token>
```

### Auth（认证）

- `POST /v2/auth/register` 注册（公开）
- `POST /v2/auth/login` 登录（公开）
- `POST /v2/auth/refresh` 刷新 Token（公开）
- `POST /v2/auth/logout` 登出
- `POST /v2/auth/send` 发送邮箱验证码（公开）

### Password（密码）

- `POST /v2/auth/password/update` 修改密码
- `POST /v2/auth/password/reset` 验证码重置密码（公开）

### User（用户）

- `GET /v2/user/me` 当前用户信息
- `GET /v2/user/list` 用户列表（分页和筛选）
- `POST /v2/user/me/update` 修改用户名
- `POST /v2/user/batch` 批量修改用户部门和职位

### Department（部门）

- `GET /v2/department/list` 部门列表
- `POST /v2/department/create` 新建部门
- `POST /v2/department/update` 更新部门

### Role（职位）

- `GET /v2/role/list` 职位列表
- `POST /v2/role/create` 新建职位
- `POST /v2/role/update` 更新职位

### Announcement（公告）

- `POST /v2/announcement/create` 创建公告
- `POST /v2/announcement/update` 更新公告
- `POST /v2/announcement/push` 推送公告
- `GET /v2/announcement/active` 当前生效公告
- `GET /v2/announcement/history` 历史公告
- `GET /v2/announcement/list` 按权限获取公告列表
- `POST /v2/announcement/delete` 删除公告

### Term（招新 / 换届周期）

- `POST /v2/term/create` 创建周期
- `POST /v2/term/update` 更新周期
- `GET /v2/term/list` 周期列表

### Application（申请表）

- `POST /v2/application/create` 创建申请
- `POST /v2/application/update` 更新当前用户最新申请
- `GET /v2/application/me` 获取当前用户申请
- `GET /v2/application/department` 按职位获取可选部门
- `GET /v2/application/role` 按部门获取可选职位
- `GET /v2/application/list` 查询周期申请列表

### Interviewer（面试官）

- `POST /v2/interviewer/create` 添加面试官
- `POST /v2/interviewer/update` 更新面试官
- `GET /v2/interviewer/list` 面试官列表

## 统一响应格式

成功响应：

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

失败响应：

```json
{
  "code": 400,
  "message": "具体错误信息",
  "data": null
}
```

`code` 与 HTTP 状态码保持一致，常见状态码为 `400`（参数或业务错误）、`401`（未登录 / Token 无效）和 `500`（服务端错误）。

## 权限与数据设计

### 基础角色

启动时自动初始化以下角色，权限通过 `level` 比较，不依赖固定数据库 ID：

| 角色 | level | 类型 |
|---|---:|---|
| 开发者 | 100 | 协会 |
| 会长 | 90 | 协会 |
| 副会长 | 80 | 协会 |
| 部长 | 60 | 部门 |
| 副部长 | 50 | 部门 |
| 干事 | 20 | 部门 |
| 会员 | 10 | 协会 |

### 基础部门

启动时自动初始化：

- 算法竞赛部
- 组织宣传部
- 秘书处
- 理事会
- 项目部
- 开放原子开源协会

### 批量修改规则

`/v2/user/batch` 的规则是：

- 同一个 `user_id` 只处理第一次出现的记录。
- 非法部门、职位或部门 / 职位类型不匹配时，会回填到失败项。
- 当前操作者职位达到副会长及以上时，可跨部门处理。
- 当前操作者职位低于副会长时，只能在本部门内修改为更低职位。
- 成功响应中的 `data` 是未成功修改项数组；完全成功时通常为空数组。

### 公告可见性

- 职位等级 `< 30`：只能看到已发布公告。
- 职位等级 `30 - 79`：可以看到本部门公告及所有已发布公告。
- 职位等级 `>= 80`：可以看到全部公告，包括草稿。
- 同一部门同时只有一条当前生效公告；推送新公告会使原公告变为历史公告。

## Token 与验证码

### Refresh Token

- 登录后生成 `refreshToken`，同时与 `user_id + device` 绑定保存。
- 刷新 Token 时请求字段为 `refresh_token`、`user_id`、`device`。
- 登录响应字段名为 `refreshToken`（驼峰），这是当前实现的字段差异。
- 当前 JWT 生成逻辑使用配置 `jwt.expire_hour` 的数值作为分钟数；默认值 `20` 实际约为 20 分钟。

### 验证码

- 验证码存储在 Redis，支持过期和发送冷却。
- `auth/send` 和 `password/reset` 的 `type` 必须保持一致。
- 重置密码成功后验证码立即失效。
- 当前验证码重置密码的默认密码为 `123456`，登录后应立即修改密码。

## 技术栈

- Go
- Gin
- GORM
- MySQL
- Redis
- JWT
- Gomail
- bcrypt

## 项目结构

```text
cmd/
  main.go
configs/
  config.yaml
internal/
  config/
  database/
  handler/
  middleware/
  model/
  repository/
  request/
  router/
  service/
pkg/
  response/
  utils/
docs/
  frontend-api.md
README.md
```

## 运行方式

### 1. 准备依赖

- MySQL
- Redis

### 2. 修改配置

配置文件：`configs/config.yaml`

按本地环境修改：

- `mysql`
- `redis`
- `jwt`
- `email`
- `server`

### 3. 启动项目

```bash
go run ./cmd/main.go
```

## 备注

- 当前项目以协会内部自用场景为主。
- 接口权限、业务规则和时间窗口均已在代码中实现。
