# SUSE OAA Backend

SUSE OAA 后端服务，基于 **Go + Gin + GORM + MySQL + Redis**，面向协会账号、组织架构、公告以及招新 / 换届业务。

## 当前进度

项目目前已完成以下核心模块：

- **认证与账号安全**：注册、登录、刷新 Token、登出、修改密码、发送邮箱验证码、验证码重置密码
- **用户模块**：当前用户信息、用户列表、更新用户资料、删除用户、批量修改部门 / 职位
- **基础数据**：部门列表 / 新建 / 更新，职位列表 / 新建 / 更新
- **公告模块**：创建、更新、推送、删除、按权限获取公告列表
- **招新 / 换届模块**：周期创建 / 更新 / 删除 / 查询、申请表提交 / 更新 / 删除 / 查询、志愿下拉数据、面试官创建 / 更新 / 删除 / 查询、面试结果创建 / 更新 / 决策枚举查询
- **文件模块**：图片上传到对象存储并返回 `uri` 与临时访问链接

## 接口一览

所有接口前缀为 `/v2`。除注册、登录、刷新 Token、发送验证码、验证码重置密码外，其余接口需要在请求头携带：

```http
Authorization: Bearer <token>
```

### Auth（认证）

| 方法 | 路径 | 说明 | 参数 |
|---|---|---|---|
| POST | `/v2/auth/register` | 注册（公开） | JSON：`student_id`、`username`、`name`、`email`、`password` |
| POST | `/v2/auth/login` | 登录（公开） | JSON：`account`、`password`、`device` |
| POST | `/v2/auth/refresh` | 刷新 Token（公开） | JSON：`refresh_token`、`user_id`、`device` |
| POST | `/v2/auth/logout` | 登出 | JSON：`device` |
| POST | `/v2/auth/send` | 发送邮箱验证码（公开） | JSON：`account`、`scene` |

### Password（密码）

| 方法 | 路径 | 说明 | 参数 |
|---|---|---|---|
| POST | `/v2/auth/password/update` | 修改密码 | JSON：`old_password`、`new_password` |
| POST | `/v2/auth/password/reset` | 验证码重置密码（公开） | JSON：`account`、`code`、`scene` |

### User（用户）

| 方法 | 路径 | 说明 | 参数 |
|---|---|---|---|
| GET | `/v2/user/me` | 当前用户信息 | 无 |
| GET | `/v2/user/list` | 用户列表（分页和筛选） | Query：`keyword`、`department`、`role`、`page`、`page_size` |
| POST | `/v2/user/me/update` | 更新当前用户资料 | JSON：`username`、`email`、`avatar` |
| POST | `/v2/user/batch` | 批量修改用户部门和职位 | JSON 数组：每项包含 `user_id`、`department_id`、`role_id` |
| POST | `/v2/user/delete` | 删除用户 | JSON：`id` |

### Department（部门）

| 方法 | 路径 | 说明 | 参数 |
|---|---|---|---|
| GET | `/v2/department/list` | 部门列表 | 无 |
| POST | `/v2/department/create` | 新建部门 | JSON：`name`、`type` |
| POST | `/v2/department/update` | 更新部门 | JSON：`department_id`、`name`、`type`、`is_active` |

`type` 目前只允许：`部门`、`协会`。

### Role（职位）

| 方法 | 路径 | 说明 | 参数 |
|---|---|---|---|
| GET | `/v2/role/list` | 职位列表 | 无 |
| POST | `/v2/role/create` | 新建职位 | JSON：`name`、`level`、`type` |
| POST | `/v2/role/update` | 更新职位 | JSON：`role_id`、`name`、`level`、`type`、`is_active` |

`type` 目前只允许：`部门`、`协会`。

### Announcement（公告）

