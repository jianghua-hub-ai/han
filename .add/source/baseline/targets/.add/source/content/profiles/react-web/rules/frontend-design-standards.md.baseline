# 前端设计规范（UI 系统硬约束 SSOT）

> **定位**：React Web 的设计系统实现实践；通用一致性、状态和无障碍合同由 core [`frontend-design-standards.md`](../../../rules/frontend-design-standards.md) 拥有。项目设计系统/组件/token/迁移和命令只取项目已有单一依据（架构/ADR 或已登记 manifest），本 Rule 不维护竞争配置表。
> 第 2 节为可选示例，不是项目现状、强制 UI 栈或已完成迁移的声明。仅取实际需要且已确认适用的机制，不因启用 React 就照抄全部组件/像素/布局。

## 1. 通用准则（核心规则，跨项目通用）

### 1.1 设计原则（铁律，按优先级）

| # | 原则 | 含义 |
|---|---|---|
| 1 | **一致且可理解** | 名称/API 按项目已选设计系统；采用 shadcn/Tailwind/Radix 才按其实际合同，不强制换库 |
| 2 | **优先复用** | 优先使用已有稳定 primitive；是否增加 wrapper 由真实消费者、共同变化与成本决定，不机械禁止 |
| 3 | **明确兼容边界** | 适配层和双 API 仅服务真实迁移需求，明确范围、责任、兼容测试与退出条件，不无限保留，也不强制一次切换 |
| 4 | **Token 语义统一** | 颜色/排版/间距等取项目 token；CSS variables 是一种实现，不在业务页复制常量 |
| 5 | **不改原视觉交互** | 实现升级时仅替换实现，不动用户已接受的视觉 / 交互 |
| 6 | **可验证演进** | 按已批准迁移顺序和真实消费者回归，确认旧入口不再被使用后才退役；不借换实现改变已接受效果 |

### 1.2 Token 命名

**单一前缀 + 语义命名**——所有 CSS 变量用同一前缀（如 `--<project>-*`），按语义分层（颜色 / 字号 / 间距 / 圆角 / 阴影），不按具体值命名（不写 `--blue-500`）。

### 1.3 字号 scale 原则

同一语义角色（标题/正文/标签等）采用项目设计系统的 token 与响应式尺度；同场景一致，合理的容器/屏幕变化由该系统表达。

字号下限、档位和例外取项目可访问性/视觉依据，不由 React profile 设置通用像素值。核真实缩放、多语言和关键操作的可读性。

项目明确选定尺度后守护未授权的档外值，不把另一项目的 7 档示例当既有标准。

### 1.4 组件命名

- 遵循已选组件系统的公开名字；只有采用相应库才保留如 `Dialog` / `DialogContent` / `Select` 的 API
- 设计系统专属组件用稳定语义名；迁移期间确需版本并存时，引用已批准范围、真实消费者和退出条件，不从名字自动删除组件
- 视觉装饰原子保留语义化命名

### 1.5 组件单一来源（SSOT）

**任何视觉概念在项目里有且只有一个组件实现**。业务页面禁止 inline 自抄，跨模块禁止复制——同一视觉概念若多实现，必有不一致。

典型场景：

- 页面顶部 H1 标题 → 统一 `<PageHeader>` 组件
- 业务卡片 → 统一卡片骨架
- 表格 → 统一 `<DataTable>`（行高 / 排序 / 操作列规范统一）
- Modal / Dialog / Tooltip → 统一组件

### 1.6 兼容与迁移策略

先核消费者使用的行为与 API，评估直接迁移、wrapper 或有界适配的成本及风险，再按项目已批准方案执行。已宣布退役且消费者迁完的入口才纳入 ESLint 禁用；尚在有效兼容范围的旧 API 不误报。阶段完成需真实消费者回归与已接受视觉/交互对照，不因“统一新 API”跳过兼容。

### 1.7 前端框架性防御

授权合同见 core [`frontend-coding-standards.md`](../../../rules/frontend-coding-standards.md) §5。本端将项目实际权限结果映射为可见、不可用及可解释反馈；不强制后端提供固定字段，也不保证任意业务变化都能前端零改动。

### 1.8 无障碍实现

