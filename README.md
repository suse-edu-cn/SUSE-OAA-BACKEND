# SUSE OAA Backend

SUSE OAA 后端服务，基于 **Go + Gin + GORM + MySQL + Redis**，面向协会账号、组织架构、公告以及招新 / 换届业务。

## 当前进度

项目目前已完成以下核心模块：

- **认证与账号安全**：注册、登录、双存储 Cache-Aside 刷新 Token、登出、修改密码、发送邮箱验证码、验证码重置密码
- **用户模块**：当前用户信息、用户列表（多条件分页筛选与头像预签名）、更新用户资料、删除用户（级联清理 Token）、权限防穿透批量修改部门与职位
- **基础数据**：部门列表 / 新建 / 更新，职位列表 / 新建 / 更新，种子数据自动初始化
- **公告模块**：创建、更新、推送、删除、按权限获取公告列表，富文本与 Markdown 中 `oss://` 链接自动解析预签名
- **招新 / 换届模块**：周期创建 / 更新 / 删除 / 公开列表查询、申请表提交 / 更新 / 删除 / 查询、志愿下拉数据、面试官创建 / 更新 / 删除 / 查询、面试结果创建 / 更新 / 决策枚举查询 / 定时常驻任务到期自动执行
- **文件与对象存储模块**：图片与通用文件上传、双端点（内网直连/公网签名）隔离、本地纯算离线预签名、文件后缀白名单校验
- **系统底层架构**：全链路 HTTP Request Context 超时与取消透传，覆盖 MySQL (GORM)、Redis 及 MinIO

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
| POST | `/v2/auth/password/reset` | 验证码重置密码（公开） | JSON：`account`、`code`、`password` |

### User（用户）

| 方法 | 路径 | 说明 | 参数 |
|---|---|---|---|
| GET | `/v2/user/me` | 当前用户信息 | 无 |
| GET | `/v2/user/list` | 用户列表（分页和筛选） | Query：`keyword`、`department_id`、`role_id`、`department`、`role`、`page`、`page_size`、`is_all` |
| POST | `/v2/user/me/update` | 更新当前用户资料 | JSON：`username`、`email`、`avatar` |
| POST | `/v2/user/batch` | 批量修改用户部门和职位 | JSON 数组：每项包含 `user_id`、`department_id`、`role_id` |
| POST | `/v2/user/delete` | 删除用户 | JSON：`user_id` |

> - 用户资料里的 `avatar` 字段存的是对象存储中的资源路径；`GET /v2/user/me` 和 `GET /v2/user/list` 返回时均会由后端自动转换为临时访问链接 `url`。若当前头像缺失或失效，后端会回退到默认头像 `avatar/default.png`。
> - `GET /v2/user/list` 支持灵活筛选：`department` 和 `role` 参数既可传名称字符串也可传数字 ID；`page` 默认 `1`，`page_size` 默认 `20`（单页上限 `100`）；`is_all=true` 时可查看包括停用人员在内的全量名单。
> - `POST /v2/user/delete` 权限与注销规则：
>   - **本人自主注销**：进入**注销冷静期**（常量 `UserDeletionGracePeriod = 24h`，目前为 1 天）。首次请求标记注销到期时间，返回 `200`（`{"code": 200, "message": "success", "data": null}`）；冷静期内账号权益与正常用户完全一致。
>   - **冷静期内重复请求**：返回 `400`，`message` 为 `"账号处于冷静期"`，`data` 返回到期彻底注销的时间（格式为 `YYYY-MM-DD HH:mm:ss`，如 `{"code": 400, "message": "账号处于冷静期", "data": "2026-09-29 15:27:00"}`）。
>   - **到期自动执行**：倒计时结束后由后台定时任务（每分钟轮询）或用户再次请求时自动彻底删除（再次请求若已过冷静期直接返回 `200` 成功）。
>   - **管理员强制删除**：副会长及以上（`level >= 80`）可直接删除级别低于自身的用户，无需等待冷静期，立即执行级联删除，返回 `200`。
>   - **级联清理**：账号彻底删除时触发级联清理，同步清除该用户在 Redis 缓存与 MySQL `refresh_tokens` 表中全设备的所有 Refresh Token，并清理可能残留的验证码与冷却标记。

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
| GET | `/v2/announcement/get` | 获取公告详情 | Query：`announcement_id` 必填 |
| POST | `/v2/announcement/delete` | 删除公告 | JSON：`announcement_id` |

