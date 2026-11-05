# SUSE OAA 后端接口与业务变更通知（面向 Web 端与移动端）

更新日期：2026-10-31  
适用版本：v2.1.13+  
基准版本：v2.1.6（2026-10-29 发布版本）  
对接对象：Web 前端、移动端（iOS / Android / Flutter / 小程序）

---

## 一、 变更背景说明

自 2026-10-29（v2.1.6）组织架构与毕业流转机制落地后，本次更新聚焦于**招新换届面试管理闭环**、**面试结果全量与自选 Excel 导出**、**查询参数防陷阱治理**、**申请表隐私安全脱敏**以及**全链路可观测性日志升级**：

1. **面试管理路由与生命周期收归**：统一收归面试结果相关接口至 `/v2/interviewer/result/*` 组，申请表与面试结果表建立强一致双向事务联动。
2. **新增面试结果 Excel 导出与元数据定义接口**：
   - 支持副会长及以上管理随时导出完整面试结果与申请表数据，不受周期时间窗口限制；
   - 引入 **64 位状态压缩掩码（Bitmask Compression）** 机制，前端按需勾选字段导出，提供元数据查询接口实现前端零维护。
3. **查询参数解析防陷阱优化**：
   - 修复前端或接口调试工具留空查询参数（如 `?has_result=`）时被底层解析框架误赋零值 `false` 的问题，统一支持空值、`null`、`all` 场景的全量查询。
4. **个人申请隐私脱敏与填报规则完善**：
   - 周期查询公示期前，申请人查询本人申请表时对录取决策进行脱敏保护，避免录用结果提前泄露；
   - 新增业务规则：普通会员用户不可填报成为“会员”职位；骨干人员允许填报“会员”以便换届卸任。
5. **全链路用户追踪与日志规范化**：
   - 服务端访问日志与异常链路自动注入 `user_id` 与 `trace_id`；
   - 严格规范 200 / 4xx / 5xx 状态码与日志告警级别的映射。

---

## 二、 新增接口规范（面试结果导出与字段定义）

### 1. 导出面试结果与申请表 Excel 文件

* **接口路径**：`GET /v2/interviewer/result/export`
* **鉴权与权限限制**：需携带 JWT Token，且仅限在任**副会长及以上管理者**（`level >= 80` 且 `is_graduated == false`，即会长、副会长、开发者）。部长（Level 60）、副部长（Level 50）、普通干事（Level 20）、普通面试官或已毕业成员调用均返回 `403 Forbidden`。
* **时间窗口限制**：**无时间限制**（不受周期的报名修改期 `edit_period` 或评审公示期 `query_period` 时间窗口约束，招新换届期间或周期归档后随时可导出）。
* **数据处理与状态**：
  - **默认全量导出**：无论申请是否已录入面试结果均会导出；未录入结果的申请，其面试决策显示为“待定”，审核人显示为“-”，录取部门与职位显示为“-”；
  - **部门筛选联动**：传 `department_id` 时按部门筛选（凡第一志愿、第二志愿或最终录取部门中任一命中该部门的考生均会导出）；
* **字段范围与限制**：
  - 目前内置 22 个固定业务字段，默认全选导出；
  - 传参仅支持 22 位 `010101` 状态位序列字符串；全选为 22 位全 1（`"1111111111111111111111"`）；若传参自选字段，必须至少命中 1 个有效字段，全 0 字段参数将返回 `400 "未选择任何有效导出字段"`；包含非 0/1 字符将返回 `400 "导出字段参数不合法"`；
  - **当前不参与导出的字段**：个人简历/工作经历（`resume`）、申请理由/竞选阐述（`reason`）、是否服从调剂（`allow_adjust`）、证件照路径（`avatar_uri`）以及自增主键（`id`）；
* **跨域与文件命名约定**：
  - 传入有效 `term_id` 时文件名为 `"<周期标题>_面试结果.xlsx"`，不传或传 0 时为 `"面试结果.xlsx"`；
  - 后端已在 `Access-Control-Expose-Headers` 中显式暴露 `Content-Disposition`，前端跨域下载可通过正则解析获取 UTF-8 真实文件名。

#### Query 参数列表：

