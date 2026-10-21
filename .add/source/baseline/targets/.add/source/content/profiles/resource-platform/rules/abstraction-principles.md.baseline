# 抽象设计原则

## 1. 通用准则（核心规则，跨项目通用）

### 1.1 核心理念：统一流程 + 差异化扩展点

资源管理（无论容器 / 模型 / 文件 / 任务等）采用**统一的执行流程** + **差异化的扩展点**支持各种资源类型。新增资源类型时，不修改核心流程，只实现对应扩展点。

### 1.2 三大扩展点（设计模式分类）

| 扩展点 | 设计模式 | 含义 |
|------|------|------|
| **执行策略**（如 Handler / Executor） | Strategy + Factory | 不同资源类型用不同执行器；Factory 按类型 ref 分发 |
| **资源获取策略**（如 FileFetcher） | Strategy | 不同资源类型用不同获取方式（远程下载 / 本地复制 / 镜像拉取） |
| **生命周期钩子**（如 ResourceLifecycle） | Template Method | Pre/Post Provision、Pre/Post Destroy 等钩子点 |

### 1.3 多形态部署核心框架隔离原则（🔒 最高纲领）

**核心框架形态无关**：当系统支持多种部署 / 交付形态时，核心框架（编排层、执行层、协议层）不写形态专属分支逻辑（如 `if form == "<form-a>"`）。具体形态来自项目 `<待填充:form-list>`，差异通过扩展点或项目配置承载。

**隔离机制（按层）**：

| 层 | 隔离方式 | 形态差异承载 |
|------|------|------|
| 协议层 | 资源 manifest 声明 `handler.ref` 不同 | 同一 kind，不同形态用不同 handler |
| 执行层 | 同一 Executor 接口，Factory 按 ref 分发 | 不同实现类对应不同形态 |
| 前端 Page | 项目登记的形态分发 hook / router 在 Page 级渲染对应组件 | 同一路由，不同形态渲染不同视图 |
| 前端组件 | 形态专属组件独立新建，按形态前缀命名 | 不混入通用组件 |
| 前端 Service | 现有接口不动，新增形态专属接口方法 | 共存而非替换 |

**增量新增原则**：

- 新形态的代码是**增量新增**，不是改造替换
- 已有形态的组件 / Service / Store **一行不动**
- **绝不允许改一个形态的代码把另一个形态改坏**

### 1.4 共性问题处理原则

识别"某问题是多形态共性问题"（如核心框架 bug / 跨形态共用组件缺陷 / 协议契约问题）时：

1. **先与用户确认**：说明"该问题是 X 个形态的共性问题"，由用户拍板是否按共性问题处理
2. **确认后所有受影响形态必须全部修复**：不允许只修一个形态而让其他形态带病
3. **每个形态都做端到端测试**：验证修复既解决问题、也没破坏任一形态既有功能
4. **禁"只在当前复现的形态修一下"**：必须在同一次修复中把其他受影响形态一起修掉

判断口诀：**如果去掉任一形态，bug 仍存在 → 共性问题；只有特定形态触发 → 形态专属**。

### 1.5 SOLID + 工程纪律

| 原则 | 含义 |
|------|------|
| 接口优先 | 先定义接口，再实现具体类 |
| 依赖注入 | 通过构造函数注入依赖，便于测试和替换 |
| 单一职责 | 每个类 / 函数只做一件事 |
| 开闭原则 | 对扩展开放，对修改关闭 |
| 显式优于隐式 | 配置和行为要明确，避免魔法值 |

### 1.6 常用设计模式参照

| 模式 | 使用场景 | 典型应用 |
|------|------|------|
| Strategy | 同一接口，多种实现 | 资源获取策略 / 执行策略 |
| Factory | 按类型创建对象 | Handler 工厂 / Scanner 工厂 |
| Template Method | 固定流程，可变步骤 | 生命周期钩子 / 编排骨架 |
| Observer | 事件通知 | 任务进度推送 / 状态变更广播 |
| Decorator | 动态添加功能 | Trait（特性叠加） |
| Chain of Responsibility | 依次处理 | 依赖解析 / 错误处理 |
| Registry | 注册 / 发现机制 | 类型注册表 / Handler 注册表 |

### 1.7 新增资源类型的扩展流程

通用三步：

1. **定义资源 manifest**（声明 kind / 资源期望状态 / 关联文件 / 依赖 / handler ref）
2. **按需实现扩展点**：
   - 需要特殊文件获取方式 → 实现 FileFetcher
   - 需要新执行方式 → 实现新 Handler
   - 需要特殊生命周期 → 实现 ResourceLifecycle
3. **注册到核心框架**：通过 Registry 注册 / Factory 自动发现

### 1.8 新增资源类型自检

- [ ] 是否复用了现有的统一流程？
- [ ] 差异化的部分是否通过扩展点实现？
- [ ] 是否定义了清晰的接口？
- [ ] 是否使用了合适的设计模式？
- [ ] 是否有足够的单元测试覆盖？
- [ ] 是否更新了相关文档？

### 1.9 内建能力与动态能力统一注册

同一种贡献点无论来自内建模块、项目模块还是运行时动态模块，都必须经过同一个 Registry / Factory 契约注册和发现。禁止“内建走硬编码 switch、插件走注册表”形成两套生命周期；否则权限过滤、审计、测试和卸载语义会长期分叉。