> - 当前路由里没有单独的 `/v2/announcement/active` 和 `/v2/announcement/history`，请使用 `/v2/announcement/list?status=active` 或 `/v2/announcement/list?status=history`。
> - 公告内容（Markdown 或 HTML）中支持直接嵌入内部对象路径（例如 `oss://oaa-img/announcement/xxx.png` 或 `oss://oaa-file/...`）。后端在返回列表详情时会自动扫描并批量生成带有效期的预签名公网 URL。

### Term（招新 / 换届周期）

| 方法 | 路径 | 说明 | 参数 |
|---|---|---|---|
| POST | `/v2/term/create` | 创建周期 | JSON：`year`、`type`、`title`、`edit_period`、`query_period` |
| POST | `/v2/term/update` | 更新周期 | JSON：`term_id`、`title`、`edit_period`、`query_period` |
| GET | `/v2/term/list` | 周期列表（公开给所有登录用户） | Query：`year`、`type` 可选 |
| POST | `/v2/term/delete` | 删除周期 | JSON：`term_id` |

时间字段格式为日期字符串（如 `2026-09-01`），后端会按 `Asia/Shanghai` 解析；`type` 目前只允许 `招新` 或 `换届`。

创建周期请求示例：

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

`GET /v2/term/list` 返回示例：

```json
{
  "id": 1,
  "year": 2026,
  "type": "招新",
  "title": "2026 年秋季招新",
  "edit_period": {
    "start_at": "2026-09-01",
    "end_at": "2026-09-15"
  },
  "query_period": {
    "start_at": "2026-09-20",
    "end_at": "2026-09-30"
  },
  "is_executed": false,
  "execute_after_at": "2026-10-01T00:00:59+08:00",
  "executed_at": null,
  "created_at": "2026-09-07T16:03:58.471+08:00",
  "updated_at": "2026-09-07T16:03:58.471+08:00"
}
```

周期权限与删除规则：

- `GET /v2/term/list` 面向所有登录用户开放，方便普通会员随时了解招新/换届时间表并参与申请。
- 创建、更新与删除周期仅限高权限用户（`level >= 80`）。
- 已执行完成的周期（`is_executed = true`）不能删除。
- 删除周期会在数据库事务中一并软删除该周期下的申请表、面试官和面试结果。

周期结果执行规则：

- 创建或更新周期时，后端会自动把 `execute_after_at` 设置为 `query_end_at + 1分钟`。
- 服务启动后会立刻扫描一次到期周期，之后每分钟扫描一次。
- 扫描条件为：`is_executed = false` 且 `execute_after_at <= 当前时间`。
- 执行时会把该周期下尚未执行的 `interview_results` 同步到 `applications` 的最终结果字段。
- 决策为 `录取第一志愿`、`录取第二志愿` 或 `已调剂` 时，会同步更新对应用户的 `department_id` 和 `role_id`；`未通过` 只同步申请表结果，不改变用户当前部门 / 职位。
- 每条面试结果执行时会记录用户执行前的 `old_department_id` / `old_role_id`，并写入 `interview_results.executed_at`。
- 周期全部执行成功后会写入 `terms.is_executed = true` 和 `terms.executed_at`；如果执行中途失败，事务会回滚，不会把周期标记为已执行，下一轮扫描会继续重试。

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
  "avatar": "application/uuid.png",
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
- `avatar` 存申请表照片在对象存储中的资源路径，上传图片时请使用 `scene=application`；没有照片时传空字符串或不传，后端会保持为空且不会回退默认头像。
- `GET /v2/application/me` 和 `GET /v2/application/list` 返回的 `avatar` 会包含 `uri` 和临时访问 `url`，没有照片时二者为空。
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
| POST | `/v2/interviewer/result/update` | 更新面试结果 | JSON：`application_id`、`decision`、`result_department_id`、`result_role_id`、`remark` |
| GET | `/v2/interviewer/result/list` | 查询面试结果列表 | Query：`term_id` 可选；不传返回全部 |
| GET | `/v2/interviewer/result/decision` | 获取面试结果决策枚举 | 无 |

`decision` 支持：

- `录取第一志愿`
- `录取第二志愿`
- `已调剂`
- `未通过`

面试结果规则：

