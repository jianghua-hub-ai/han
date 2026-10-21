# 前端编码规范（TS / 状态 / i18n / Vibe Coding / 微前端）

> **范围**：明确采用 React/TypeScript 时的非 UI 实现实践；稳定数据/状态、授权、安全与 mock 质量合同取 core [`frontend-coding-standards.md`](../../../rules/frontend-coding-standards.md)。微前端、Store、i18n 库按项目已选方案适用。
> UI 系统硬约束见 [`frontend-design-standards.md`](./frontend-design-standards.md)。
> 实施方法直达 `skill:task-implementation` 前端适用段；跨活动关系由当前 Workflow 决定，本 Rule 不再安排第二套开发顺序。

## 1. 通用准则（核心规则，跨项目通用）

### 1.1 TypeScript 严格

- 禁 `any` 类型；不确定时用 `unknown` + 注释（含 TODO 标记待后端确认）
- 明确接口定义所有数据结构
- 启用 strict mode；启用 `noImplicitAny` / `strictNullChecks`

```typescript
// ✅ 正确
interface UserProps {
  id: string;
  name: string;
  role: 'admin' | 'operator' | 'viewer';
}

// ❌ 错误
interface UserProps {
  data: any;
}
```

### 1.2 命名规范

| 类型 | 命名规则 | 示例 |
|---|---|---|
| 组件文件 | PascalCase | `UserProfile.tsx` |
| 工具文件 | camelCase | `apiClient.ts` |
| Hooks / Stores | camelCase + `use` 前缀 | `useAuthStore.ts` |
| 接口 / 类型 | PascalCase | `User`, `ClusterConfig` |
| 常量 | UPPER_SNAKE_CASE | `API_BASE_URL` |

### 1.3 函数组件 + Hooks

- 禁 class 组件（过时）；用函数组件 + Hooks
- Props 用 TypeScript 接口

### 1.4 状态管理实现选择

本地/共享/服务端状态的所有权边界见 core Rule。本表是可选实现示例；Store、缓存/请求库和目录取项目单一依据，不强迫已有查询缓存改成 Store，不复制服务端状态。

| 状态类型 | 方案 | 示例 |
|---|---|---|
| UI 临时状态 | `useState` | 下拉菜单开关 / 表单输入 |
| 业务数据 | 全局 Store（如 Zustand） | 用户信息 / 列表数据 |
| 跨组件通信 | 全局 Store | 全局通知 / 主题切换 |
| 服务端状态 | Store + API 缓存 | API 响应缓存 |

### 1.5 已选 Store 的领域拆分 + Action 封装

以下只在项目已选对应 Store 模式时适用；其他数据访问方式遵循其既有公开接口，不要求 API 一律进入 Store Action。

```typescript
// ✅ 正确：按领域拆分 + 业务逻辑在 Store Action 中
stores/
├── useUserStore.ts
├── useAppStore.ts
└── useModelStore.ts

// Store Action 封装业务逻辑
export const useAppStore = create<AppState>((set, get) => ({
  apps: [],
  fetchApps: async () => {
    set({ loading: true });
    const apps = await appService.getList();
    set({ apps, loading: false });
  },
}));

// 组件中只调用 Action，不直接 fetch
const apps = useAppStore((s) => s.apps);
const fetchApps = useAppStore((s) => s.fetchApps);
useEffect(() => { fetchApps(); }, [fetchApps]);
```

### 1.6 选择器优化（避免无关重渲染）

```typescript
// ✅ 正确：单字段选择器
const userName = useUserStore((state) => state.name);

// ❌ 错误：解构整个 Store——任意字段变化都重渲染
const { name, ...rest } = useUserStore();
```

### 1.7 i18n 必须

- 用户可见文本按项目语言策略处理，不因是英文就豁免；未知策略先核项目依据
- 禁字符串拼接构建翻译文本
- mock 实例可在隔离 fixture 中给出固定测试数据；API 地址/路径仍取项目契约与配置，不以 TODO 豁免生产硬编码

### 1.8 微前端边界

仅项目已采用微前端时适用；单应用不为本条引入 Remote/Module Federation。