| 参数名 | 类型 | 必填 | 默认值 | 说明 |
|---|---|:---:|---|---|
| `term_id` | `uint64` | 否 | `0` | 周期 ID。传指定周期 ID 时导出该周期申请并将文件命名为 `"<周期标题>_面试结果.xlsx"`；不传或传 0 时导出全部周期，命名为 `"面试结果.xlsx"`。若传入不存在的 ID 返回 `400 "周期不存在"`。 |
| `department_id` | `uint64` | 否 | `0` | 部门 ID。按部门筛选数据（包含一志愿、二志愿及调剂/录取到该部门的考生）。传入不存在的部门 ID 返回 `400 "部门不存在"`。 |
| `fields` | `string` | 否 | 默认全选 | **01 状态位序列**。未传或传空时默认全选导出所有 22 个字段。直接传由 `'0'` 和 `'1'` 组成的字符串（如 `"0011000000000001000000"`），从左至右第 `i` 位字符（下标 0~21）对应固定字段列表中的第 `i` 个字段，`'1'` 导出，`'0'` 不导出。全选为 22 位全 1：`"1111111111111111111111"`。不兼容数值掩码，传非 0/1 字符返回 400。 |

#### 响应格式：
- `Content-Type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`
- `Content-Disposition: attachment; filename="<周期标题>_面试结果.xlsx"; filename*=UTF-8''<URL_ENCODED_FILENAME>`
- 文件格式为标准 `.xlsx`，内置深蓝专业表头、隔行斑马纹、自适应列宽与文字左右对齐排版。

---

### 2. 获取导出字段元数据定义字典

前端在弹出“导出选项弹窗”或渲染可选列复选框时，先调用此接口获取动态字段列表与全选掩码，无需在前端硬编码任何字段字典。

* **接口路径**：`GET /v2/interviewer/result/fields`
* **鉴权要求**：登录用户均可获取
* **请求方式**：`GET`
* **Query 参数**：无

