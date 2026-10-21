---
name: add-tdd-go
description: Go 测试规范化——Red → Green → Refactor + 表驱动测试 + 属性测试（gopter）。backend-dev 在新增功能 / 修 bug 前按需启用。；仅适用于 Codex；其他 Host 不使用此投影。
---

> 仅适用于 Codex；其他 Host 不使用此投影。


# TDD-Go — Go 测试规范化

> **方法论基座**：业界 Red → Green → Refactor 纪律。本 SKILL 自包含该纪律在 code profile（Go）下的取值：表驱动 / gopter 属性测试 / 项目测试命名规范，不依赖外部插件。

## 何时启用

- **新增核心业务逻辑**：服务层 / domain 模型 / 复杂算法 → TDD 流程驱动设计
- **修 bug 前**：按 [`systematic-debugging`](../add-systematic-debugging/SKILL.md) Step 1 取得复现证据；已有同版本、同环境且仍有效的失败证据可复用，不重复整套诊断。失败测试按本 SKILL Step 1（Red）建立；取得修复授权后修复 → 测试转绿
- **核心算法 / 不变量**：用属性测试覆盖（如序列化往返 / 排序保持 / 幂等性）

**不适用**：

- 单纯重构（已有测试覆盖时只需保持绿）
- 配置 / 常量改动（无逻辑无需 TDD）
- UI 组件（用 React Testing Library，不归本 SKILL）

## 加载的规则

- [`coding-standards.md`](../../profiles/go-service/rules/coding-standards.md) §1.3 测试规范（文件命名 / 函数命名 / Arrange-Act-Assert）

## 核心理念

按业界 TDD：**Red → Green → Refactor**——

1. **Red**：先写**失败的测试**（明确想要什么行为）
2. **Green**：写**最少代码**让测试通过（不顺手优化）
3. **Refactor**：测试保持绿的前提下重构代码（测试是安全网）

测试分层（单元、集成、E2E）按当前改动风险确定；项目有已确认测试计划时沿用其分层安排，未有计划不阻塞普通开发，不套固定比例。

- 表驱动测试覆盖多输入组合（见 [`coding-standards §1.3`](../../profiles/go-service/rules/coding-standards.md)）
- 属性测试覆盖核心算法不变量

## 测试栈（项目特定）

- **单元 + 集成**：`testing` 标准库 + `testify/assert` + `testify/require`
- **mock**：`testify/mock` 或自定义 interface mock
- **属性测试**：`gopter`（生成器 + 100 次随机循环）
- **race 检测**：`go test -race`
- **覆盖率**：`go test -cover`

---

## 流程

### Step 1：Red — 写失败测试

按测试命名规范（[`coding-standards §1.3`](../../profiles/go-service/rules/coding-standards.md)）：

- 文件：`<source>_test.go`（同包测试） / `<source>_integration_test.go`（集成）
- 函数：`Test<Function>_<Scenario>_<Expected>` —— 三段式

**单元测试模板**（Arrange-Act-Assert）：

```go
func TestUserService_Register_Success(t *testing.T) {
    // Arrange
    db := newTestDB(t)
    svc := NewUserService(db)
    req := newRegistrationRequest(t, "alice") // helper 提供测试输入，不内嵌凭据

    // Act
    user, err := svc.Register(context.Background(), req)

    // Assert
    require.NoError(t, err)
    assert.Equal(t, "alice", user.Username)
    assert.NotEmpty(t, user.ID)
}
```

**跑测试**：

```bash
go test -run TestUserService_Register_Success ./...
# 必须 FAIL（因为实现还没写）
```

**判断**：

- 测试通过 → ❌ Red 阶段不应该通过；可能"过于宽松的断言"或"测试了已有功能"——重写
- 测试 fail 但跑不起来（编译错） → 先让测试**编译通过 + 运行 fail**（不是不能编译）

### Step 2：Green — 写最少代码让测试通过

**动作**：

1. 实施"够通过测试"的最少代码
2. **不顺手优化**——禁加额外功能 / 重构其他文件
3. 跑测试 → 必须 PASS

```bash
go test -run TestUserService_Register_Success ./...
# ✅ PASS
```

**判断**：

- 测试通过 → 进 Step 3
- 测试仍 fail → 看测试 vs 实现差距，继续小步实施
- 顺手改其他文件 → 立刻 revert，遵守"最少代码"

### Step 3：Refactor — 重构 + 测试保持绿

**动作**：

