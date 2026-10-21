# 编码规范

## 1. 通用准则（核心规则，跨项目通用）

### 1.1 Go 语言基础

- **格式化**：`gofmt` + `goimports` 强制；`golangci-lint` 检查
- **命名**：包名小写短词；变量驼峰，缩写大写（`userID` 而非 `userId`）
- **注释**：所有导出的包 / 函数 / 类型必须有注释；项目内允许中文注释
- **错误处理**：用 `%w` 保留错误链；立即检查 err；自定义错误类型含错误码
- **接口**：小而精，单一职责；单方法接口用 `-er` 后缀（`Reader` / `Closer`）
- **并发**：用 `WaitGroup` 等待 goroutine；用 `context` 控制超时；用 `defer` 释放资源

### 1.2 函数复杂度（🔒 强制）

- 函数最大 **50 行**；最大嵌套 **5 层**
- 超出必须拆分为多个小函数
- 用**早返回**减少嵌套

```go
// ✅ 正确：早返回 + 函数拆分
func processData(data []Item) error {
    if data == nil {
        return nil
    }
    for _, item := range data {
        if err := processItem(item); err != nil {
            return err
        }
    }
    return nil
}

func processItem(item Item) error {
    if !item.Valid || item.Type != "A" {
        return nil
    }
    // 处理逻辑...
    return nil
}
```

### 1.3 测试规范

- **文件命名**：单元 `_test.go` / 集成 `_integration_test.go` / 属性 `_property_test.go`
- **函数命名**：单元 `Test<Function>_<Scenario>_<Expected>` / 属性 `TestProperty<N>_<PropertyName>`
- **结构**：Arrange-Act-Assert 模式；表驱动测试；每个测试独立数据库 schema

### 1.4 gRPC 规范

- **Proto**：proto3 语法；包名按 `<project>.<service>.v1` 三段式；服务和消息必须有注释
- **实现**：SDK 自动记录审计日志；用 gRPC status 返回错误；错误码映射到 gRPC codes

### 1.5 API 默认值与向前兼容

- 对外公开 API / 跨团队契约新增字段时，旧调用方不传该字段也必须得到明确、稳定的默认行为；默认值属于契约，需在规约和兼容性测试中固定。
- 服务内部且调用方同步升级的接口，可以采用更严格的必填策略，但必须明确标记为内部契约，不能把内部假设泄露成公共 API 行为。
- 新枚举值、未知字段和能力协商优先采用“可忽略 / 可降级 / 明确拒绝”的确定语义，禁止因客户端版本差异静默执行另一条业务路径。
- 兼容策略必须同时覆盖序列化、服务实现和生成客户端；只改其中一层不算完成。

### 1.6 Git 提交格式

```text
<type>(<scope>): <subject>
```

- **Type**：feat / fix / docs / style / refactor / test / chore
- **Scope**：服务名 / 模块名（按项目划分）
- **Subject**：中文描述，≤50 字符，不以句号结尾

### 1.7 抽象设计原则

通用抽象纪律见 core [`abstraction-principles.md`](../../../guidance/abstraction-principles.md)。FileFetcher、Handler、ResourceLifecycle 的完整方法只由 不适用（未选择此方法） 拥有，项目明确启用该 profile 且任务涉及资源模型才读取；选择 Go 本身不引入资源协议。

### 1.8 DDD 服务层设计（明确选择时适用）

仅在项目明确选择 DDD、CQRS、富模型或事件驱动且当前服务涉及该选择时应用对应部分；各项可分别采用，不互相隐含。依据引用项目已有架构/ADR，不在此重复登记。简单服务、脚本或 CRUD 不因使用 Go 必须升级架构；仍须明确状态合法性、责任、错误和可测试性。

**已选方法的实践**（下列示例展示同时采用时的结构，不作为未选项目的禁止清单）：

1. **领域模型分离**：服务返回领域模型（domain object），不返回 ORM 实体（如 `ent.Resource`）
2. **CQRS 分离**：命令服务（写）和查询服务（读）分接口
3. **状态机约束**：状态转换必须通过状态机验证；禁止任意 `SetStatus(x)`
4. **事件驱动**：状态变更时发布事件（支持 WebSocket / 异步处理 / 审计扩展）
5. **富领域模型（拒绝贫血 + 防流水账）🔒**：行为跟着它所属的数据走（tell-don't-ask）——领域对象自己封装规则、状态转换、约束；服务只做协调编排。**禁两种退化**：① **贫血模型**（只有字段的数据对象 + 把所有逻辑塞进"聪明的服务"）② **god-function**（一个方法从头跑到尾的流水账过程编码）。变化点（按类型 / 形态 / 状态分支）用**多态吸收**，不用大 switch / if-else 链。设计模式（State / Strategy / Command / Observer / Composite 等）是良好职责分解的**自然结果——为分解职责而用，不为用模式而用**。判据：函数 ≤50 行（§1.2）通常意味着职责已落到对象上；一个几百行的服务方法 = 流水账信号，回去重分解。

**接口设计反例 → 正例**：