按 core 约束及项目已选可访问性基线验证；Web 实现核语义 DOM/ARIA、键盘 Tab 顺序和焦点、图片合适的替代文本（装饰图按语义处理）、React Label 的 `htmlFor` 与输入 ID 关联。对比度、主题、缩放和减少动效取实际标准与项目矩阵，不只检查属性存在。

### 1.9 ESLint 防回滚

ESLint 守护旧 API / 旧命名（如废弃组件库 / 废弃 token 前缀 / 版本化命名）的硬性禁用：

- `no-restricted-imports`：禁旧库 import
- `id-match`：禁特定命名 pattern（如 `V1xxx` 标识符）

### 1.10 验收准入

任何 UI 改动 / Refactor 完成时必须满足：

- 本次受影响包及真实消费者的项目构建/回归通过，不默认要求所有 package
- ESLint 0 error
- 在批准迁移范围核已退役 API/命名/token，无待迁消费者被误伤；历史说明或合法兼容例外不冒充违规
- 视觉跟改动前一致（preview 浏览器人工对照浅 / 深主题）

### 1.11 自适应 / 响应式

按项目已批准布局与目标浏览器/设备验证内容可读、操作可达、缩放和窄屏不意外裁切。列数、断点、导航形式由实际设计决定；已选六列或窄屏折叠菜单均可合法，不能为遵循母本改掉已接受视觉。以下是解决具体溢出问题的可选实现，不是通用布局准入。

| 维度 | 可选实现与适用前提 | 应核验的风险 |
|---|---|---|
| **卡片/网格栅格** | 已选封顶网格时可用 `repeat(N, 1fr)` + resize 或对应 CSS 断点；自动填充布局也按其真实设计验证。采用 CSS/Tailwind 时核构建产物和目标浏览器的 `minmax()` 等能力 | 实际列数/卡片宽度偏离批准布局、无意挤压/裁切、构建缺响应式样式 |
| **一级菜单 / 横向导航** | 已选单行横滚时可用 `white-space: nowrap` + `min-width: 0` + `overflow-x: auto`；已选折叠菜单或换行布局则按其设计实现并核键盘/焦点与操作可达 | 文本意外逐字竖排、隐藏关键入口、折叠后键盘或焦点不可达 |
| **卡内标签 / chips** | `flex-wrap: wrap`（挤不下换行）或"显 N 个 + 余 +M 溢出标"；每 chip `white-space: nowrap` 自身不断字 | `flex-wrap: nowrap` + `overflow: hidden` 裁切文字 |
| **横向 padding** | 窄屏收窄，别让大 hpad 在小屏吃掉内容宽 | 固定大 hpad 不随宽度变 |
| **桌面平台极窄兜底（可选）** | 项目明确采用固定最小宽度时才设置对应 `min-width` 并核横滚/顶栏；若采用移动端响应式则验证其独立布局，不能用桌面范例替换 | 兜底策略未经确认、固定顶栏与内容错位、窄屏关键操作无法使用 |

判据：按批准布局从宽到窄实测，列数/导航变化符合项目设计，关键文本和操作可达、无意外裁切或错位；不同语言、缩放和辅助技术验证覆盖真实使用范围。不能仅因超过四列或出现折叠就判失败。

## 2. 可选示例（不是项目现状）

以下保留组件/布局实现的具体参考，方便采用相似设计的项目复用。代码、像素、库名、检查命令都只在项目已明确采用且当前任务涉及时适用；它们不是默认配置或历史事实，也不覆盖现有设计。若不同，只引用项目真实选择，不维护第二份改填表。示例中的“禁用/统一/铁律”只描述该示例方案自身的约束，不能成为其他项目的迁移命令。

### 2.1 UI 实现栈

UI、状态、路由、表单、样式、token 命名及公开入口取项目单一依据；缺失就说明未知，不从以下范例选出一个默认栈。

业务侧 UI import **统一入口**：`from '<待填充:ui-entry>'`（语法按项目语言调整）

### 2.2 Token 命名（单一前缀 `--<待填充:token-prefix>-*`）