- `GET /v2/interviewer/result/list` 可以按 `term_id` 查询某个周期的面试结果；不传 `term_id` 时返回全部面试结果。
- 创建和更新必须处于周期的查询 / 结果填报时间窗口内。
- `录取第一志愿` 时，最终部门 / 职位必须等于申请表第一志愿。
- `录取第二志愿` 时，最终部门 / 职位必须等于申请表第二志愿。
- `已调剂` 时，申请表必须允许调剂，且最终部门 / 职位必须合法匹配。
- `未通过` 时，`result_department_id` 和 `result_role_id` 必须传 `0`。
- 更新面试结果时通过 `application_id` 定位面试结果，只允许改决策、最终部门、最终职位、操作人和备注，不允许改变关联的申请表、周期或用户。
- `executed_at` 为空表示草稿，非空表示已经执行后的历史结果。
- 面试结果在查询期内只是草稿；到 `execute_after_at` 后由后台执行器统一生效。

### Upload（文件上传）

| 方法 | 路径 | 说明 | 参数 |
|---|---|---|---|
| POST | `/v2/upload/image` | 上传图片 | `multipart/form-data`：`scene`、`file` |
| POST | `/v2/upload/file` | 上传通用文件 | `multipart/form-data`：`scene`、`file` |

申请表照片请调用图片上传接口并传 `scene=application`；用户头像仍使用 `scene=avatar`。

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

| ID | 角色 | level | 类型 | 职责说明 |
|---|---|---:|---|---|
| 1 | 开发者 | 100 | 协会 | 系统维护与超级管理员 |
| 2 | 会长 | 90 | 协会 | 协会最高负责人 |
| 3 | 副会长 | 80 | 协会 | 协会主管负责人，拥有全量业务审批权限 |
| 4 | 部长 | 60 | 部门 | 部门负责人，管理本部门内部日常事务 |
| 5 | 副部长 | 50 | 部门 | 部门协助负责人 |
| 6 | 干事 | 20 | 部门 | 部门日常执行成员 |
| 7 | 会员 | 10 | 协会 | 普通会员，新用户注册后的默认角色 |

### 基础部门

启动时由系统自动初始化以下部门种子数据：

| ID | 部门名称 | 类型 | 说明 |
|---|---|---|---|
| 1 | 算法竞赛部 | 部门 | 负责算法日常培训、集训与各类竞赛组织 |
| 2 | 组织宣传部 | 部门 | 负责协会宣传物料、新媒体运营与活动策划 |
| 3 | 秘书处 | 部门 | 统筹文档、活动统筹、财务与协会行政事务 |
| 4 | 理事会 | 部门 | 协会重大事项审议与指导机构 |
| 5 | 项目实践部 | 部门 | 负责协会内外技术研发、工程落地与项目孵化 |
| 6 | 开放原子开源协会 | 协会 | 协会顶层机构，新用户注册后默认归属 |

### 权限级别阶梯说明

- **高权限管理者（`level >= 80`：副会长、会长、开发者）**：
  - 招新 / 换届周期的创建、修改与删除
  - 面试官的批量添加、更新备注与移除
  - 查看全协会所有部门的申请表与面试结果列表
  - 跨部门批量调整人员部门与职位（可处理低于自身职位的用户）
  - 用户账号删除与注销（仅能删除职位低于自身的用户）
  - 查看所有部门的草稿公告、当前公告与历史归档公告
- **部门管理者（`50 <= level < 80`：副部长、部长）**：
  - 仅限调整本部门内部成员的职位（不可操作同级或更高职位的成员，严禁跨部门修改外部门成员）
  - 查看本部门名下的申请表与面试结果
  - 查看本部门内部的草稿公告与已发布公告
- **普通成员（`level < 50`：干事、会员）**：
  - 查看全协会已发布的公共公告
  - 查看招新/换届周期开放列表，提交与更新个人的申请表，查询个人申请记录

### 批量修改规则（`POST /v2/user/batch`）

- **幂等去重**：同一批次请求中同一个 `user_id` 仅处理第一次出现的记录。
- **状态检查**：操作者自身的职位或所属部门若处于停用状态（`is_active = false`），直接拒绝批量修改。
- **防权限倒挂**：严禁修改同级或更高职位的用户（例如部长无法修改其他部长、副会长或会长）。
- **防越权改派**：低于副会长的管理者（如部长），严禁跨部门修改其他部门的用户。
- **防越权升迁**：严禁将目标用户的职位提升至同级或高于操作者当前的职位。
- **组织架构类型校验**：部门与职位类型必须合法匹配（协会级职位只能归属于“开放原子开源协会”，部门级职位只能归属于各业务部门）。
- **部分错误返回**：若有部分条目未通过校验，响应状态码为 `400`，`data` 数组中返回具体失败项及 `error_message` 详细原因。

### 公告可见性

