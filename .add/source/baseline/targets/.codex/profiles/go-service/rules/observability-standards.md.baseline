# 可观测性规范（日志 / 监控 / 错误码 / 国际化）

## 1. 通用准则（核心规则，跨项目通用）

### 1.1 日志规范

**底层选型**：使用项目已选结构化日志方案；日志库、格式、SDK 入口和语言策略取项目单一依据。Go `log/slog` 与 JSON Lines 是可选示例，不表示已经采用。

**统一访问**：通过项目统一 Logger 入口（可能是 SDK）访问，**禁止业务代码直接 `fmt.Println` / `log.Println`**——否则日志格式不一致、缺关键字段。没有独立 SDK 不要求为了本规则另造一个。

**必填字段**：

| 字段 | 说明 |
| --- | --- |
| `ts` | ISO8601 时间戳（UTC） |
| `level` | DEBUG / INFO / WARN / ERROR |
| `service` | 服务名 |
| `module` | 模块名（服务内子模块） |
| `trace_id` | 链路追踪 ID（由 SDK 自动注入，遵守 tracing-standards） |
| `msg` | 日志消息（建议英文，便于跨语言团队协作） |

**例外**：仅以下场景允许直接 `fmt.Printf` / stderr：

- CLI 版本信息（`--version` 输出）
- SDK 初始化前的致命错误（SDK 未就绪时）
- 框架自动生成（如 ORM 默认日志，无修改权限）

**日志级别语义**（业界编码标准 OpenTelemetry / K8s / Azure。一句话判定：**这条记录是否需要运维介入？**）：

| 级别 | 用在 |
| --- | --- |
| ERROR | 服务端自身非预期故障、操作失败且兜不住、需运维介入。**必带错误码** |
| WARN | 出问题但能继续（降级 / 兜底 / 预期内失败）；瞬时失败重试中途 |
| INFO | 低频正常里程碑（启动完成、关键状态变更）。**不打每请求 / 每循环** |
| DEBUG | 开发排错细节，生产默认关 |
| 不打 | 高频正常路径（每请求 / 缓存命中）交给 metric + trace；**敏感信息任何级别都不打** |

关键判定（🔒）：

- **客户端错误 ≠ ERROR**：用户输入导致的 4xx（密码错 / 参数非法 / 404 / 权限不足 / 限流）只打 WARN 或不打。ERROR 只留给**服务端自身故障**——系统拒绝坏请求是正常工作，不是故障。
- **可重试失败**：单次瞬时失败、重试最终成功 → DEBUG / 不打（避免假告警）；**重试耗尽最终失败 → ERROR**。
- **同一错误只打一次**：在"判定不可处理"的那层打一次 ERROR；下层一律 `fmt.Errorf("...: %w", err)` 包装上抛，**不重复打**。
- **错误链不能断**：错误往上传每层必须 `%w` 包装并附上下文；**禁吞掉原始错误**（禁 `return errors.New("failed")` 丢 cause、禁只打日志不返回、禁 `_ = err`），保证最上层拿到完整 cause 链定位根因。（基础见 [`coding-standards §1.1`](coding-standards.md)）

**审计日志**（**≠ 普通日志**）：审计回答"**谁在何时对什么做了什么、结果如何**"。留存与防篡改能力以项目已选机制的真实保证为准，不能仅凭文件/接口名宣称不可篡改；需要审计的操作必须走项目审计入口，普通日志不替代审计。`sdk.Audit().Record` 只作已采用该 SDK 时的接口示例。

- **🔒 硬规则**：**页面上任何改变系统状态的操作一律审计，不得遗漏**——部署 / 卸载、增删改、授权变更、登录登出（含失败）都在内。纯读、健康检查、无责任主体的内部流转可不审计。
- **必带字段**（SDK 从 Context 自动带大部分）：谁（UserID / Roles / IP）/ 何时 / 对什么（ResourceKind / ID）/ 做了什么（Action）/ 结果（Status + 失败 ErrorCode）/ TraceID。

### 1.2 错误码规范

**格式**：数字或字符串均可；类型、位数、命名空间与范围来自项目单一依据，不在本 Rule 分配另一套。错误标识与语义、对外状态、日志级别和本地化消息保持稳定映射；新增/迁移要核真实消费者与兼容测试，不能靠改变类型消除冲突。

**定义位置分级**：通用错误（跨服务）在 common 模块；服务专属错误在各服务内部。

**全量覆盖 + 对外贴切（🔒 硬规则）**：

- **全量覆盖**：在项目定义的对外/服务错误边界提供稳定标识与上下文；内部 Go error 可保留真实 cause，由责任边界映射，不给每层强造新码。禁止以裸错误字符串或本地化展示文本作为公共错误协议。
- **对外贴切**：顶层 / 接口暴露响应、以及任何调用方或用户可能看见的错误，错误码必须**语义贴切、符合错误含义**，兼顾可读性与用户可理解性。**不能图省事一律返回"内部错误"**——用户看不懂、前端也无法按语义渲染合适提示。
- **理由**：错误码是国际化锚点（裸字符串无法按语言翻译）+ 全链路可结构化追踪（深层模糊码 → 上层贴切码，形态统一）。

**反例**：

```go
// ❌ 错误：用不稳定展示文本冒充公共错误标识（伪代码）
return errors.NewAppError(userFacingMessage, err)
```