统一注册至少保证：

- 相同的身份、版本、能力声明与冲突策略；
- 相同的启用、禁用、发现与校验路径；
- 相同的权限、审计、观测和测试入口；
- 静态编译优化只改变装配时机，不改变领域契约。

### 1.10 扩展点设计三纪律 🔒

1. **按设计列全，不照代码列**：盘点系统有哪些扩展点时，从**设计的可变性维度**列全（执行 / 获取 / 存储 / 生命周期钩子 / 权限·审计·监控等系统能力 / 前端贡献 / 告警通道 / 监控对接源 …），**不要只照"代码里有 Registry 那几个"列**——后者会漏掉用 Strategy / Template Method / 声明式契约 实现的扩展点。
2. **抽象概念 ≠ 单一实现**：一个抽象机制（如"就绪探测"）不要收窄成它当前唯一的实现（如 http-probe）；保留它作为**可扩展族**（留 tcp / exec 等扩展位），概念层和实现层分清。
3. **可插拔贡献用声明式契约 + lazy**：业务模块向框架贡献能力（菜单 / 路由 / 操作 / 权限 / 形态适用性）时，用**声明式 manifest**（host 不加载模块代码就能读元数据、先建骨架 + 权限过滤）+ **lazy 实现**（命中才加载）。对标 VS Code contribution points / Backstage Blueprint。

## 2. 本项目实现（按选型可替换）

> 🚧 **待对齐**：§2 整节为可选 `resource-platform` profile 的项目骨架。只有项目确实管理声明式资源并选择了资源协议时才填；普通业务项目不启用本节。资源描述协议（`<待填充:resource-protocol>`）与编排机制由架构设计拍板（对齐项见 [项目入口](../../../entry/project-guide.md)）。

### 2.1 资源协议核心结构（`<待填充:resource-protocol>`）

🚧 待填充内容：协议定位 ADR 引用 + 资源逻辑结构定义。可按项目需要从以下通用维度取舍：

```text
Resource = Metadata（身份） + Spec（期望状态） + Files（关联文件）
         + Dependencies（依赖声明 = 制品依赖 + 运行/环境需求）
         + Workloads（执行单元） + Outputs（就绪后对外暴露）
```

需拍板：序列化承载（是否 OCI 原生对象）/ 版本与寻址（tag / digest）/ 形态适用性的推导方式。

### 2.2 统一编排流程

🚧 待填充内容：编排服务从「解析 manifest → 构建依赖 DAG → 拓扑排序 → 节点调度 → 逐资源执行（Fetch 文件 → 逐 workload 执行 → 就绪检查 → 收集 outputs）→ 聚合返回」的项目化定义。

### 2.3 扩展点 1：Handler（执行策略）

🚧 待填充内容：本项目 Handler 清单（维度：Handler 名 / 职责 / 适用形态）。接口骨架如下：

```go
// HandlerExecutor 接口 —— 所有 Handler 实现统一接口
type HandlerExecutor interface {
    Execute(ctx context.Context, req *ExecutionRequest) (*ExecutionResponse, error)
    Abort(ctx context.Context, executionID string) error
}

// HandlerFactory 根据 handler.ref 创建对应执行器
type HandlerFactory interface {
    Create(handlerRef string) (HandlerExecutor, error)
}
```

### 2.4 扩展点 2：FileFetcher（文件获取策略）

🚧 待填充内容：本项目「资源类型 → 文件获取方式」映射表。接口骨架沿用：

```go
type FileFetcher interface {
    Fetch(ctx context.Context, resource *Resource) (map[string]string, error)
}
```

### 2.5 扩展点 3：ResourceLifecycle（生命周期钩子）

```go
type ResourceLifecycle interface {
    PreProvision(ctx context.Context, resource *Resource) error
    PostProvision(ctx context.Context, resource *Resource, state *RuntimeState) error
    PreDestroy(ctx context.Context, resource *Resource) error
    PostDestroy(ctx context.Context, resource *Resource) error
}
```

### 2.6 形态隔离的项目落地

🚧 待填充内容：形态清单（`<待填充:form-list>`）拍板后，给出「形态 → 形态值 → 主要 Handler」映射 + 前端形态分发 hook 约定。

### 2.7 新增资源类型示范

🚧 待填充内容：按 `<待填充:resource-protocol>` 给出最小 manifest 示例（`apiVersion: <待填充:api-group>/v1`）+ 注册到编排框架的代码示例（RegisterFetcher / Handler 自动发现 / RegisterLifecycle）。

### 2.8-2.9 完整示例

🚧 待填充内容：协议定稿后补 1-2 个来自本项目领域的端到端真实资源示例。

### 2.10 跨项目复用指引

未来其他项目复用本规则：

- §1 通用准则直接复制（统一流程+扩展点 / 多形态隔离 / 共性问题处理 / SOLID / 设计模式都是通用工程纪律）
- §2 本项目实现按新项目选型替换：
  - 资源 manifest 协议（使用项目已拍板协议）
  - 编排流程（按新项目协议）
  - Handler 清单（按新项目类型）
  - 形态值清单（如新项目只有单形态则简化）
  - YAML 示例（按新项目领域）