声明位置：`<待填充:web-repo>/<待填充:theme-config-path>`
注入方式：`applyCSSVariables(mode, schemeId)` 写入 `:root.style`
样式框架映射：`<待填充:web-repo>/<待填充:style-config-path>`

#### 颜色 token

| 用途 | Token | Tailwind 类 |
|---|---|---|
| 主色 | `--<待填充:token-prefix>-primary` | `bg-primary` / `text-primary` |
| 主色变体 | `--<待填充:token-prefix>-primary-hover` / `--<待填充:token-prefix>-primary-weak` / `--<待填充:token-prefix>-primary-soft` / `--<待填充:token-prefix>-primary-glow` | `bg-primary-hover` 等 |
| 背景层 | `--<待填充:token-prefix>-background` / `--<待填充:token-prefix>-surface` / `--<待填充:token-prefix>-surface-elevated` | `bg-background` 等 |
| 文字层 | `--<待填充:token-prefix>-foreground` / `--<待填充:token-prefix>-foreground-2` / `--<待填充:token-prefix>-foreground-3` / `--<待填充:token-prefix>-muted` | `text-foreground` 等 |
| 边框 / 分隔 | `--<待填充:token-prefix>-border` / `--<待填充:token-prefix>-border-soft` / `--<待填充:token-prefix>-divider` | `border-border` 等 |
| 状态色 | `--<待填充:token-prefix>-success` / `--<待填充:token-prefix>-warning` / `--<待填充:token-prefix>-error` | `bg-success` 等 |
| 填充 | `--<待填充:token-prefix>-fill-soft` / `--<待填充:token-prefix>-chip-bg` / `--<待填充:token-prefix>-input-bg` / `--<待填充:token-prefix>-table-header` | inline / Tailwind |
| 阴影 | `--<待填充:token-prefix>-shadow-card` / `--<待填充:token-prefix>-shadow-card-hover` / `--<待填充:token-prefix>-shadow-modal` | inline |
| 弹窗遮罩 | `--<待填充:token-prefix>-mask-gradient` / `--<待填充:token-prefix>-mask-blur` / `--<待填充:token-prefix>-modal-radius` | inline |

#### 字号 scale（7 个角色，每角色单一字号）

| 角色 | 字号 | 用途与位置 |
|---|---|---|
| 超大 H1 | 36px | 首页 Hero 大标题（`DisplayTitle`） |
| H1 | 22px | 所有页面顶部 H1（`PageHeader.title` 等） |
| Hero 资源名 | 18px | Hero 业务资源名（`ResourceBillboard.name`） |
| 资源名 / 内页 H3 | 16px | 列表卡片资源名 + 详情页区块 H3 |
| 正文 | 14px | 描述 / 副标题 / breadcrumb / FormLabel / DataTable 行 + 表头等 |
| 小标签 chip | 12px | Chip / CatChip / 角标（NEW 等） |
| 副信息小字 | 11px | 卡片版本号 / vendor·size |

**禁档外字号**：`fontSize: 13 / 12.5 / 15 / 17 / 28` 在业务或 ui/ 组件里出现报错。

**正文字号下限 14px** 适用：表格行主字段 / 按钮文字 / 表单 input / FormLabel / 状态文字。

#### 间距 / 圆角 / 字体栈

```ts
// 字体
'--<待填充:token-prefix>-font':         '-apple-system, BlinkMacSystemFont, "SF Pro Display", "SF Pro Text", "PingFang SC", sans-serif',
'--<待填充:token-prefix>-line-height':  '1.5714285714285714',

// 间距档位
'--<待填充:token-prefix>-spacing-xxs':  '4px',
'--<待填充:token-prefix>-spacing-xs':   '8px',
'--<待填充:token-prefix>-spacing-sm':   '12px',
'--<待填充:token-prefix>-spacing':      '16px',     // 默认
'--<待填充:token-prefix>-spacing-md':   '20px',
'--<待填充:token-prefix>-spacing-lg':   '24px',
'--<待填充:token-prefix>-spacing-xl':   '32px',
'--<待填充:token-prefix>-spacing-xxl':  '48px',

// 圆角
'--<待填充:token-prefix>-border-radius':     '6px',
'--<待填充:token-prefix>-border-radius-xs':  '2px',
'--<待填充:token-prefix>-border-radius-sm':  '4px',
'--<待填充:token-prefix>-border-radius-lg':  '12px',

// 页面水平 padding —— 具体适用页面由项目布局规范定义
// 有侧栏页面的紧凑 padding 也应由项目 token 定义
'--<待填充:token-prefix>-hpad': '80px',
```