- 职位等级 `< 30`：只能看到已发布公告。
- 职位等级 `30 - 79`：可以看到本部门公告及所有已发布公告。
- 职位等级 `>= 80`：可以看到全部公告，包括草稿。
- 同一部门同时只有一条当前生效公告；推送新公告会使原公告变为历史公告。

## Token 与系统架构

### Refresh Token 双存储与 Cache-Aside

- **双写机制**：用户登录成功后生成 `refresh_token`，同时写入 Redis（键名：`<user_id>-<device>`，TTL 默认 15 天）并持久化到 MySQL `refresh_tokens` 表中。
- **Cache-Aside 容灾回源**：调用 `POST /v2/auth/refresh` 刷新令牌时，优先读取 Redis 缓存；若 Redis 发生重启或缓存未命中，系统自动回源查询 MySQL 持久化表，验证通过后重新写回 Redis 缓存，保证服务高可用与零断连。
- **级联失效**：用户登出时立即删除对应设备的 Redis 与 MySQL 记录；当用户被管理员删除（`POST /v2/user/delete`）时，系统自动清理该用户在所有设备上的全部 Refresh Token，杜绝幽灵会话。
- **访问控制**：JWT Access Token 有效期由配置项 `jwt.expire_minute` 控制（默认 20 分钟）。

### 验证码频控与安全

- 验证码存储在 Redis，具备自动过期（默认 5 分钟）与发送冷却机制（默认 1 分钟）。
- `POST /v2/auth/send` 传入 `account` 和 `scene`（支持通过用户名、学号或邮箱触发发送）。
- `POST /v2/auth/password/reset` 重置密码成功后，Redis 验证码立即删除作废，并由后端使用 `bcrypt` 重新加密持久化新密码。

### 全链路 Context 生命周期管理

- **取消与超时穿透**：所有 HTTP 接口统一从 Gin 提取 `c.Request.Context()`，并深度贯穿 Service 层、Repository 层以及 MinIO 存储交互。
- **底层协同**：
  - 关系型数据库：GORM 事务与 SQL 执行统一挂载 `.WithContext(ctx)`。
  - 缓存层：`go-redis/v9` 所有读写命令均绑定请求上下文。
  - 对象存储：MinIO 文件的元数据获取与流上传/删除绑定请求上下文。
- 当客户端提前关闭连接、断网或请求超时，服务端将立即感知并取消后续的数据库查询与网络 I/O，杜绝资源悬挂与死锁隐患。

## 技术栈

- **语言与核心运行时**：Go (1.23+)
- **Web 框架**：Gin
- **持久化 ORM**：GORM + MySQL
- **内存缓存**：Redis (`github.com/redis/go-redis/v9`)
- **对象存储**：MinIO SDK (`github.com/minio/minio-go/v7`)
- **认证鉴权**：JWT (`github.com/golang-jwt/jwt/v5`)
- **邮件服务**：Gomail (`gopkg.in/gomail.v2`)
- **安全哈希**：`golang.org/x/crypto/bcrypt`

## 项目结构