1. 重构 Step 2 写的"够通过测试"代码——按 [`coding-standards`](../../profiles/go-service/rules/coding-standards.md) §1.2 函数复杂度；仅在项目明确选择 DDD 且当前服务涉及该选择时，应用 §1.8 DDD 服务层的对应部分；普通 Go 服务不因此强加 DDD。
2. 每次小步重构 → 跑测试 → 仍 PASS → 继续
3. 重构后测试一次 fail → 立刻 revert 这步重构（测试是安全网）

```bash
# 每次重构后必跑
go test ./...
```

**判断**：

- 没必要重构（代码已最简） → 跳过 Step 3
- 重构破坏测试 → revert，重试更小步骤

### Step 4：表驱动测试（多输入组合）

复杂逻辑用表驱动测试覆盖多 case：

```go
func TestParseDuration_TableDriven(t *testing.T) {
    cases := []struct {
        name    string
        input   string
        want    time.Duration
        wantErr bool
    }{
        {"valid seconds", "10s", 10 * time.Second, false},
        {"valid minutes", "5m", 5 * time.Minute, false},
        {"invalid format", "abc", 0, true},
        {"empty string", "", 0, true},
    }

    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            got, err := ParseDuration(tc.input)
            if tc.wantErr {
                require.Error(t, err)
                return
            }
            require.NoError(t, err)
            assert.Equal(t, tc.want, got)
        })
    }
}
```

**判断**：

- ≥3 个相似 case → 必用表驱动（DRY）
- 单一 case → 直接写测试函数

### Step 5：属性测试（gopter）

核心算法 / 不变量用属性测试——`gopter` 生成 100 次随机输入验证不变量：

```go
import (
    "github.com/leanovate/gopter"
    "github.com/leanovate/gopter/gen"
    "github.com/leanovate/gopter/prop"
)

func TestSerialize_RoundTrip_Property(t *testing.T) {
    properties := gopter.NewProperties(nil)
    properties.Property("serialize-deserialize round trip", prop.ForAll(
        func(s string) bool {
            bytes, err := Serialize(s)
            if err != nil {
                return false
            }
            got, err := Deserialize(bytes)
            return err == nil && got == s
        },
        gen.AnyString(),
    ))
    properties.TestingRun(t)
}
```

**典型场景**：

- 序列化往返：`Deserialize(Serialize(x)) == x`
- 排序保持：`Sort(Sort(xs)) == Sort(xs)`
- 幂等性：`f(f(x)) == f(x)`
- 不变量：`Validate(x) → Process(x) → Validate(result)`

**判断**：

- 关键算法 + 数学性质 → 必用属性测试
- 简单 CRUD → 表驱动够用

### Step 6：跑全集 + race + 覆盖率

实施完成前必跑：

```bash
# 1. 全测试集
go test ./...

# 2. race 检测
go test -race ./...

# 3. 覆盖率（可选——按任务卡片要求）
go test -cover ./...
```

**判断**：

- race 检测报 data race → 必修（并发安全是硬约束）
- 覆盖率低于要求 → 补单元测试

---

## 失败模式速查

| 错误 | 修正 |
|---|---|
| 跳 Red 直接写实现 | 必先 Red——测试驱动设计 |
| Red 阶段测试通过 | 断言过宽，重写 |
| Green 阶段顺手重构 | 一次只做一件事——重构归 Step 3 |
| Refactor 破坏测试不 revert | 立刻 revert，重试更小步骤 |
| ≥3 case 不用表驱动 | DRY——必用表驱动 |
| 核心算法无属性测试 | 用 gopter 加属性测试 |
| 跳 race 检测 | `go test -race` 必跑 |
| 测试名不三段式 | 按 `Test<Function>_<Scenario>_<Expected>` |
| 用 mock 替代真实 DB（集成测试） | 集成测试用真实 DB（按 `feedback_inprocess_teammate_dead` 教训反例——别让 mock 掩盖问题） |

## 与其他规则 / SKILL 的关系

- **调用方**：[`task-implementation SKILL`](../add-task-implementation/SKILL.md) 实施改动期间——启用 go-service profile 且涉及核心逻辑时调本 SKILL
- **bug 修复**：[`systematic-debugging SKILL`](../add-systematic-debugging/SKILL.md) 验证 → 加单元测试覆盖根因路径（防回归）
- **测试规范**：[`coding-standards §1.3`](../../profiles/go-service/rules/coding-standards.md)
- **后端检查命令**：[项目入口](../../../AGENTS.md) 的 profile 登记段