- Remote 模块必须**自包含导出**，不依赖 `window.*` 全局对象
- 共享依赖通过 Module Federation 配置（React / 状态库 / 路由库等）
- UI 组件必须从项目登记的**统一 UI 入口** `<待填充:ui-entry>` import，不绕过

### 1.9 错误处理

- API 失败 → 明确错误状态并保留根因，不能用空数据冒充加载成功；由项目指定错误边界接收并提示
- WebSocket / SSE 断连 → 按项目策略有界、可取消重试和用户感知，组件销毁及时清理
- 表单校验失败 → inline 显示（如 `FormMessage` 红字），不弹 Modal

### 1.10 原型与真实实现

按 `skill:task-implementation` 实施，不另规定“布局→mock→接口”顺序。原型需要可复核代表数据，也要覆盖空/错误/极端输入；数据与权限仍遵守 core 隔离合同。已有批准设计的修复直接复用，不重跑页面需求/设计。

### 1.11 ESLint 守护类别

| 类别 | 类典型规则 |
|---|---|
| TypeScript | `no-explicit-any` / `no-unused-vars` / `no-non-null-assertion` |
| React | `rules-of-hooks` / `exhaustive-deps` / `jsx-key` |
| 代码质量 | `no-console`（允许 warn/error） / `no-debugger` |
| 最佳实践 | `eqeqeq` / `prefer-const` / `no-var` |

### 1.12 提交前自检清单

- [ ] TS 编译无错误（type-check）
- [ ] ESLint 0 error
- [ ] 组件使用函数式
- [ ] 异步消费项目已选入口；采用 Store Action 模式时核相应封装
- [ ] Loading / Error 状态处理完整
- [ ] 无 `any` / 无硬编码文本（应 i18n）
- [ ] 无内联 mock（sim / 假数据 / 假延时）；按项目隔离方案核实际生产产物，§1.13 网关是可选实现
- [ ] UI 部分通过设计系统验收

### 1.13 Mock 隔离（mock 严禁污染生产 🔒）

隔离要求由 core Rule 拥有。本节是项目选择构建期开关/服务适配时的 React/TS 实现示例，`isMock`、`__mocks__` 和路径可依项目方案替换；是否真正从生产构建移除必须检查产物，不能仅凭动态 import 推定。假数据与假逻辑都纳入隔离，真实 UI/工具逻辑不当 mock 误删。

铁律：

1. **统一 `isMock` 网关**：mock 与真实实现的分叉**只在 service 层**，由构建期开关 `isMock` 判定（`isMock ? mock : 真 API`）。组件 / 页面 / store 只调 service 接口，不感知 mock。
2. **mock 放 `__mocks__/` + 动态 import**：mock 数据和 mock 逻辑全部放 `__mocks__/`；service 用 `await import('../__mocks__/x')` **动态加载**（只在 `isMock` 分支），保证生产构建 tree-shake 掉。**禁顶层 static import mock**。
3. **生产组件零 mock**：组件 / 页面 / store 里**禁内联 sim**（`setInterval` 假进度 / `setTimeout` 假延时 / `Math.random` 假数据）、禁硬编码假业务数据、禁直接 `import '__mocks__'`。
4. **禁 catch-返回-mock**：真实 API 调用失败时**返回错误**（让上层兜底 / 提示），**绝不在 `catch` 里返回假数据**——否则生产环境 API 故障会给用户看假内容。
5. **真实实现未就绪 = 留 stub，不是用 mock 顶替**：设计 / 原型阶段真实分支可留 `throw new Error('TODO: 接 XXX')` 占位，由 dev 接；不允许"先用 mock 当生产逻辑跑着"。
6. **类型不放 `__mocks__/`**：领域类型定义在 `types/`，`__mocks__/` 只放 mock 实例数据 + mock 函数。

合法例外（不算 mock 污染，不要误清）：客户端唯一 ID 生成（`Date.now() + Math.random`）、UI 动画延时（`setTimeout(onClose, 200)`）、真实交互功能里的随机（如"换一批"洗牌）——它们是真实产品逻辑，不是"假装后端"。

