# SUSE OAA Backend

SUSE OAA 后端服务，基于 **Go + Gin + GORM + MySQL + Redis**。

## 当前进度

项目目前已完成以下核心模块：

- 认证登录：注册、登录、刷新 Token、登出
- 账号安全：修改密码、发送验证码、验证码重置密码
- 用户模块：个人信息、用户列表、修改用户名、批量修改部门/职位
- 基础数据：部门列表、职位列表、启动初始化基础角色/部门
- 公告模块：创建、更新、推送、删除、查看当前公告/历史公告/列表

## 接口一览

### Auth
- `POST /api/v2/auth/register` 注册
- `POST /api/v2/auth/login` 登录
- `POST /api/v2/auth/refresh` 刷新 Token
- `POST /api/v2/auth/logout` 登出
- `POST /api/v2/auth/send` 发送验证码

### Password
- `POST /api/v2/password/update` 修改密码
- `POST /api/v2/password/reset` 验证码重置密码

### User
- `GET /api/v2/user/me` 当前用户信息
- `GET /api/v2/user/list` 用户列表
- `POST /api/v2/user/me/update` 修改用户名
- `POST /api/v2/user/batch` 批量修改用户部门和职位

### Department
- `GET /api/v2/department/list` 部门列表
- `POST /api/v2/department/create` 新建部门
- `POST /api/v2/department/update` 更新部门
### Role
- `GET /api/v2/role/list` 职位列表
- `POST /api/v2/role/create` 新建职位
- `POST /api/v2/role/update` 更新职位

### Announcement
- `POST /api/v2/announcement/create` 创建公告
- `POST /api/v2/announcement/update` 更新公告
- `POST /api/v2/announcement/push` 推送公告
- `GET /api/v2/announcement/active` 当前生效公告
- `GET /api/v2/announcement/history` 历史公告
- `GET /api/v2/announcement/list` 按权限获取公告列表
- `POST /api/v2/announcement/delete` 删除公告

## 权限与数据设计

### 基础角色
启动时自动初始化：

- 开发者
- 会长
- 副会长
- 部长
- 副部长
- 干事
- 会员

权限通过 `level` 比较，不依赖固定数据库 ID。

### 基础部门
启动时自动初始化：

- 算法竞赛部
- 组织宣传部
- 秘书处
- 理事会
- 项目部
- 开放原子开源协会

### 批量修改规则
`/api/v2/user/batch` 的规则是：

- 同一个 `user_id` 只处理一次
- 非法部门 / 职位会回填到失败项
- 当前操作者职位达到副会长及以上时，可跨部门处理
- 当前操作者职位低于副会长时，只能在本部门内修改更低职位

## 实现说明

### Refresh Token
- 登录后生成 Refresh Token
- Refresh Token 同时写入 Redis 和数据库
- Redis key 与 `user_id + device` 绑定

### 验证码
- 验证码存储在 Redis
- 支持过期时间
- 支持发送冷却时间
- 重置密码成功后验证码会立即失效

### 密码
- 密码使用 `bcrypt` 加密
- 验证码重置密码后默认重置为 `123456`

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
README.md
```

## 运行方式

### 1. 准备依赖
- MySQL
- Redis

### 2. 修改配置
配置文件：`/Users/starry/Documents/projects/suse_oaa_backend/configs/config.yaml`

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

- 当前项目以协会内部自用场景为主
- 接口权限与业务规则已在代码中实现
- 如需进一步接口细节，可直接查看 `internal/router`、`internal/handler`、`internal/service`