#### 响应体示例 (JSON)：
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "total_fields": 22,
    "all_bits": "1111111111111111111111",
    "fields": [
      { "index": 0, "label": "业务周期" },
      { "index": 1, "label": "业务类型" },
      { "index": 2, "label": "学号" },
      { "index": 3, "label": "姓名" },
      { "index": 4, "label": "性别" },
      { "index": 5, "label": "学院" },
      { "index": 6, "label": "专业班级" },
      { "index": 7, "label": "政治面貌" },
      { "index": 8, "label": "出生年月" },
      { "index": 9, "label": "联系电话" },
      { "index": 10, "label": "QQ" },
      { "index": 11, "label": "第一志愿部门" },
      { "index": 12, "label": "第一志愿职位" },
      { "index": 13, "label": "第二志愿部门" },
      { "index": 14, "label": "第二志愿职位" },
      { "index": 15, "label": "面试决策" },
      { "index": 16, "label": "录取部门" },
      { "index": 17, "label": "录取职位" },
      { "index": 18, "label": "审核人" },
      { "index": 19, "label": "决策备注" },
      { "index": 20, "label": "提交时间" },
      { "index": 21, "label": "更新时间" }
    ]
  }
}
```

#### 严格规定的 22 导出字段定义说明表（位位置严格固定）：

| 字段下标（Index / 0-based） | 对应 01 序列字符位置 | 导出列名（Label） | Excel 排版对齐 |
|:---:|:---:|:---|:---:|
| 0 | 第 1 位 | 业务周期 | 居中 |
| 1 | 第 2 位 | 业务类型 | 居中 |
| 2 | 第 3 位 | 学号 | 居中 |
| 3 | 第 4 位 | 姓名 | 居中 |
| 4 | 第 5 位 | 性别 | 居中 |
| 5 | 第 6 位 | 学院 | 居左 |
| 6 | 第 7 位 | 专业班级 | 居中 |
| 7 | 第 8 位 | 政治面貌 | 居中 |
| 8 | 第 9 位 | 出生年月 | 居中 |
| 9 | 第 10 位 | 联系电话 | 居中 |
| 10 | 第 11 位 | QQ | 居中 |
| 11 | 第 12 位 | 第一志愿部门 | 居中 |
| 12 | 第 13 位 | 第一志愿职位 | 居中 |
| 13 | 第 14 位 | 第二志愿部门 | 居中 |
| 14 | 第 15 位 | 第二志愿职位 | 居中 |
| 15 | 第 16 位 | 面试决策 | 居中 |
| 16 | 第 17 位 | 录取部门 | 居中 |
| 17 | 第 18 位 | 录取职位 | 居中 |
| 18 | 第 19 位 | 审核人 | 居中 |
| 19 | 第 20 位 | 决策备注 | 居左 |
| 20 | 第 21 位 | 提交时间 | 居中 |
| 21 | 第 22 位 | 更新时间 | 居中 |

> **字段过滤说明**：按照需求，个人简历（`resume`）、申请理由（`reason`）、是否调剂（`allow_adjust`）以及自增序号不参与默认导出列表。

---

## 三、 面试管理接口规范与路由统一（Breaking Changes）

为解决历史接口分散、路径兼容冗余的问题，**面试结果的所有管理操作全部收归至 `/v2/interviewer/result` 统一前缀**，旧别名路径已下线：

| HTTP 方法 | 接口路径 | 权限要求 | 接口功能与联动机制 |
|---|---|---|---|
| `POST` | `/v2/interviewer/result/create` | 面试官 / 管理员 | **录入面试结果**：事务内同时更新 `interview_results` 表并同步写回 `applications` 申请表，记录 `decision`、`result_department_id`、`result_role_id`、`operator_id` 与 `decision_remark`。 |
| `POST` | `/v2/interviewer/result/update` | 面试官 / 管理员 | **更新面试结果**：修改已录入的录取决策、拟录用部门、角色或备注，同步写回申请表。 |
| `POST` | `/v2/interviewer/result/delete` | 副会长及以上 | **删除面试决策（重置回退）**：请求体传 `{"application_id": <id>}`。软删除面试结果记录，并在事务内**自动将申请表重置为 `decision: "待定"`，清空拟录用部门与角色，清空审核人与备注**。删除后该申请表允许重新录入。 |
| `GET` | `/v2/interviewer/result/list` | 面试官 / 管理员 | **获取面试结果列表**：支持传入 `?term_id=<id>` 筛选。 |
| `GET` | `/v2/interviewer/result/decision` | 登录用户 | **获取合法决策枚举**：返回 `["录取第一志愿", "录取第二志愿", "未通过", "已调剂"]`。 |
| `GET` | `/v2/interviewer/result/export` | 副会长及以上 | **导出 Excel 文件**（详见第二节）。 |
| `GET` | `/v2/interviewer/result/fields` | 登录用户 | **查询导出字段元数据定义**（详见第二节）。 |

---

## 四、 既有接口优化与防陷阱治理

### 1. 申请表列表查询接口 (`GET /v2/application/list`) 参数优化与姓名筛选

* **新增 `name` 姓名筛选参数（支持右模糊前缀匹配）**：
  - 新增可选 Query 参数 `name`，支持按申请人姓名进行**前缀右模糊匹配**（SQL: `WHERE name LIKE 'name%'`）；
  - **匹配示例**：传 `?name=王` 可匹配“王五”、“王小明”等以“王”开头的候选人，不会错误匹配“小王”或“老王”，无需两端通配；传空或不传时不做姓名过滤。
* **`has_result` 参数解析防陷阱优化**：
  - 原实现中使用布尔指针绑定，前端组件或调试工具（如 Apifox）在勾选了 `has_result` 复选框但参数值为空（发送 `?has_result=`）时，框架会默认将其解析为 `false`，导致“默认不填参数全查”意外变成了“只查未录入/待定申请”。
* **普通面试官跨部门查询权限严格校验（Breaking Change）**：
  - **全权限角色（会长 Level 90、副会长 Level 80、开发者 Level 100）**：可查看所有部门的全部申请表；若传入 `?department_id=X` 则精确筛选该部门；
  - **全协会面试官（周期中委派部门为“开放原子开源协会” ID: 6）**：具备协会全域评审权限，支持跨部门查阅所有申请表或筛选指定部门；
  - **普通部门面试官（如技术部 ID: 1）**：
    - 不传 `department_id` 或传 `0`，默认只返回本人所属部门的申请表；
    - 传入本人所属部门 ID，正常返回本部门申请表；
    - **若在前端切换到非本人所属部门（传入其他部门 ID）**：后端**立即返回 HTTP 403 Forbidden 显式拦截**，错误提示为 `"无权限查看其他部门申请"`（原逻辑为静默覆盖返回本部门，导致前端切换部门选项后界面仍显示原部门数据）。

* **优化后行为规范**：

| 请求传参形式 | 服务端解析结果 | 对应查询行为 |
|---|:---:|---|
| 未传 `has_result` 参数 | `nil` | **全量查询**（不设过滤，待定与已录入全部返回） |
| `?has_result=`（留空） | `nil` | **全量查询**（安全防护，不设过滤） |
| `?has_result=null` / `undefined` / `all` | `nil` | **全量查询**（兼容各类前端组件未选择状态） |
| `?has_result=true` 或 `?has_result=1` | `&true` | **仅查询已录入面试结果**（已审核） |
| `?has_result=false` 或 `?has_result=0` | `&false` | **仅查询未录入面试结果**（待定中） |
| 传其他非法字符串（如 `?has_result=abc`） | 抛出错误 | 返回 `400 "has_result 参数不合法"` |

> **同类防护覆盖**：成员管理接口 `GET /v2/user/list` 中的 `is_graduated` 与 `is_all` 参数亦已接入该统一防陷阱解析器。

---

### 2. 个人申请表查询接口 (`GET /v2/application/me`) 隐私脱敏

* **脱敏规则**：
  - 当申请人查询自己的申请表时，**在周期结果查询公示期（`query_period`）之前**，返回结构中的 `decision` 字段强制脱敏展示为 `"待定"`，`result` 部门与角色脱敏为 `0`，`operator_user_id` 脱敏为 `0`，`decision_remark` 脱敏为 `""`；
  - 只有到达周期设定的 `query_start_at` 结果公示时间后，申请人本人才可查看最终的录取结果与调剂说明。

---

### 3. 申请表志愿填报会员限制 (`POST /v2/application/create` & `POST /v2/application/update`)

* **限制规则说明**：
  - **已是协会会员者不可申请成为会员**：如果当前提交或修改申请表的用户在系统中的角色已是“会员”（`role.Name == "会员"`），其第一志愿（`first_choice`）或第二志愿（`second_choice`）中**严禁**选择“会员”职位；
  - 若已是会员的用户尝试将志愿职位设置为“会员”，接口将直接拒绝并返回 HTTP 400（错误信息：`"当前已是协会会员，不可申请成为会员"`）；
  - 会员用户可以正常申请竞选“干事”、“副部长”、“部长”等其他职务。
* **卸任豁免通道**：
  - 现任各级学生骨干（如“干事”、“副部长”、“部长”、“副会长”等非会员角色人员）在换届填报时，**允许**将志愿职位选择为“会员”，以此作为正规卸任流转通道，任期届满退回普通会员。

---

### 4. 照片路径校验规范（移除存储探活开销）

* **安全规范**：
  - 申请表照片路径（`avatar`）必须以 `application/` 开头，严禁非法前缀或路径穿越（禁止包含 `..`）；
  - 文件扩展名严格限制为 `.png`、`.jpg`、`.jpeg`、`.webp`、`.avif`、`.gif`；
  - 移除了提交流程中对远程 MinIO 对象的阻塞式探活查询，由前端在完成直传后将存储 URI 提交至后端，大幅提升提交与修改响应速度。

---

### 5. 文件上传 0 字节空文件防御拦截 (`POST /v2/upload/image` & `POST /v2/upload/file`)

* **防御背景与机制**：
  - 针对移动端原生相册/文件选择器在未就绪或表单被意外截断时上传 0 字节损坏文件的问题，后端增加前置大小防御校验；
  - 若 `file == nil` 或文件大小 `file.Size <= 0`，接口立即返回 `400 Bad Request`，错误提示为 `"上传文件不能为空"`；
  - 彻底杜绝 0 字节空对象写入 MinIO / S3 导致后续图片无法加载或渲染裂开。

---

## 五、 全链路日志规范与问题排查

服务端日志已全面接入结构化（JSON `slog`）与链路追踪体系，各端联调排查问题时可依赖以下机制：

1. **自动关联用户信息（`user_id`）**：
   - 只要请求通过 JWT 鉴权成功，所有 HTTP 访问日志及业务内部错误日志中均会自动附带 **`"user_id": <当前用户ID>`** 与 **`"trace_id": "<UUID>"`**。
2. **规范化 HTTP 状态码与日志级别**：
   - **`200 OK` 正常请求**：日志级别为 `INFO`，`msg` 为 `"HTTP Request"`。正常请求不含错误日志；
   - **`4xx Client Error` 业务/参数异常**：日志级别自动提升为 **`WARN`**，`msg` 为 `"HTTP Client Error"`，并带有 `"error": "<具体业务报错文案>"`（如参数不合法、权限不足、不在时间范围内等）；
   - **`5xx Server Error` 系统异常**：日志级别提升为 **`ERROR`**，`msg` 为 `"HTTP Server Error"`，并记录完整系统堆栈。

---

## 六、 前端联调开发示例代码

### 1. 导出字段 01 状态位序列生成（JavaScript / TypeScript）

系统采用直接的 **01 状态位序列字符串** 传参：

```javascript
// 从 /v2/interviewer/result/fields 获取到 22 个字段列表后：
// 比如前端使用复选框绑定了各字段的 checked 状态：
const fieldsBitSeq = allFieldsList.map(field => field.checked ? '1' : '0').join('');