**特例：超前于后端的权限码 / 菜单声明也按 mock 隔离**（设计先行流程，见 [`task-execution-workflow 场景 F`](../../../rules/task-execution-workflow.md)）。前端先画页面/原型时，**超前声明后端尚未注册的权限码 / 菜单是合法的**，但必须满足两条铁律，否则就是污染：

1. **超前码进 `__mocks__` + `isMock` 网关**——超前的权限码字典 / 用户权限集 / 菜单声明放 `__mocks__/`，生产态从后端注册中心拿（`isMock ? mock : 真 API`）。**禁写死在前端生产代码**（组件内 `PERMISSIONS` 常量 / manifest 字面量硬编码权限码是反例——它们绕过后端权威、必然漂移）。
2. **作为"待开发"声明驱动后端注册 + R 对账下线**——mock 里的超前码是"前端认为需要、待后端注册实现"的输入需求，不是前端自造的孤儿码；后端注册落地后由 R 对账机制把超前码下线（改消费真 API）。漂移检测：设计声明 ∩ mock ∩ 后端注册结果差集 = drift（对账见场景 F）。

### 1.14 Mock / demo 数据要真实可核对（🔒）

mock / 原型 / 演示数据若指向**真实世界实体**（真实产品 / 版本 / 规格 / 价格），数值必须从**权威来源核对**取真实值（如官方站、官方 API 或权威仓库），**不要凭记忆或公式推算编造**。演示内容必须能由观众从权威来源复核。

- **知识截止后发布的真实实体不是虚构的**：遇到不认识的产品或版本，先查权威来源，不得因模型知识中不存在就当作虚构。
- **取数过程可复核**：记录来源 URL、查询时间、API/页面字段及必要的计算口径；涉及聚合时保留命令或公式，禁止用未经验证的经验公式替代真实数据。

### 1.15 已选微前端的可观测 / 日志（host-owned shared singleton）

微前端跨 remote 的日志 / 错误 / trace / RUM **不各 remote 各起一套**——全应用**一个可观测实例，host 初始化一次，以 Module Federation `shared singleton` 分发给所有 remote**（对标 Sentry / Datadog / Grafana Faro 微前端：host 持单实例，remote 复用）。否则各 remote 各起一个，trace 串不起来、错误各报各的、字段不一致。

**三条铁律**：

1. **import 零副作用**——可观测模块在**模块顶层不 `new`**（不 eager 构造 client / 不在构造期装定时器 / unload 钩子）；改懒初始化（首次用才构造，副作用挪到 host 调一次的 `init`）。否则被 `singleton` 共享的那个 chunk 会被早期求值，**把整个可观测模块图拖进共享链、破坏共享依赖（React 等）解析**。
2. **独立共享模块分发，不挂 `window.*`、不经 UI 入口**——可观测当**独立** `shared singleton` 模块（与状态库 / UI 库解耦）。**禁挂 `window.*` 全局**（违反 §1.8 微前端边界）；**禁塞进被狂共享的 UI 单一入口**（UI 入口被所有 remote import，塞可观测 = 拖整个模块图进 UI chunk，实测会让 remote `Cannot read null`）。
3. **结构化字段 + 禁 `console.*`**——日志走统一 logger，字段对齐 observability-standards（`ts/level/service|mfe/module|route/trace_id/msg`）；禁散落 `console.*`，ESLint `no-console` 守护。

remote 侧**透明复用**（零配置但非零感知）：import 同一个 logger 即与 host 同实例、同 trace context、同上报缓冲——跨 remote 一次操作串成一条 trace。

## 2. 项目依据与可选实现示例

### 2.1 技术栈

框架/语言版本、构建、模块化、UI、状态、路由、HTTP、测试和命令只引用项目已有单一依据（架构/ADR 或已登记 manifest），不在 Rule 填写另一张技术表。以下代码/路径是明确选择相应机制时的示例，不自动成为本项目配置。

### 2.2 禁用清单

