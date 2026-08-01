# SUSE OAA Backend

SUSE OAA 的后端服务，当前基于 **Go + Gin + GORM + MySQL + Redis** 实现。

> 本 README 只记录**目前已经完成**的内容，未完成或后续规划暂不展开。

## 1. 当前已完成功能

### 1.1 认证与登录
- 用户注册
- 用户登录
- JWT 鉴权
- Refresh Token 刷新登录态
- 登出

### 1.2 账号安全
- 修改密码
- 发送验证码邮件
- 基于验证码重置密码
- Redis 冷却时间控制

### 1.3 用户模块
- 获取当前登录用户信息
- 分页获取用户列表
- 修改当前用户用户名
- 批量修改用户部门和职位
  - 已支持权限判断
  - 已支持重复用户去重
  - 已支持非法部门 / 非法职位的失败结果回填
  - 返回结果中会区分成功修改与失败项信息

### 1.4 基础数据模块
- 获取全部部门列表
- 获取全部职位列表
- 启动时自动初始化基础职位与基础部门

---

## 2. 当前接口列表

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
- `GET /api/v2/user/me` 获取当前用户信息
- `GET /api/v2/user/list` 获取用户列表
- `POST /api/v2/user/me/update` 修改当前用户信息
- `POST /api/v2/user/batch` 批量修改用户部门和职位

### Department
- `GET /api/v2/department/list` 获取全部部门

### Role
- `GET /api/v2/role/list` 获取全部职位

更详细的接口说明可查看：[`/Users/starry/Documents/projects/suse_oaa_backend/api文档.md`](/Users/starry/Documents/projects/suse_oaa_backend/api文档.md)

---

## 3. 技术栈

- Go 1.26
- Gin
- GORM
- MySQL
- Redis
- JWT
- Gomail
- bcrypt

---

## 4. 项目结构

```text
cmd/
  main.go                  程序入口
configs/
  config.yaml              项目配置
internal/
  config/                  配置读取
  database/                MySQL / Redis 初始化
  handler/                 HTTP 接口层
  middleware/              JWT 中间件
  model/                   数据模型
  repository/              数据访问层
  request/                 请求结构体
  router/                  路由注册
  service/                 业务逻辑层
pkg/
  response/                统一响应结构
  utils/                   JWT / UUID 等工具
api文档.md                  接口文档
README.md                  项目说明
```

---

## 5. 当前数据与权限设计

### 5.1 基础角色
项目启动时会自动初始化以下角色：
- 开发者
- 会长
- 副会长
- 部长
- 副部长
- 干事
- 会员

角色通过 `level` 做权限比较，而不是依赖固定数据库 ID。

### 5.2 基础部门
项目启动时会自动初始化以下部门：
- 算法竞赛部
- 组织宣传部
- 秘书处
- 理事会
- 项目部

### 5.3 批量修改用户权限规则
`/api/v2/user/batch` 当前已实现的核心逻辑：
- 当前操作者的 `department_id` 和 `role_id` 来自 JWT Token
- 若当前操作者职位 **高于或等于“副会长”分界线**，可跨部门修改，但目标职位仍需低于副会长分界线
- 若当前操作者职位 **低于副会长**，则必须：
  - 自己有合法部门
  - 目标部门与自己部门一致
  - 自己职位高于目标职位
- 同一个 `user_id` 在一次批量请求中只处理一次
- 非法 `department_id` / `role_id` 会进入失败项返回，不会直接中断整个批量流程

---

## 6. 运行方式

### 6.1 准备依赖
需要先准备以下服务：
- MySQL
- Redis

### 6.2 配置文件
当前配置文件位置：
- `/Users/starry/Documents/projects/suse_oaa_backend/configs/config.yaml`

建议根据本地环境修改以下配置：
- `mysql`
- `redis`
- `jwt`
- `email`
- `server`

> 建议不要直接使用仓库中现有敏感配置，启动前请替换为自己的本地 / 测试环境配置。

### 6.3 启动项目
在项目根目录执行：

```bash
go run ./cmd/main.go
```

默认监听端口由配置文件中的 `server.port` 决定。

---

## 7. 响应格式

项目当前统一响应格式如下：

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

失败时同样返回：

```json
{
  "code": 400,
  "message": "错误信息",
  "data": null
}
```

---

## 8. 当前实现说明

### Refresh Token
- 登录后会生成 Refresh Token
- Refresh Token 同时存储在 Redis 与数据库表中
- Redis 中的 key 与 `user_id + device` 绑定

### 验证码
- 验证码存储在 Redis 中
- 支持过期时间控制
- 支持发送冷却时间控制

### 密码
- 密码使用 `bcrypt` 加密存储
- 当前验证码重置密码逻辑会将密码重置为固定值：`123456`

---

## 9. 当前状态

目前该项目已经完成：
- 基础用户体系
- 登录鉴权
- Refresh Token
- 邮箱验证码
- 用户列表与个人信息接口
- 部门 / 职位基础接口
- 批量修改用户部门与职位

后续若有新增功能，可继续在 README 与 `api文档.md` 中同步补充。
