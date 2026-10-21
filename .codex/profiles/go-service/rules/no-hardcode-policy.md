# 禁止硬编码策略

## 1. 通用准则（核心规则，跨项目通用）

**🔒 强制**

- 禁止硬编码服务地址（如 `localhost` / `127.0.0.1` 等任何具体地址）
- 禁止硬编码 IP fallback（`127.0.0.1` 等）
- 禁止硬编码文件路径、端口号、文件权限、业务类型字符串、版本号、魔法数字
- 服务地址必须通过环境变量注入；默认配置中应为空字符串
- 文件路径常量必须统一定义在单一位置（避免跨仓重复定义）
- 提供工具函数封装：`GetServiceAddr()` / `RequireServiceAddr()` / `DetectNodeIP()` 模式

### 1.1 反例

```go
// ❌ 错误：硬编码本地地址
addr := "localhost:<port>"

// ❌ 错误：fallback 127.0.0.1
if ip == "" {
    ip = "127.0.0.1"
}

// ❌ 错误：跨仓重复路径常量
const DataDir = "/var/lib/myapp"  // 已在 common 定义过
```

### 1.2 正例

```go
// ✅ 正确：环境变量注入
addr := consts.RequireServiceAddr(consts.EnvServiceAAddr)

// ✅ 正确：自动探测非 loopback IP
ip := consts.DetectNodeIP()

// ✅ 正确：引用 common 定义
import "github.com/.../common/pkg/consts"
dir := consts.DefaultDataDir
```

## 2. 本项目实现（按选型可替换）

### 2.1 命名空间

- 环境变量前缀：`<待填充:env-prefix>_*`（如 `<待填充:env-prefix>_SERVICE_A_ADDR` / `<待填充:env-prefix>_TRACE_ID`）
- 常量定义位置：`<待填充:common-repo>/pkg/consts/consts.go`

### 2.2 工具函数（位于 `<待填充:common-repo>/pkg/consts`）

- `consts.GetServiceAddr(envKey)` — 读环境变量
- `consts.RequireServiceAddr(envKey)` — 读环境变量，缺失时 panic
- `consts.DetectNodeIP()` — 自动探测非 loopback IP

### 2.3 参考决策

> 🚧 **待对齐**：本段为占位骨架；本项目 落地硬编码治理的 ADR 产生后在此登记（位置惯例：`../../../../.add/project/knowledge/decisions/infrastructure.md`）。

### 2.4 跨项目复用指引

未来其他项目复用本规则：

- §1 通用准则直接复制
- §2 本项目实现按新项目选型替换：
  - 环境变量前缀（如 `MYAPP_*`）
  - 常量定义位置（如 `myapp-common/pkg/consts`）
  - 工具函数命名（保持 `GetServiceAddr` / `DetectNodeIP` 这种通用语义）