### 2.3 组件命名示例（仅采用相应系统时）

- **shadcn 标准组件**：`Dialog` / `DialogContent` / `Select` / `SelectTrigger` / `SelectValue` / `SelectContent` / `SelectItem` / `Popover` 等
- **设计系统专属**：`Card` / `PageHeader` / `DataTable` / `SearchBar` / `Pagination` / `StatusDot` / `ActionLinks`（无 V1/V2 前缀）
- **视觉装饰原子**：`Illust` / `AppIcon` / `HeroBillboard` / `SquircleAppIcon` / `CatChip` / `Chip` / `DisplayTitle` / `SectionHeader`
- **旧 JS token namespace 的迁移示例**：项目确认弃用旧命名并完成消费者迁移后才禁用；若采用本例 CSS 变量，可用 `var(--<待填充:token-prefix>-*)`，不宣称任何项目已删除旧入口

### 2.4 antd-shape 到组合 API 的迁移示例

仅供已经选择相应迁移的项目参考；逐项核真实组件版本/行为，项目没有该迁移就不采用此表。合法兼容层按 §1.6 执行，不能把表中旧用法一律判错：

| 旧用法（antd-shape） | 新用法（shadcn 标准） |
|---|---|
| `<Button type="primary">` | `<Button variant="primary">` |
| `<Button danger>` | `<Button variant="danger">` |
| `<Button htmlType="submit">` | `<Button type="submit">`（HTML 原生） |
| `<Button size="small/middle/large">` | `<Button size="sm/default/lg">` |
| `<Select options={[...]}>` | `<Select><SelectTrigger>...</SelectTrigger><SelectContent><SelectItem>...</SelectItem></SelectContent></Select>` |
| `<Select.Option>` | `<SelectItem>` |
| `<Popover content={...} trigger="click">` | `<Popover><PopoverTrigger asChild>...</PopoverTrigger><PopoverContent>...</PopoverContent></Popover>` |
| `<Input.Password>` | `<PasswordInput>` |
| `<Input.TextArea>` | `<Textarea>` |
| `Modal.confirm({...})` | `<AlertDialog>` 嵌套 或 `confirmDialog({...})` helper |

### 2.5 SSOT 组件实现规范

#### PageHeader（所有页面 H1 标题区唯一来源）

```tsx
<PageHeader
  title="资源中心"
  subtitle={`共 ${count} 项资源`}
  extra={<SearchBar onChange={...} onRefresh={...} />}
/>
```

两套水平 padding 模式：

| 模式 | 触发 | px | 适用 |
|---|---|---|---|
| 默认 | 不传 prop | `<待填充:page-padding>` | 无 sidebar 页面 |
| 紧凑 | `compact` prop | `px-8`（32） | 有 SecondaryNav 的二级菜单页面 |

**H1 字号 22 + leading-none + pt-7** —— 跟 SecondaryNav 模块标题底部对齐（基线 50px）。

#### 二级菜单页布局（数量与页面由项目定义）

```tsx
<div style={{
  minHeight: 'calc(100vh - 64px)',
  background: 'var(--<待填充:token-prefix>-background, #f6f8fc)',
  margin: -24,
  width: 'calc(100% + 48px)',
}}>
  <PageHeader compact title="..." subtitle="..." extra={<SearchBar .../>} />
  <div style={{ padding: '0 32px 36px' }}>
    <DataTable ... />
  </div>
</div>
```

内页元素左 padding = 32px（距 SecondaryNav 右边线 32px）。

#### DataTable 内置规范

- **行高**：`min-h-[68px]`（跨表统一最小行高）
- **Name 列**：禁塞 icon + chip 等额外元素；辅助分类单独成列
- **排序**：关键字段全部 `sortable: true`
- **操作列**（`ActionLinks`）：
  - 状态不可用 → `disabled: true`（链接置灰，位置保留）
  - 无权限 → `hidden: true`（完全不渲染）
  - 禁加 icon（跨表必不一致）
  - 字号 `text-[14px]`（同正文）