// 示例：仅勾选了学号(index:2)、姓名(index:3)、面试决策(index:15)：
// 拼出: "0011000000000001000000"

// 全选导出：直接传空或传 data.all_bits ("1111111111111111111111")
```

### 2. 前端 Axios 文件流下载与文件名自动提取

```javascript
import axios from 'axios';

async function exportInterviewResults(termId, departmentId, fieldsParam) {
  try {
    const response = await axios.get('/v2/interviewer/result/export', {
      params: {
        term_id: termId,
        department_id: departmentId,
        fields: fieldsParam // 直接传 01 状态位序列，如 "0011000000000001000000"
      },
      responseType: 'blob' // 必须指定为 blob
    });

    // 从 Content-Disposition 响应头解析文件名 (已支持 UTF-8 RFC 5987)
    let fileName = '面试结果.xlsx';
    const disposition = response.headers['content-disposition'];
    if (disposition) {
      const utf8Match = disposition.match(/filename\*=UTF-8''([^;]+)/i);
      if (utf8Match && utf8Match[1]) {
        fileName = decodeURIComponent(utf8Match[1]);
      } else {
        const defaultMatch = disposition.match(/filename="?([^";]+)"?/i);
        if (defaultMatch && defaultMatch[1]) {
          fileName = decodeURIComponent(defaultMatch[1]);
        }
      }
    }

    // 触发浏览器原生下载
    const blob = new Blob([response.data], {
      type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet'
    });
    const url = window.URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = fileName;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    window.URL.revokeObjectURL(url);
  } catch (error) {
    console.error('导出失败:', error);
  }
}
```

---

## 七、 前端联调自测清单

- [ ] **面试结果导出**：
  - [ ] 使用副会长账号调用 `GET /v2/interviewer/result/export`，能否成功触发 `.xlsx` 下载？
  - [ ] 下载的文件名是否与周期标题一致（例如 `2026年秋季招新_面试结果.xlsx`）？
  - [ ] 打开 Excel 文件，表头与数据行样式是否正常（深蓝表头、对齐方式、录取结果与审核人列完整）？
  - [ ] 使用部长或普通会员账号调用导出，是否被安全拦截并返回 `403 Forbidden`？
- [ ] **010101 状态位自选字段**：
  - [ ] 调用 `GET /v2/interviewer/result/fields`，能否正确返回 22 个字段元数据定义及 `all_bits: "1111111111111111111111"`？
  - [ ] 仅勾选“学号、姓名、面试决策”导出（传 `fields="0011000000000001000000"`），下载的 Excel 是否严格仅包含该 3 列？若传非 0/1 字符串（如 `fields="32780"`），是否正确拦截报错？
- [ ] **申请表列表筛选（防陷阱、姓名筛选与面试官部门拦截）**：
  - [ ] 在查询申请表列表（`GET /v2/application/list`）时，`has_result` 不传或传空（`has_result=`），是否返回周期内的**全量申请表**？
  - [ ] 传 `has_result=true` 时，是否仅返回已录入面试结果的申请表？
  - [ ] 传 `has_result=false` 时，是否仅返回待定申请表？
  - [ ] 传 `name=王` 时，是否仅返回姓名以“王”开头的申请人？
  - [ ] 使用普通部门面试官账号传入非所属部门的 `department_id`，是否被拦截并返回 `403 Forbidden` 且提示 `"无权限查看其他部门申请"`？
  - [ ] 会长、副会长或开发者账号切换不同 `department_id` 时，是否可自由切换查看？
- [ ] **0 字节损坏文件上传拦截**：
  - [ ] 上传 0 字节文件至 `/v2/upload/image` 或 `/v2/upload/file` 时，接口是否拦截并返回 `400 "上传文件不能为空"`？
- [ ] **面试结果删除联动**：
  - [ ] 调用 `POST /v2/interviewer/result/delete` 删除某申请人的结果，再次查询申请表列表，该记录状态是否已安全回退为 `"待定"` 且审核人清空？
- [ ] **申请表隐私脱敏**：
  - [ ] 候选人调用 `GET /v2/application/me` 查看本人申请表，在周期公示期开始前，决策是否展示为 `"待定"`？
- [ ] **申请表志愿填报会员限制**：
  - [ ] 当前角色为普通“会员”的账号在创建或修改申请表时，选择志愿为“会员”，接口是否拦截并提示 `400 "当前已是协会会员，不可申请成为会员"`？
  - [ ] 普通“会员”账号填报“干事”等其他职务时是否能正常提交成功？
  - [ ] 现任“干事”或“部长”等骨干账号填报志愿为“会员”时是否允许提交（卸任通道）？