```go
// ❌ 错误：CRUD 风格 + 返回 ORM 实体（泄露持久化细节，缺乏业务语义）
type ResourceService interface {
    Create(ctx, req) (*ent.Resource, error)
    Update(ctx, id, req) error
    UpdatePhase(ctx, id, phase) error  // 无状态机约束
}

// ✅ 正确：业务语义动词 + CQRS 分离 + 状态机
type ResourceCommandService interface {
    Submit(ctx, req) (*Resource, error)  // 业务语义
    Approve(ctx, id) error
    Activate(ctx, id) error
    Retire(ctx, id) error
}
type ResourceQueryService interface {
    Get(ctx, id) (*Resource, error)        // 返回领域模型
    List(ctx, filter) ([]*Resource, error)
}
```

**状态机用法**：

```go
// ✅ 正确：状态机验证
if !s.stateMachine.CanTransition(current.Phase, newPhase) {
    return ErrIllegalStateTransition
}

// ❌ 错误：直接设置
s.db.Resource.UpdateOneID(id).SetPhase(phase).Exec(ctx)
```

**事件发布**：

```go
// 状态变更 → 验证 → 持久化 → 发布事件
s.eventPublisher.Publish(PhaseChangedEvent{...})
```

### 1.9 禁止硬编码（🔒 强制）

> 完整规范见 [`no-hardcode-policy.md`](no-hardcode-policy.md)

禁止硬编码：

- 业务类型字符串（用常量）
- 配置值（用环境变量）
- 版本号（用构建注入 `-ldflags`）
- 文件路径 / 端口 / 文件权限
- 魔法数字（用命名常量说明含义）

### 1.10 常量定义位置分级

| 常量类型 | 定义位置 | 频度 |
|---------|---------|------|
| 协议层（极少变更） | common 模块的协议包 | Kind / APIVersion 等 |
| 通用常量（极少变更） | common 模块的 consts 包 | 文件权限 / API 版本等 |
| 业务层（可能变更） | 服务内部的 consts 包 | 端口 / 路径 / 业务类型枚举 |

**禁止**：在 common 中定义业务类常量（common 只放接口和协议）。

### 1.11 微服务目录结构

每个微服务仓库的 `internal/` 目录推荐结构：

```text
<service>/internal/
├── consts/
│   └── consts.go      # 业务常量（端口 / 路径 / 类型枚举）
├── common/
│   └── utils.go       # 公共辅助函数（验证 / 判断函数）
├── errors/
│   └── codes.go       # 服务专属错误码
├── i18n/
│   ├── i18n.go        # 国际化初始化
│   └── locales/       # 语言文件
└── <module>/          # 业务模块
```

要点：常量和辅助函数分离；常量按功能分组注释。

### 1.12 代码审查清单（提交前）

- [ ] 格式化通过（`gofmt -l .` 无输出）
- [ ] Lint 通过（`golangci-lint run` 无错误）
- [ ] 注释完整
- [ ] 单元测试覆盖
- [ ] 无调试代码（`fmt.Println` 等）
- [ ] 错误处理完整
- [ ] SDK 日志 / 审计正确使用

## 2. 本项目实现（按选型可替换）

### 2.1 Go 版本

- Go 版本和工具命令读取项目已选技术的单一依据（架构/ADR 或已登记项目 manifest）；未知不由母本填默认版本。

### 2.2 Proto 包名空间

- `<待填充:proto-ns>.<service>.v1`

### 2.3 Git Scope 清单

> 🚧 **待对齐**：随 本项目 微服务划分（`<待填充:service-list>`）拍板后填充——惯例为各服务短名 + `common` / `deploy` 等横切 scope。

### 2.4 业务类型常量示例

业务常量按实际领域命名；只有采用资源协议的项目才使用 Kind 常量示例。不为普通服务补造资源类型。

### 2.5 配置环境变量前缀

`<待填充:env-prefix>_*`（详见 [`no-hardcode-policy.md`](no-hardcode-policy.md)）

### 2.6 常量位置（具体）

| 类型 | 位置 |
|---------|---------|
| `<待填充:resource-protocol>` 协议常量（Kind / APIVersion） | `<待填充:common-repo>/pkg/<协议包>/core.go` |
| 通用常量（DirPerm / FilePerm） | `<待填充:common-repo>/pkg/consts/consts.go` |
| 业务层常量 | `<service>/internal/consts/consts.go` |

### 2.7 DDD 适用服务清单（本项目）

引用项目已批准的服务适用范围及原实施任务，不另写“已实现/待重构”状态表；未选择 DDD 的项目不需要该清单。

### 2.8 前端规范引用

前端实现由项目选定的前端 profile 负责；`go-service` 不对前端技术栈作默认要求。

### 2.9 代码审查归档

> 🚧 **待对齐**：本项目 编码规范详细档案产生后在此登记（位置惯例：`../../../../.add/project/knowledge/archive/original-design/.../编码规范.md`）。

### 2.10 跨项目复用指引

未来其他项目复用本规则：

- §1 按具体协议/已选架构适用，语言工程纪律不等于全选 DDD/资源模式
- §2 只引用项目既有单一依据；下列维度由该依据维护，不在 Rule 再填一次：
  - Go 版本（按项目升级）
  - Proto 包名空间（按项目名）
  - Git scope 清单（按项目微服务划分）
  - 业务类型常量（按项目领域）
  - 配置环境变量前缀
  - 常量位置（按项目结构）
  - DDD 适用服务清单（按项目）