| 方法 | 路径 | 说明 | 参数 |
|---|---|---|---|
| POST | `/v2/announcement/create` | 创建公告草稿 | JSON：`department_id`、`title`、`content` |
| POST | `/v2/announcement/update` | 更新公告 | JSON：`announcement_id`、`title`、`content` |
| POST | `/v2/announcement/push` | 推送公告 | JSON：`announcement_id` |
| GET | `/v2/announcement/list` | 按权限获取公告列表 | Query：`status` 可选，支持 `active`、`history`、`draft`；不传则返回已发布公告 |
| POST | `/v2/announcement/delete` | 删除公告 | JSON：`announcement_id` |

> 当前路由里没有单独的 `/v2/announcement/active` 和 `/v2/announcement/history`，请使用 `/v2/announcement/list?status=active` 或 `/v2/announcement/list?status=history`。

### Term（招新 / 换届周期）

| 方法 | 路径 | 说明 | 参数 |
|---|---|---|---|
| POST | `/v2/term/create` | 创建周期 | JSON：`year`、`type`、`title`、`edit_period`、`query_period` |
| POST | `/v2/term/update` | 更新周期 | JSON：`term_id`、`title`、`edit_period`、`query_period` |
| GET | `/v2/term/list` | 周期列表 | Query：`year`、`type` 可选 |
| POST | `/v2/term/delete` | 删除周期 | JSON：`term_id` |

时间字段格式为日期字符串，后端会按 `Asia/Shanghai` 解析：

```json
{
  "year": 2026,
  "type": "招新",
  "title": "2026年秋季招新",
  "edit_period": {
    "start_at": "2026-09-01",
    "end_at": "2026-09-10"
  },
  "query_period": {
    "start_at": "2026-09-11",
    "end_at": "2026-09-20"
  }
}
```

周期删除规则：

- 只有高权限用户可以删除。
- 已执行的周期不能删除。
- 删除周期会在事务中一并软删除该周期下的申请表、面试官和面试结果。

### Application（申请表）

| 方法 | 路径 | 说明 | 参数 |
|---|---|---|---|
| POST | `/v2/application/create` | 创建申请 | JSON：见下方申请表字段 |
| POST | `/v2/application/update` | 更新当前用户最新申请 | JSON：见下方申请表字段；不需要传 `term_id` |
| GET | `/v2/application/me` | 获取当前用户申请 | 无 |
| GET | `/v2/application/department` | 按职位获取可选部门 | Query：`role_id` 可选 |
| GET | `/v2/application/role` | 按部门获取可选职位 | Query：`department_id` 可选 |
| GET | `/v2/application/list` | 查询周期申请列表 | Query：`term_id` 必填，`department_id` 可选 |
| POST | `/v2/application/delete` | 删除申请表 | JSON：`application_id` |

申请表创建字段：

```json
{
  "term_id": 1,
  "college": "计算机科学与工程学院",
  "major_class": "计科241",
  "gender": "男",
  "phone": "13800000000",
  "qq": "123456789",
  "political_status": "共青团员",
  "birth_date": "2005-09",
  "first_choice": {
    "department_id": 1,
    "role_id": 6
  },
  "second_choice": {
    "department_id": 2,
    "role_id": 6
  },
  "allow_adjust": true,
  "resume": "个人简历 / 在会工作经历",
  "reason": "竞选理由 / 申请阐述"
}
```

申请表规则：

- 创建和更新必须处于周期的编辑时间窗口内。
- `name` 和 `student_id` 来自当前登录用户，不由前端传入。
- `first_choice` 和 `second_choice` 允许相同，但部门与职位类型必须匹配。
- 删除申请时，申请人本人可以删；高权限用户可以删除低权限用户的申请。

### Interviewer（面试官）

| 方法 | 路径 | 说明 | 参数 |
|---|---|---|---|
| POST | `/v2/interviewer/create` | 批量添加面试官 | JSON：`term_id`、`interviewers` |
| POST | `/v2/interviewer/update` | 更新面试官备注 | JSON：`interviewer_id`、`remark` |
| GET | `/v2/interviewer/list` | 面试官列表 | Query：`term_id` 可选 |
| POST | `/v2/interviewer/delete` | 删除面试官 | JSON：`interviewer_id` |