```text
cmd/
  main.go
configs/
  config.yaml
  config_example.yaml
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

## 配置文件说明

项目启动时会从 `configs/config.yaml` 读取配置。首次部署可以先复制示例文件：

```bash
cp configs/config_example.yaml configs/config.yaml
```

`config_example.yaml` 只保留配置结构和空值，不包含任何真实凭据。各字段说明如下：

### `server`

| 字段 | 类型 | 说明 |
|---|---|---|
| `host` | string | 服务监听地址，例如 `0.0.0.0`。 |
| `port` | string | 服务监听端口，例如 `8080`。 |
| `mode` | string | Gin 运行模式，常见值为 `debug`、`release`、`test`。当前配置结构未读取该字段，实际模式需由代码或环境设置。 |

### `mysql`

| 字段 | 类型 | 说明 |
|---|---|---|
| `host` | string | MySQL 主机地址或域名。 |
| `port` | string | MySQL 端口，通常为 `3306`。 |
| `username` | string | MySQL 用户名。 |
| `password` | string | MySQL 密码。 |
| `database` | string | 要连接的数据库名。 |
| `charset` | string | 字符集，通常为 `utf8mb4`。当前配置结构未读取该字段；数据库连接参数由代码固定拼接。 |

### `jwt`

| 字段 | 类型 | 说明 |
|---|---|---|
| `secret` | string | JWT 签名密钥，请使用足够长且随机的字符串，并妥善保管。 |
| `expire_minute` | integer | Access Token 有效期，单位为分钟。 |
| `refresh_time` | integer | Refresh Token 有效期，具体单位取决于项目当前实现。 |

### `redis`

| 字段 | 类型 | 说明 |
|---|---|---|
| `host` | string | Redis 主机地址或域名。 |
| `port` | integer | Redis 端口，通常为 `6379`。 |
| `password` | string | Redis 密码；无密码时填写空字符串。 |
| `database` | integer | Redis 逻辑数据库编号，通常为 `0`。 |

### `email`

| 字段 | 类型 | 说明 |
|---|---|---|
| `host` | string | SMTP 服务器地址。 |
| `port` | integer | SMTP 服务器端口，例如 SSL 常用 `465`。 |
| `user` | string | SMTP 登录账号及发件人邮箱。 |
| `pass` | string | SMTP 登录密码或授权码。 |
| `cool_down` | integer | 同一账号再次发送验证码前的冷却时间，单位为分钟。 |
| `expire` | integer | 邮箱验证码有效期，单位为分钟。 |

### `minio`

| 字段 | 类型 | 说明 |
|---|---|---|
| `minio_endpoint` | string | 后端与 MinIO 内部通信端点，例如 `localhost:9000` 或内网域名 `obj.in.suseoaa.com`。用于上传、元数据探测与删除，不消耗公网流量。 |
| `public_endpoint` | string | 前端或外部访问的公网端点/域名，例如 `obj.suseoaa.com`。用于生成对外预签名临时访问 URL；留空则默认同 `minio_endpoint`。 |
| `minio_region` | string | MinIO 区域代码，需与 MinIO 服务端配置一致（如 `cn-west-yb0`，未配置时默认为 `cn-west-yb0`）。 |
| `minio_access_key` | string | MinIO Access Key。 |
| `minio_secret_key` | string | MinIO Secret Key。 |
| `minio_use_ssl` | boolean | 后端与内部 MinIO 通信是否使用 HTTPS。 |
| `public_use_ssl` | boolean | 外部公网访问生成的预签名链接是否使用 HTTPS（外网配置 SSL 证书时需设为 `true`）。 |
| `minio_img_bucket` | string | 用于保存图片资源的 Bucket 名称（如 `oaa-img`）。 |
| `minio_file_bucket` | string | 用于保存通用文件的 Bucket 名称（如 `oaa-file`）。 |
| `max_file_size` | integer | 普通文件允许的最大大小，单位为 MB。 |
| `max_image_size` | integer | 图片允许的最大大小，单位为 MB。 |
| `expire_time` | integer | 对象存储临时访问链接有效期，单位为分钟。 |

> `mode` 和 `charset` 当前会出现在 YAML 示例中，但没有对应的配置结构字段，因此修改它们不会改变当前程序行为。
> `minio_endpoint` 与 `public_endpoint` 支持带或不带 `http://` / `https://`，后端初始化时会自动识别清洗并校正 SSL 模式。预签名客户端已显式配置 Region（默认 `cn-west-yb0`），预签名链接纯本地离线计算生成，无需向公网反向代理发送 BucketLocation 探测请求，彻底避免 502 及 Region 不匹配错误。

## Makefile 命令说明

Makefile 位于项目根目录，用于快速编译、检查和运行项目。直接执行 `make` 等同于执行 `make linux`。

| 命令 | 说明 |
|---|---|
| `make` | 编译 Linux `amd64` 可执行文件，编译完成后重命名为 `bin/OAAbeta`。 |
| `make linux` | 编译 Linux 可执行文件。默认使用 `GOARCH=amd64`、`CGO_ENABLED=0`，适合生成便于部署的静态二进制文件。 |
| `make build` | `make linux` 的别名。 |
| `make clean` | 删除 `bin/` 构建目录及其中的可执行文件。 |
| `make test` | 执行 `go test ./...`。 |
| `make vet` | 执行 `go vet ./...`。 |
| `make run` | 使用 `configs/config.yaml` 启动开发服务。 |
| `make help` | 显示 Makefile 中的可用目标。 |

可以通过变量覆盖默认构建参数，例如交叉编译 Linux ARM64：

```bash
make linux GOARCH=arm64
```

也可以自定义临时编译名称或输出目录；最终文件仍会重命名为 `OAAbeta`：

```bash
make linux APP_NAME=my-service BUILD_DIR=dist
# 输出：dist/OAAbeta
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
go test ./...
go vet ./...
# 或直接使用 Makefile 目标：
make vet
make test
```

## 备注

- 当前项目以协会内部自用场景为主。
- 接口权限、业务规则和时间窗口均已在代码中实现。