#### 业务卡片骨架（`ResourceCard` 等领域卡片共用）

| 维度 | 值 | 说明 |
|---|---|---|
| 容器 | `bg-surface` + `1px border-soft` + `radius 14` + `padding 16` + `minHeight 220` | `position: relative` + `overflow: hidden`（为右上角标） |
| 容器 gap | `10` | — |
| Hover | `translateY(-2px)` + `shadow` 加深 | 鼠标移开还原 |
| Header 行 | `icon 52 + name + vendor/size 11px muted` | — |
| Name | `fontSize 16 / fontWeight 700 / 单行省略` | — |
| Description | `fontSize 14 / WebkitLineClamp 2` | 2 行 clamp |
| Chips 行 | 业务字段下移：`slice(0, 4)` | 第一个 accent，其余 chip-bg |
| CTA | 独立底行右对齐 `h28 px18 radius999 fontSize14 fontWeight600` | 禁加 boxShadow |

Chip 默认底色：`--<待填充:token-prefix>-chip-bg: rgba(60,90,180,0.08)`。non-accent chip 走 `color-mix(in srgb, ${accent} 14%, transparent)`。

#### NewBadge 角标

| 维度 | 值 |
|---|---|
| 位置 | `position absolute / top 0 / right 0` |
| 形状 | 右上角 + 左下角圆角 = `cardRadius`（14）；其他角 radius 0 |
| 尺寸 | `height 22 / padding 10-12 / fontSize 12` |
| 视觉 | 主题色背景 `var(--<待填充:token-prefix>-primary)` / 白字 / 斜体 / fontWeight 700 |

#### SecondaryNav 二级菜单（`<待填充:secondary-nav-path>`）

业务页面禁自抄 sidebar；通过 `MainLayout` 自动注入。

| 维度 | 值 |
|---|---|
| 容器 | `w-[200px]` + `py-7 px-3.5` + `bg-surface` + 右 `border-divider` |
| 模块标题 | `18px / fontWeight 700 / tracking -0.3` 纯文字（不带 icon） |
| 列表项 | `13px label + 16px icon` + `py-2 pr-3 pl-3` + `gap-2` + `rounded-lg` |
| 列表项 border-l | **`3px` 一直占位**（idle transparent / active primary）——禁止只 active 加 |
| 列表项 active | `bg-primary-weak + text-primary + fontWeight 600` |
| 列表项 hover | **无反馈** |

#### 智能助手悬浮交互组件（可选；`<待填充:assistant-module-path>`）

业务页面禁自抄；悬浮球 + 浮窗由统一模块自持，并由应用入口全局挂载（框架归属由本项目设计文档确认）。

**悬浮球 FloatingButton**：

| 维度 | 值 |
|---|---|
| 尺寸 | `82×82`（移动端 `64×64`） |
| 默认位置 | 右下 `24/24` |
| 拖动 vs 点击 | 移动距离 `< 5px` 算点击（开浮窗）；`≥ 5px` 算拖动（Pointer Events，鼠标 + 触屏 + 触控笔） |
| 边界 clamp | 永不超出 viewport；resize 自动 clamp |
| 位置持久化 | `localStorage['<待填充:assistant-button-position-key>']` |
| 拖动连续性 | `setPointerCapture`（快速移出仍继续拖） |
| 视觉反馈 | cursor `grab → grabbing`；拖动中禁 hover 图标切换 + pulse |
| touch-action | `none`（移动端拖动不触发页面滚动） |

**浮窗 ChatSidebar**：

| 维度 | 值 |
|---|---|
| 尺寸 | `400×753`；平板 (769-1024px) `360×678`；移动 (<768px) 全屏 |
| 默认位置 | 右下 `24/24` |
| 拖动 | `Header 区当 drag handle`；点 button/input/textarea/a 等不触发拖（`target.closest()` 过滤） |
| 位置持久化 | `localStorage['<待填充:assistant-panel-position-key>']` |
| 圆角 / 阴影 | `16px` / `0 8px 32px rgba(0,0,0,0.12)` |
| 边框 | 无 |
| Header padding / 标题 | `20px 20px 12px 20px` / 标题 `16px` |
| 进入动画 | `popIn 0.3s ease`（opacity + translateY + scale） |