添加面试官示例：

```json
{
  "term_id": 1,
  "interviewers": [
    {
      "user_id": 12,
      "remark": "一面"
    }
  ]
}
```

面试官规则：

- 只有高权限用户可以创建、更新、删除面试官。
- 已执行的周期不能再修改面试官。
- 创建时后端会根据 `user_id` 自动回填该用户的部门。
- 同一周期内同一用户不能重复成为面试官；重复项会被跳过，若全部重复会返回错误。
- 普通面试官只能查看自己有权限范围内的面试官列表；高权限用户可查看全量。

### Interview Result（面试结果）

| 方法 | 路径 | 说明 | 参数 |
|---|---|---|---|
| POST | `/v2/interviewer/result/create` | 创建面试结果 | JSON：`application_id`、`decision`、`result_department_id`、`result_role_id`、`remark` |
| POST | `/v2/interviewer/result/update` | 更新面试结果 | JSON：`interview_result_id`、`decision`、`result_department_id`、`result_role_id`、`remark` |
| GET | `/v2/interviewer/result/decision` | 获取面试结果决策枚举 | 无 |

`decision` 支持：

- `录取第一志愿`
- `录取第二志愿`
- `已调剂`
- `未通过`

面试结果规则：

- 创建和更新必须处于周期的查询 / 结果填报时间窗口内。
- `录取第一志愿` 时，最终部门 / 职位必须等于申请表第一志愿。
- `录取第二志愿` 时，最终部门 / 职位必须等于申请表第二志愿。
- `已调剂` 时，申请表必须允许调剂，且最终部门 / 职位必须合法匹配。
- `未通过` 时，`result_department_id` 和 `result_role_id` 必须传 `0`。
- 更新面试结果时，只允许改决策、最终部门、最终职位、操作人和备注，不允许改变关联的申请表、周期或用户。
- `executed_at` 为空表示草稿，非空表示已经执行后的历史结果。

### Upload（文件上传）

| 方法 | 路径 | 说明 | 参数 |
|---|---|---|---|
| POST | `/v2/upload/image` | 上传图片 | `multipart/form-data`：`scene`、`file` |
| POST | `/v2/upload/file` | 上传通用文件 | `multipart/form-data`：`scene`、`file` |

上传成功返回：

```json
{
  "uri": "scene/uuid.png",
  "url": "临时访问链接"
}
```

图片接口支持的后缀：`.jpg`、`.jpeg`、`.png`、`.webp`、`.gif`、`.avif`。通用文件接口不限制后缀，只校验文件大小。

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
- 当前 JWT 生成逻辑使用配置 `jwt.expire_minute`，默认值 `20` 表示约 20 分钟。

### 验证码

- 验证码存储在 Redis，支持过期和发送冷却。
- `auth/send` 和 `password/reset` 的 `scene` 必须保持一致。
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
  设计清单.md
  错误.md
  INTERVIEW_AND_SOFT_DELETE_REVIEW.md
README.md
```

## 运行方式

### 1. 准备依赖

- MySQL
- Redis
- MinIO 或兼容 S3 的对象存储

### 2. 修改配置

配置文件：`configs/config.yaml`

按本地环境修改：

- `mysql`
- `redis`
- `jwt`
- `email`
- `server`
- `minio`

### 3. 启动项目

```bash
go run ./cmd/main.go
```

### 4. 本地检查

```bash
GOCACHE=/private/tmp/gocache go test ./...
GOCACHE=/private/tmp/gocache go vet ./...
```

## 备注

- 当前项目以协会内部自用场景为主。
- 接口权限、业务规则和时间窗口均已在代码中实现。