引用项目设计系统的实际废弃清单与适用消费者，ESLint 按清单守护。兼容阶段遵循已批准迁移范围，canonical 不预置具体包名，也不声称项目已完成迁移。
- ❌ 项目已登记的 legacy 版本化命名（如 `V1xxx`）
- ❌ 绕过 `<待填充:ui-entry>` 直接引用项目列入 restricted imports 的底层组件包
- ❌ 项目架构决策明确禁用的状态库、构建工具或 UI 包

### 2.3 项目特定 ESLint 规则（历史 API 守护）

```javascript
'no-restricted-imports': ['error', {
  paths: [
    { name: '<forbidden-ui-package>', message: '使用项目统一 UI 入口 <待填充:ui-entry>' },
    { name: '<legacy-ui-entry>', message: '历史 UI 入口已禁用' },
  ],
  patterns: [
    { group: ['<legacy-ui-package>/*'], message: '历史 UI 子路径已禁用' },
    { group: ['*/components/<legacy-version>', '*/components/<legacy-version>/*'], message: '历史版本目录已禁用' },
  ],
}],

'id-match': ['error', '^(?!V1[A-Z]).*$', {
  properties: false,
  classFields: false,
  onlyDeclarations: true,
}],
```

### 2.4 i18n 语言包位置

`<待填充:i18n-locales-path>`，按模块组织：

```typescript
// locales/zh.ts
export default {
  common: { search: '搜索', filter: '筛选', total: '共 {{count}} 项' },
  resourceCenter: { title: '资源中心', open: '打开' },
  // ...
};
```

使用：`const { t } = useTranslation(); <h1>{t.resourceCenter.title}</h1>`

### 2.5 业务侧 UI 唯一入口

```typescript
// ✅ 正确
import { Button, PageHeader, DataTable, message } from '<待填充:ui-entry>';

// ❌ 禁用
import { Button } from '<legacy-ui-package>';
import * as Dialog from '<forbidden-ui-package>';  // 应走项目统一 UI 入口
```

### 2.6 Module Federation 共享依赖

```typescript
federation({
  shared: ['react', 'react-dom', 'react-router-dom', 'zustand'],
})
```

本例仅展示配置位置，shared 内容由实际消费者及版本兼容要求决定，不因示例遗漏某库就视其废弃。

### 2.7 检查命令

从项目单一依据取实际命令；下面仅是已采用 pnpm workspace 且确实存在对应脚本时的示例，不自动安装依赖或运行全仓检查。

```bash
cd <待填充:web-repo>
pnpm run type-check    # TS 编译
pnpm run lint          # ESLint 0 error
pnpm run lint:fix      # 自动修复 fixable
pnpm run -r build      # 全部 workspace package 构建
```

### 2.8 Mock 隔离落地（isMock 网关）

| 要素 | 位置 / 值 |
| --- | --- |
| 构建开关 | 各 package `src/config/env.ts` → `isMock`（读 `import.meta.env.VITE_MOCK`） |
| mock 位置 | 各 package `src/__mocks__/`（mock 数据 + mock 函数） |
| 网关位置 | 各 package `src/services/*Service.ts`（`isMock ? await import('../__mocks__/x') : fetch(真 API)`） |
| 启动 mock | `VITE_MOCK=true` / `pnpm dev:mock` |
| 类型位置 | 各 package `src/types/`（不放 `__mocks__/`） |

参考实现路径由项目入口登记：`<待填充:auth-service-path>`（登录 isMock 分支）。**评审强制**：mock 隔离按 core [`code-review-checklist`](../../../rules/code-review-checklist.md) 审查——存量违规在对应文件被评审时拦截，不堆 improvements backlog。

### 2.9 跨项目复用指引

未来其他项目复用本规则：

- §1 只采用与当前技术选择相符的实践，通用质量约束引用 core，不复制第二流程
- §2 只引用项目单一依据，不在 profile 再维护以下技术表：
  - 状态管理库（如换成 Jotai / Redux Toolkit）
  - UI 组件库（如换成 Chakra / Mantine）
  - 构建工具（如换成 Next.js / Turbopack）
  - 微前端方案（如换成 single-spa）
  - 项目特定 ESLint 规则（历史 API 守护按本项目迁移清单裁剪）
  - i18n 语言包位置