**欢迎区 WelcomeSection**（首次打开 / 无消息时）：avatar `56×56` 左 + 欢迎语右（`14px / 600 / 行高 1.6`）；副标题 `13px / --<待填充:token-prefix>-muted`；换一批按钮 `--<待填充:token-prefix>-muted` 灰（非主色，hover 才加深）；推荐问题 3 条（padding `10/14`、圆角 `8`、边框 `--<待填充:token-prefix>-divider`、hover 主色边）。

**输入框 ChatInput**：textarea 默认 `160px`（min 80 / max 320，顶部 `8px` drag handle 可拖高）；无边框 + 直角（由 sidebar `16px` 圆角 + overflow:hidden 裁切）；focus 视觉 = 顶部 `2px` 主色高光直线（`--<待填充:token-prefix>-primary-glow` 28%，**不画整框 border**）；padding `top 14 / right 56 / bottom 56 / left 18`；发送按钮 `position:absolute` 右下 `36×36` 圆形（svg 原色、无背景圆）；思考态 `send.svg → thinking.svg` **不旋转**；Enter 发送 / Shift+Enter 换行。

**消息列表 MessageList**：头像 `24×24` 无背景；用户气泡主色背景白字（跟 scheme 切换）；助手气泡白底黑字；思考态在 ChatInput 不进气泡。

#### 首页 dashboard 组件（可选）

**StatCard 资源总览卡**：数量、指标和数据源由对应特性设计确定；展示采用图标 + 数值 + 标签文字的统一骨架。

**QuickAction 快捷入口**：数量和业务动作由项目确定；采用圆角矩形 + 浅色背景 + 图标 + 文字的统一骨架。

**响应式断点**（首页推荐区列数）：

| 断点 | 主资源卡列数 | 次资源卡列数 |
|---|---|---|
| Desktop `≥1200px` | 4 | 3 |
| Tablet `768-1199px` | 2 | 2 |
| Mobile `<768px` | 1 | 1 |

#### 标签色系 tagColorMap（业务标签 → 颜色，数据驱动映射）

标签类型可扩展；前端用 `tagColorMap` 把后端下发的标签语义映射到设计 token，不在组件中硬编码业务标签、裸色值或资源类型。具体映射由项目设计系统登记。

#### 登录页（满幅沉浸式 · `<待填充:login-page-path>`）

登录页是少数**满幅沉浸式**页面（无 PageHeader / 无 sidebar）。布局 = 左 3D 主视觉 + 右细长表单栏：

| 区 | 规范 |
|---|---|
| 容器 | `display:flex` 横向；`background var(--<待填充:token-prefix>-background)`；窄屏（≤900px）堆叠 |
| 左 Hero | `flex:1`；3D 插画 `object-fit:cover` 满幅；叠 4 层——主色泛光（`heroGlow`，深色 `mix-blend-mode:screen`）/ 右缘羽化（→`--<待填充:token-prefix>-background`）/ 底部 scrim / 标语 overlay |
| 标语 overlay | slogan 三段 `font-size 22 / weight 800 / color var(--<待填充:token-prefix>-primary)`，分隔点 `color-mix(primary 45%)`；tagline `14px / --<待填充:token-prefix>-foreground-2`。**文案全走 i18n `t.login.*`** |
| 右 Rail | `width clamp(380px,30vw,468px)`；`background var(--<待填充:token-prefix>-surface)`；左缘 `box-shadow: -1px 0 0 var(--<待填充:token-prefix>-border-soft), -28px 0 60px -34px var(--<待填充:token-prefix>-primary-glow)` |
| Rail 顶部 | 语言 / 主题切换（`Button variant="ghost"`），`top:22 right:36` |
| Rail 居中组 | 品牌（logo+名 22px）/ 欢迎 H1 22px / 副标 14px / 表单 / 登录键；底部版权 12px `--<待填充:token-prefix>-muted` |
| 输入框 | `<待填充:ui-entry>` 的 `Input`/`PasswordInput`，具体尺寸由项目设计 token 决定 |
| 登录键 | `Button variant="primary" block size="lg"`，`h-[52px]` + `letter-spacing 4px` + `box-shadow ... var(--<待填充:token-prefix>-primary-glow-strong)` |