**正例**：

```go
// ✅ 正确：项目注册的稳定常量；底层可为数字或字符串（伪代码）
return errors.NewAppError(autherrors.ErrCodeUserNotFound, err)
```

### 1.3 国际化（i18n）规范

**架构**：分布式语言文件——通用错误码翻译在 common；服务专属错误码翻译在各服务。

**消息码与错误码对应**：按项目稳定映射，不从数字前缀或字符串名称推断日志级别；显示语言不改变错误身份。

**占位符**：使用编程语言原生格式（如 Go `%s`、Python `{var}`），不用 `{0}` `{1}` 这种序号格式（可读性差）。

**禁止**：

- ❌ 在 common 中定义服务专属错误码翻译
- ❌ 消息 key 与项目已选错误标识/本地化映射冲突
- ❌ 使用 `{0}` 序号占位符（用命名 / 类型占位符）

### 1.4 监控指标规范

**指标类型**：

| 类型 | 用途 |
| --- | --- |
| Counter | 累计计数（请求总数 / 错误总数） |
| Gauge | 瞬时值（活跃连接数 / 队列深度） |
| Histogram | 分布统计（延迟分布 / 响应大小分布） |

**命名规范**：`<namespace>_<service>_<metric>_<unit>`，单位是国际单位（seconds / bytes）。

**标签规范**：使用低基数标签（service / method / status / error_code 等）；**禁止高基数标签**（如 user_id / trace_id——会导致指标爆炸）。

## 2. 项目依据与可选示例

项目采用值只在已有架构/ADR 或已登记 manifest 中维护；这里的结构、路径和命令仅说明填写维度及适用范例，不构成另一张配置表。选定方案不同就引用实际入口，未选不得从范例推定。

### 2.1 SDK 与底层库

从项目单一依据读取日志库/字段/格式、统一访问与审计入口。若确实采用对应 SDK，调用示例为 `sdk.Global().Log().WithModule("auth").Info("...", "user_id", userID)`；没有该接口不得照抄。

### 2.2 日志格式范例

```json
{"ts":"2026-01-01T10:15:30.123Z","level":"INFO","service":"<service>","module":"auth","trace_id":"01ARZ3NDEKTSV4RRFFQ69G5FAV","msg":"user login successful","user_id":"u-001","duration_ms":45}
```

### 2.3 服务名清单（service 字段值）

> 🚧 **待对齐**：本段为占位骨架，待本项目模块/服务划分（`<待填充:service-list>`）拍板后填充（对齐项见 [项目入口](../../../../AGENTS.md)）。

`<待填充:service-list>`（按微服务清单逐一列出，如 `<service-a>` / `<service-b>`）

### 2.4 错误码服务前缀分配

只引用项目既有错误标识注册表/架构约定的位置与版本；由该单一依据分配服务范围/命名空间，Rule 不再维护前缀表。未选则说明未知，不给客户预填数字段。

### 2.5 错误码定义位置

| 类型 | 位置 |
|-----------|---------|
| 通用错误（`<待填充:common-error-range>`） | `<待填充:common-repo>/pkg/errors/codes.go` |
| 公共基础设施错误（`<待填充:infrastructure-error-range>`） | `<待填充:common-repo>/pkg/errors/codes.go` |
| 服务专属错误 | `<service>/internal/errors/codes.go` |

### 2.6 错误码注册范例

```text
# 伪代码；按项目语言和错误码方案替换
ErrCodeResourceNotFound = <待填充:service-a-resource-not-found-code>
ErrCodeInvalidRequest = <待填充:service-a-invalid-request-code>

register(ErrCodeResourceNotFound, <not-found-status>, <severity>, "Resource not found")
register(ErrCodeInvalidRequest, <bad-request-status>, <severity>, "Invalid request")
```

### 2.7 i18n locales 位置

- 通用：`<待填充:common-repo>/pkg/i18n/locales/{zh,en}.yaml`
- 服务专属：`<service>/internal/i18n/locales/{zh,en}.yaml`

服务初始化时合并到全局：

```go
//go:embed locales/*.yaml
var locales embed.FS

func Init() error {
    return commoni18n.Global().MergeMessagesFromFS(locales, "locales")
}
```

### 2.8 监控指标命名空间

- 前缀：`<待填充:metric-prefix>_`
- 示例：`<待填充:metric-prefix>_<service>_login_requests_total` / `<待填充:metric-prefix>_<service>_deploy_duration_seconds`

### 2.9 检查方式

按项目已登记命令与精确影响范围检查：业务是否绕过统一 Logger、公共错误是否来自注册常量、状态/消息映射是否正确、cause 是否保留、敏感信息是否外泄。区分合法 CLI 输出、初始化失败和受控生成代码。数字/字符串类型本身不作为违规；不以搜索到大写 key 就否定项目选型。

### 2.10 跨项目复用指引

未来其他项目复用本规则：

- §1 保留追踪、错误链、稳定映射和审计纪律，选型相关条款按实际合同适用
- §2 只引用既有单一依据，不在 profile 重新填写下列值：
  - SDK 名称（替换 `<待填充:sdk-name>` / `sdk.Log()`）
  - 服务名清单
  - 错误码前缀分配（每个项目按自己微服务划分）
  - 错误码定义位置
  - i18n locales 路径
  - 监控命名空间前缀（替换 `<待填充:metric-prefix>_`）