铁律：

- **颜色随 token**——本例沉浸装饰（泛光/羽化/scrim/辉光）用 `color-mix()` 从项目 primary/background 派生；若采用此实现，核浏览器兼容与真实主题，不从母本颜色值推定项目历史
- **深色 = 炭黑基底 + 主色泛光叠加**——基底用全局 `--<待填充:token-prefix>-background`（跟内页一致），沉浸感靠 `heroGlow` 主色泛光层，登录页深色不另立一套 navy token
- **字号收敛 7 档**——slogan/welcome/brand 22 · 正文/tagline/副标 14 · 版权 12，无 14.5/27 等档外值
- **构建模式不写死**——登录走 `authService.login`（mock 分支在 service 层由 `isMock` 处理），组件内**不写死任何 mock 数据**；保留品牌 store（白标 logo/名）+ 主题/语言 store 接线

### 2.6 ESLint 守护（防历史 API 回滚）

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

CSS 守护（CI 时 grep，应返回 0 结果）：

```bash
grep -rE "--v1-|--<待填充:token-prefix>-color-" packages/*/src --include="*.css" --include="*.tsx" --include="*.ts"
```

### 2.7 验收准入

以下命令仅展示对应范例的检查位置；实际脚本、范围、废弃清单、字号和兼容例外必须从项目单一依据解析。不是必须执行的固定清单，不能搜索注释中旧库名就认定生产回滚。

| # | 验收项 | 命令 |
|---|---|---|
| 1 | 全部 workspace package 构建通过 | `pnpm run -r build` |
| 2 | ESLint 0 error | `pnpm run lint` |
| 3 | 业务 0 处 antd-shape API | `grep -rohE '<Button[^>]*\btype="(primary\|link\|text\|dashed)"' packages/ --include="*.tsx"` 0 结果 |
| 4 | 业务 0 处 V1 命名 | `grep -rln 'V1Card\|V1PageHeader\|V1Table\|V1[A-Z]' packages/*/src` 0 结果 |
| 5 | CSS 变量 0 处 legacy | `grep -rE '\-\-v1-\|\-\-<待填充:token-prefix>-color-' packages/*/src` 0 结果 |
| 6 | ui/ 注释 0 处历史库字面 | 在 `<待填充:ui-source-path>` 检索项目登记的历史库名，0 结果 |
| 7 | 字号档外值 0 处 | `grep -rE 'fontSize: (13\|15\|17\|28)\b\|text-\[(13\|15\|17\|28)px\]' packages/ --include="*.tsx"` 0 结果 |
| 8 | 视觉跟改动前一致 | preview 浏览器人工对照各页面浅 / 深主题 |

### 2.8 文件位置

```text
<待填充:web-repo>/<待填充:frontend-root>/
├── <entry-module>/src/components/ui/  # 全部 UI 组件
├── <entry-module>/src/theme/          # Token 定义 + scheme 注册
├── <entry-module>/src/lib/            # 样式工具
└── <待填充:style-config-path>          # 样式框架 ↔ --<待填充:token-prefix>-* 映射
```

如项目采用 federation，expose 名称以 `<待填充:ui-entry>` 的项目登记值为准。

**迁移清单**：项目应在入口登记已删除的历史组件目录、模块入口、legacy token、JS namespace 与 UI 包；canonical 不写死具体名称。

### 2.9 跨项目复用指引

未来其他项目复用本规则：

- §1 结合 core 和已选设计系统适用，保留责任、兼容、可访问性与回归，不强制某库或迁移方式
- §2 是按需示例，不作项目现状；具体值只引用单一依据，以下维度不在 Rule 重复维护：
  - UI 库（如换成 Mantine / Chakra）
  - Token 前缀（如换成 `--myapp-*`）
  - 字号 scale 具体值（按项目设计稿）
  - 组件命名（按项目设计系统）
  - 禁用 API 清单（按项目废弃栈）
  - ESLint 规则（按项目历史包袱）
