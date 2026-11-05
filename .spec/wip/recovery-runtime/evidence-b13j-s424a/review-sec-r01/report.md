# B13J-S424A 安全侧独立评审 r01

## 0. 元信息与边界

- 角色：安全侧独立评审（`add-reviewer` / `.codex/skills/add-code-review/SKILL.md`），非设计作者。
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`（只读核对，未变）
- 评审对象：`.add/project/knowledge/design-workspace/identity-auth/design.md`
  - 现行：876660 B / 5109 行 / SHA256 `EF55DC9272F24C7626F028F705B4322B4E33DCFA6D0197545015CA7DE4FE5811`
  - revision：`identity-auth-design-ir01-r61`
  - 改前 r60：854991 B / SHA256 `FED48FA9908F9EADD970A3E57B0B448181C51030A4460B437852009722F34D8D`
- 本次只评 r61 增量：`:3925`–`:4087`，以及同步修改的 `:38`、`:83`、`:3611`、`:3878`、`:3882`、`:3915`。
- 作者证据：`.spec/wip/recovery-runtime/evidence-b13j-s424a/design-revision-r01/report.md`，18793 B / SHA256 `B78CA378F6050B9494BEBE18ABC6C11171E6DB9ECCBB594206EE7C5AA7D4C6FA`
- 上轮阻断证据：`.spec/wip/recovery-runtime/evidence-b13j-s424a/report.md`，10670 B / SHA256 `815CC215DB615B77E176FBEA8CA45079235A0082854F500A50682B47C3443763`
- 上游依据：ADR-0031 r07（`.add/project/knowledge/decisions/framework.md:1115`–`:1169`）、S4.24a 卡面（`.add/project/knowledge/tasks/tasks.md:3185`–`:3215`）、两路 ADR r07 评审报告（见 §5）。
- 只读方法：两轮独立阅读 + 有界上游/下游核对。第一轮核对象身份、r61 增量、作者证据与上轮阻断报告；第二轮沿 ADR-0031 不变量/约束 8、卡面关闭条件与现行代码 `cmd/han-server/recovery_restore.go`、`recovery_config.go`、`internal/recovery/runtime.go` 交叉核对，并只读核对 Go `encoding/json` / `os/exec` 语义。未修改 design、ADR、`cmd/**`、`internal/**`、`tasks.md`，未做 Git 写，未删除或移动文件。
- 未执行：`go test`、`go build`、`go vet`、`gofmt`、Linux 实测。评审环境为 Windows；r61 自身把 (d)(e)(f) 标为 NOT RUN，本次未补任何 Linux 证据。
- 未读取：design.md 全文 5109 行的全部上下文、工作区 293 行并行改动、两路 ADR 评审报告的完整 findings、作者证据之外的实现细节。上述缺口列入 §6。

## 1. 逐题判定

| # | 问题 | 判定 | 一句话结论 |
|---|---|---|---|
| 1 | 批准记录抗伪造 | **FAIL / BLOCKER** | `approval_digest` 只是同文件内可重算的完整性摘要；无签名、外部登记、独立可信副本或独立读取锚，等价自证；否证条件只能发现不重算的误改。 |
| 2 | 复核人独立性 | **FAIL / BLOCKER** | 只有 `reviewer.person_name != issuer.person_name` 字符串比较；无 executor principal、无 reviewer 任命引用、无 ref/角色交叉约束，一人两名可绕过。 |
| 3 | 有效期/撤销 TOCTOU | **FAIL / BLOCKER** | `Prepare` 重读与 `Replace`/`VerifyRestored` 经 `prepared()` 的链条未闭合；现行 `prepared()` 不重读批准文件，`now` 也不中途重取；撤销生效时限与 in-flight 语义未定义。 |
| 4 | 目标凭据处理 | **FAIL / HIGH** | Go 无法保证真正清零；`[]byte` 走默认 `encoding/json` 会变成 base64，不是原 DSN；JSON/管道/子进程/GC 仍有副本；日志/Trace 面基本封闭但未完全闭合。 |
| 5 | 宿主文件边界 | **FAIL / BLOCKER** | 已覆盖 `st_dev`/`st_ino`、owner/mode、ACL、mountinfo、`ld.so.preload`、`ld.so.cache`、RPATH/RUNPATH，但漏 `PT_INTERP`、相对 `DT_NEEDED`、`DT_AUDIT`/`DT_DEPAUDIT`、库内容/挂载、`AT_SECURE`/capabilities、网络隔离、CWD/fd 继承；schema 还缺 `ld.so.cache` 路径/digest 等字段。 |
| 6 | fd-exec 正确性 | **PARTIAL / HIGH** | `execveat(AT_EMPTY_PATH)` 对已核 fd 执行确实消除 executable 的“验后按路径重开”TOCTOU；`/proc/self/fd/3` 在 `/proc` 可用且 fd 3 稳定时等价；Go `Path`/`Args` 语义描述基本正确；但 `Args[0]`“仅作展示”不可强制，且 `execveat` 的 CAP_SYS_ADMIN、root-owned `0600`/`0700` 工具与非 root 执行身份、降权顺序未说明。 |
| 7 | 负向矩阵诚实性 | **FAIL for closure / BLOCKER** | r61 把 (d)(e)(f) 标 NOT RUN 是诚实的，不是伪造 PASS；但 Windows 上 skip/无匹配 exit 0 不满足卡面关闭条件 (4)(5)，不得用 NOT RUN 换 PASS，须在 Linux 目标/CI 补证。 |
| 8 | 是否把 unknown 写成已解决 | **PASS with caveats / LOW** | r61 `:3925`、`:4087` 与作者报告 §9 明确保留 unknown；未发现把真实批准角色、工具/备份、原生回执/外部见证、ACL/挂载实测写成已解决。但摘要自证、`[]byte` 清零、`receipt` 信任锚与 `ld.so.cache` 路径的措辞仍需收紧。 |

## 2. Findings

### B13J-S424A-SEC-R01-F001 — 批准摘要自证，无可信外部锚

- 严重级：**BLOCKER**
- 是否阻断：**是**（阻断按 r61 进入实施与关闭 S4.24a）
- 定位：`design.md:3929`、`:3984`、`:4004`、`:4010`；`design.md:3977`；`framework.md:1115`、`:1133`、`:1134`、`:1156`、`:1161`–`:1162`
- 问题：`approval_digest` 的定义是“取该记录对象、删除 `approval_digest` 键、按固定键序序列化、取 SHA256”。这只是一个自包含完整性校验值，不是签名、不是外部登记证明，也没有要求任何独立可信副本。`approval_ref` 只是“不透明引用（例如部署方记录编号）”，design 未要求该引用可被独立读取、独立验证或与批准文件之外的权威记录交叉核对。`RestorePlan` 只携带 `SourceApprovalRef`/`SourceApprovalDigest`，不携带 `tool.approval_ref`/`tool.approval_digest` 的外部锚；`tool` 记录的摘要只与自身及运行时实测比较。
- 攻击/失败场景：能写批准文件的主体（root、部署方特权进程、或能替换受控挂载视图的同一部署方）改写 `issuer`、`reviewer`、`executable_path`、`executable_sha256`、`st_dev`/`st_ino`、`mount`、`valid_until`、`revocation` 等全部字段，重新计算 `approval_digest`，并同步替换二进制与 mount 视图。文件内所有校验自洽通过，`Prepare`/`Replace` 也会通过；`approval_digest` 不能区分“原始批准”与“被重写后重算的批准”。`0600`/root/只读挂载只防非 root 或误改，不防能写该挂载视图或能重挂载的部署方/root。
- 依据：ADR-0031 r07 `framework.md:1115` 要求“批准记录为备份来源与可执行文件两条独立记录”，并给出“可核判据”；`:1133`/`:1134` 要求批准只能由具名角色签发；`:1156` 要求批准与执行分离；`:1161`–`:1162` 要求两条独立批准记录与可核 SHA/owner/ACL/挂载。r61 的摘要机制不满足“抗伪造”或“独立可信副本”。
- 建议：
  1. 在批准记录之外增加可信锚：批准角色签名（工具/备份系统之外的签名密钥）、外部登记（部署方审计记录/不可变日志）或至少一条与批准文件不同的独立读取路径与独立副本；摘要只声明为完整性校验，不声明为真实性。
  2. 把 `tool` 记录的 `approval_ref`/`approval_digest` 纳入 `RestorePlan` 或 `PhasePermit` 的绑定，避免工具批准只存在于同一文件内。
  3. 明确否证条件：能够重算摘要的未授权写者仍必须被拒绝；补“重写全部字段并重算摘要 + 替换二进制/mount 视图”的负向测试。
  4. 若组织侧无法提供签名/外部见证/独立副本，应显式登记残余风险“批准抗伪造止于部署方/root”，并保持生产 closed，不得以“有摘要”暗示已抗伪造。
- 证据：只读核对上述 design/ADR 行、`internal/recovery/runtime.go:47`–`:52`（`RestorePlan` 无 tool approval 字段）、`cmd/han-server/recovery_restore.go:67`–`:80`（plan wire 无 tool approval 字段）、作者报告 §9（真实批准角色/记录不存在）。未跑实现测试。

### B13J-S424A-SEC-R01-F002 — 复核人独立性不可机核

- 严重级：**BLOCKER**
- 是否阻断：**是**
- 定位：`design.md:3938`、`:3943`、`:3962`、`:4001`、`:4010`；`framework.md:1115`、`:1156`
- 问题：r61 对复核人独立性的唯一可机核判据是 `reviewer.person_name != issuer.person_name`（design `:4001` 的“同一人即记录不成立”，失败分类 `:4010` 的“签发=复核”）。`reviewer.role_name` 可任意填写；`review_ref` 只要求非空；`reviewer` 对象没有 `appointment_ref`，而 `issuer` 有；批准记录没有 executor principal 字段，因此无法机核“独立于执行者”。`review_ref` 也没有要求解析到一条独立、可核的复核记录，或与任命/角色/人员 ID 建立交叉约束。
- 攻击/失败场景：同一自然人使用两个 `person_name`（别名、拼写变体、不同系统账号）即可同时充当 issuer 与 reviewer；同一人可同时是 issuer 与执行身份，因为记录没有 executor 字段；同一 `review_ref` 可复用于多条记录；`role_name` 可填成任意“独立复核人”字符串。所有检查通过，但 ADR 要求的“具名且独立于签发人与执行者”未被满足。
- 依据：ADR-0031 r07 `framework.md:1115`：“复核人须具名且独立于签发人与执行者，同一人兼任签发与复核时该记录不成立”；`:1156`：“批准角色必须具名且不得自批、不得兼任执行者；执行身份不得等于 root，也不得与批准角色共用凭据”。r61 没有给出与这些要求等价的机核判据。
- 建议：
  1. 为 issuer、reviewer、executor 增加稳定 principal ID（不是显示名），并要求 reviewer 的 ID 与 issuer、executor 的 ID 均不同；显示名只作展示。
  2. 给 reviewer 增加 `appointment_ref`，并要求其解析到独立书面任命；要求 `review_ref` 解析到独立复核记录，且该记录的 reviewer/issuer/executor 关系可核。
  3. 在批准记录中增加 executor principal/identity 字段（或引用），使“独立于执行者”可机核；要求 reviewer 的 `role_name` 与 issuer/executor 的角色不同，或至少由 `appointment_ref` 固定。
  4. 补负向测试：同一 principal 双名、同一 `review_ref`、reviewer=executor、reviewer 无任命、reviewer 角色伪造，均须在 Begin 前 closed。
- 证据：只读核对上述 design 行、ADR 行；作者报告 §9 明确真实批准角色/记录不存在；未跑实现测试。

### B13J-S424A-SEC-R01-F003 — 有效期/撤销 TOCTOU 链条未闭合

- 严重级：**BLOCKER**
- 是否阻断：**是**（阻断实施/关闭）
- 定位：`design.md:3929`、`:4006`、`:4008`、`:4010`、`:4085`；`cmd/han-server/recovery_restore.go:247`–`:255`、`:318`–`:338`、`:341`–`:363`、`:385`–`:397`
- 问题：
  1. `design.md:4006` 说 `Prepare` 在 `Journal.Begin` 之前重读文件精确字节并与快照比对；`Replace`/`VerifyRestored` “经 `prepared()` 再次校验有效期与撤销状态”。但现行 `prepared()`（`recovery_restore.go:247`–`:255`）只调用 `source.CheckApproved` 与 `checkRestoreArtifact`，不重开/重读批准文件；r61 没有明确要求 `prepared()` 重读文件、比对字节/metadata、重新解析 `revocation`。进程持有快照时，批准文件在 `Prepare` 后被原子替换为 revoked，`Replace` 可能继续使用旧快照。
  2. `design.md:4008` 说 `now` 取预检注入的时钟、不在校验中途重取。若 restore 持续时间较长，`valid_until` 在 `Replace` 前已过，但 `Replace` 仍可能用预检时的 `now` 通过。
  3. 未定义撤销最大生效窗口、in-flight restore 的撤销语义、`Replace` 执行中撤销的处理、读失败/删除/重挂载/mount 视图变化的 fail-closed 行为。
  4. `design.md:3929` 要求批准文件位于“登记过的受控只读挂载来源”，而 `:4008` 说“撤销只经部署方原子替换批准文件生效”。只读挂载与原子替换的可见性/重挂载语义未说明；原子替换会改变 inode，`:4010` 把 inode 变化归为 `ErrAdmissionBinding`，没有把它识别为撤销状态并返回 `ErrAdmissionClosed`。
  5. `design.md:4085` 说“除 `recoveryRestoreStartupPreflight` 外不得在别处读批准文件或凭据文件”，但 `:4006` 的 `Prepare` 重读和 `Replace`/`VerifyRestored` 的再次校验必然在别处读文件；两者自相矛盾，未定义唯一允许的 revalidation 方法。
- 攻击/失败场景：审批人在 `Prepare` 后撤销批准（部署方原子替换文件），但 `Replace` 仍用旧快照执行整库替换；或 `valid_until` 已过，但 `Replace` 用旧 `now` 放行。攻击者只需在 `Prepare` 与 `Replace` 之间触发撤销，或拉长 restore 时间，即可让已撤销/已过期的批准继续生效。
- 依据：ADR-0031 r07 `framework.md:1115` 的“存在、未撤销、字段齐全且与本次 backup ref 和 artifact digest 精确匹配”；`:1133` 的“未撤销”；`:1161`–`:1162` 的批准记录要求。r61 的撤销/有效期链没有闭合到 `Replace`/`VerifyRestored`。
- 建议：
  1. 在 `Prepare`、`Replace`（执行前与执行后）、`VerifyRestored` 以及实际 exec 前，重开批准文件，精确比对 bytes + `st_dev`/`st_ino`/size/mtime/owner/mode/ACL/mount，并重新解析 `valid_from`/`valid_until`/`revocation`；任一读失败、删除、漂移、重挂载视图变化一律 closed。
  2. 每次校验用当前时钟（或单调时钟 + 明确的最大 restore 时长），不要复用预检 `now`；定义时钟偏差处理。
  3. 定义撤销最大生效窗口与 in-flight 语义：撤销阻止后续 `Replace`/`VerifyRestored`/exec；已开始的工具执行不能回滚 DB 写入时，必须进入 `restore_unknown` 并保留证据，不得宣称成功。
  4. 统一 `:4085` 与 `:4006`：明确“唯一预检入口 + 唯一 revalidation 方法”是仅有的两处读取，其他组件不得读；或删掉“除该函数外不得在别处读”的绝对表述。
  5. 补 active→revoked（Prepare 与 Replace 之间）、valid_until 跨越、文件删除、inode/mount 变化、读失败测试；全部须在 Begin 前或 Replace 前 closed。
- 证据：只读核对 design 行与 `recovery_restore.go:247`–`:255`、`:318`–`:338`、`:341`–`:363`、`:385`–`:397`；未跑实现测试。

### B13J-S424A-SEC-R01-F004 — DSN `[]byte` 清零与“线上形状不变”矛盾

- 严重级：**HIGH**
- 是否阻断：**否（不单独阻断设计方向；但属于实施前必须修复的 HIGH，未修复不得进入实施）**
- 定位：`design.md:3882`、`:4051`–`:4053`、`:4068`；`cmd/han-server/recovery_restore.go:71`–`:80`、`:196`、`:203`–`:207`、`:216`、`:342`、`:346`、`:385`–`:397`、`:592`–`:609`
- 问题：
  1. Go 不能保证“写后清零”：`string` 不可变，无法清零；`string`→`[]byte` 转换会产生副本；`json.Marshal` 会生成包含 DSN 的新缓冲区；`os/exec` 的 stdin 通过 goroutine/pipe 复制到内核 pipe 缓冲；子进程读取后有自己的缓冲；GC 复制、编译器保留、core dump、swap、`/proc/<pid>/mem` 都可能留下副本。`clear(input)` 只清父进程最终 buffer，不能清上述副本。
  2. `design.md:4053` 说把 `target_connection.dsn` 由 `string` 改为 `[]byte`，又说“不改变 stdin JSON 的线上形状（仍是字符串）”。但 Go 默认 `encoding/json` 对 `[]byte` 编码为 **base64 字符串**，不是原 DSN；若子进程按 `{"dsn":string}` 读取原始 DSN，会收到 base64。要“线上形状不变且值是原 DSN”，必须自定义 `MarshalJSON` 或手工 JSON 转义，design 没有要求。
  3. `design.md:4051` 说 DSN 在预检阶段读取并得到进程内不可变快照，`Replace` 复用同一快照；`Close` 只把 `d.dsn = ""`（当前代码 `:392`），底层 string/副本仍在。DSN 在 driver 生命周期内长期驻留。
  4. `design.md:3882`/`:4068` 禁止日志/Trace/stdout/stderr，但错误路径、DB driver 的错误文本、子进程 stderr 的“只记退出分类和 Trace”仍可能携带 DSN；未定义统一的 redaction/错误包装合同、子进程读后清零合同、core dump 抑制（`RLIMIT_CORE`、`PR_SET_DUMPABLE`、`MADV_DONTDUMP`）和同机 root 内存取证边界。
- 攻击/失败场景：子进程收到 base64 DSN，连接失败（功能失败）；或实现者为“保持原 DSN”增加一次 `string`/`[]byte` 转换，留下无法清零的副本；父进程/子进程/pipe/core dump 中的 DSN 被同机 root、崩溃转储、日志或 Trace 读取。设计宣称“写后可清零”，但实际只是尽力而为。
- 依据：ADR-0031 r07 `framework.md:1136`（执行身份只持“经私有 stdin 传入的目标连接配置”）、`:1144`（零继承业务 secret）、`:1156`（执行身份不得与批准角色共用凭据）。r61 的 `[]byte` 方案与这些要求方向一致，但清零/编码/错误面未闭合。
- 建议：
  1. 明确 `target_connection` 的编码：使用自定义 `MarshalJSON`/流式写入，把 DSN 作为 JSON 字符串逐字节转义；禁止默认 `json.Marshal([]byte)` 的 base64 语义；给出 wire 测试向量。
  2. 所有中间 buffer（读取文件、JSON、pipe 写入、receipt、错误文本）在写后/用后清零；子进程合同要求读后清零；父进程 `Close` 清零快照；避免不必要的 `string` 转换。
  3. 对 DSN 做统一 redaction：任何 error/log/Trace/audit 字段不得包含 DSN、DSN 文件路径中的敏感部分、子进程 stderr 原文；只保留分类与哈希/长度。
  4. 明确“尽力清零、不抗同机 root/内存取证/核心转储”的残余风险；若要求更强保证，需使用 `memfd_secret`、`mlock`、`MADV_DONTDUMP`、`PR_SET_DUMPABLE=0`、`RLIMIT_CORE=0` 等并说明限制。
  5. 补 canary 测试：DSN 不出现在 argv/env/log/Trace/stdout/stderr/Journal/audit；子进程收到的 JSON 解码后与原 DSN 逐字节相等；父进程/子进程缓冲区在可控范围内被清零。
- 证据：只读核对 design 行、`recovery_restore.go:71`–`:80`（`json.Marshal` + `DSN string`）、`:196`（`dsn string`）、`:203`–`:207`（构造器）、`:216`（空检查）、`:342`（`privateRestoreInput`）、`:346`（`clear(input)`）、`:385`–`:397`（`Close` 只置空）、`:592`–`:609`（pipe）。Go `encoding/json` 对 `[]byte` 的 base64 行为为语言/库事实；未跑实现测试。

### B13J-S424A-SEC-R01-F005 — 宿主 loader 边界不完整且 schema 缺字段

- 严重级：**BLOCKER**
- 是否阻断：**是**
- 定位：`design.md:3955`–`:3980`、`:3997`–`:4000`、`:4064`、`:4068`、`:4076`–`:4081`、`:4087`；`framework.md:1117`、`:1144`、`:1164`
- 问题：r61.5 已覆盖 `st_dev`/`st_ino`、owner uid/gid/mode、`system.posix_acl_access` 摘要、`/proc/self/mountinfo` 的 `ro`/`nodev`/`nosuid`/无 `noexec`、`/etc/ld.so.preload` 非空拒绝、`ld.so.cache` 摘要、`DT_RPATH`/`DT_RUNPATH` 非空拒绝，以及 artifact no-follow regular file。但仍缺：
  1. **schema 缺字段**：`tool` 对象没有 `ld_so_cache_path`、`ld_so_cache_sha256`、`elf_type`（static/dynamic）、`pt_interp`/interpreter 摘要、`dt_needed`/依赖列表、`security_capability`/setuid 标志等字段；`:4064` 却说“`ld.so.cache`（路径登记在工具记录）”，与 `:3955`–`:3980` 的 schema 自相矛盾。
  2. **解释器/依赖注入**：未检查 `PT_INTERP`（自定义/被替换的解释器）、`DT_NEEDED` 相对路径或含斜杠条目、`$ORIGIN` 在 `DT_NEEDED`/`DT_AUDIT` 中的展开、`DT_AUDIT`/`DT_DEPAUDIT`、`/etc/ld.so.conf`/`ld.so.conf.d` 间接影响。
  3. **库内容与挂载**：`ld.so.cache` 摘要只固定“soname→path”的搜索映射，不固定实际加载库文件的内容；未要求依赖库所在挂载为 `ro`/`nodev`/`nosuid`，未核对实际 `/proc/<pid>/maps` 加载结果。
  4. **ELF 安全属性**：未拒绝 setuid/setgid 位、未检查 `security.capability` xattr，未处理 `AT_SECURE` 对 loader 行为的影响；`mode` 只做相等比较，没有禁止 `S_ISUID`/`S_ISGID`。
  5. **网络隔离**：ADR-0031 r07 `framework.md:1117`/`:1164` 要求“除目标连接外无网络”，但 r61.5 的 Linux host 检查、exec 段、ENV 段均未定义 network namespace/seccomp/firewall/路由限制；子进程可连接任意网络。
  6. **CWD 与 fd 继承**：未规定子进程工作目录（相对 `DT_NEEDED`/工具相对路径的注入面）；未明确除 0/1/2/3 外关闭所有 fd（业务 secret/审批 key/宿主凭据可能通过继承 fd 泄漏）。
  7. **artifact 挂载**：只要求 artifact 为 no-follow regular file 且父目录受保护，未要求 artifact 所在挂载 `ro`/`nodev`/`nosuid`；SHA256 校验与子进程读取之间若有写者，仍可替换内容。
  8. **脚本/解释器**：未禁止 shebang 脚本；`execveat` 对脚本会走解释器路径，解释器不在批准/摘要范围内。
- 攻击/失败场景：攻击者替换 `PT_INTERP` 指向的解释器、在 `DT_NEEDED` 中放入相对路径/`$ORIGIN`、用 `DT_AUDIT` 加载审计库、替换被加载的 libc/依赖库（cache 摘要不变）、在可写挂载上放恶意库、给工具加 setuid/file capability、或让工具连接外网。r61 的检查全部通过，但实际执行的代码/库/网络边界已超出批准。
- 依据：ADR-0031 r07 `framework.md:1117`（宿主隔离矩阵：固定 argv、no shell、no-follow fd、最小 ENV、零继承业务 secret/审批 key/宿主凭据、除目标连接外无网络）、`:1144`（动态加载/注入变量与 PATH 约束）、`:1164`（约束 8 第 4 条：POSIX 宿主隔离实测，含 ENV 与“目标连接外无网络”）。r61.5 未覆盖这些项的完整集合。
- 建议：
  1. 优先静态链接；若必须动态链接，固定 `PT_INTERP` 的路径+SHA256、所有 `DT_NEEDED` 的解析路径+内容摘要、`ld.so.cache` 路径+摘要、依赖库挂载属性；exec 后核对 `/proc/<pid>/maps` 实际加载的 inode/路径/摘要。
  2. 在 `tool` schema 补 `ld_so_cache_path`、`ld_so_cache_sha256`、`elf_type`、`pt_interp`/interpreter digest、`dt_needed`/dependency list、`security_capability`/`setuid` 标志；消除 `:4064` 与 schema 的矛盾。
  3. 拒绝脚本/非 ELF；拒绝 `PT_INTERP` 不在批准列表、`DT_NEEDED` 含斜杠/`$ORIGIN`、`DT_AUDIT`/`DT_DEPAUDIT` 非空、`/etc/ld.so.conf` 影响、setuid/setgid、非空 `security.capability`；明确 `AT_SECURE` 处理。
  4. 增加网络隔离（network namespace/seccomp/egress 规则）并纳入负向测试；固定 CWD；关闭除 0/1/2/3 外所有 fd；artifact 挂载要求 `ro`/`nodev`/`nosuid`。
  5. 若无法做到上述完整边界，保持生产 closed，不得把“检查了 preload/cache/RPATH”当作完整 loader 隔离。
- 证据：只读核对 design schema 与 host 检查行、ADR 行；`:3955`–`:3980` 无 cache path/digest/interpreter/dependency 字段；`:4064` 声称“路径登记在工具记录”；未跑 Linux loader/ACL/mount 实测。

### B13J-S424A-SEC-R01-F006 — fd-exec 正确性部分成立，但执行身份/权限链未闭合

- 严重级：**HIGH**
- 是否阻断：**否（不单独阻断设计方向；但按当前示例/措辞实施会失败或产生歧义，未修复不得进入实施）**
- 定位：`design.md:3997`–`:4000`、`:4066`、`:4079`；`cmd/han-server/recovery_restore.go:470`–`:477`、`:592`–`:609`
- 问题：
  1. **正确部分**：`execveat(fd, "", argv, envp, AT_EMPTY_PATH)` 对已核 fd 直接执行，确实消除“校验后按路径重新打开再 exec”的 executable TOCTOU；`exec.Cmd.ExtraFiles` 把 fd 放到子进程 fd 3、`Path="/proc/self/fd/3"` 在 `/proc` 可用且 fd 3 稳定时指向同一 inode；Go 的 `Path` 是执行路径、`Args[0]` 是 argv[0]，r61 的语义描述基本正确。
  2. **`execveat` 前置条件**：`AT_EMPTY_PATH` 按 man page 需要 `CAP_SYS_ADMIN`；若先降权到专用执行身份再 exec，可能没有该 capability；若 root 先 exec 再降权，子进程会以 root 运行一段窗口，和 ADR“专用执行身份”不一致。设计未规定 exec/降权顺序。
  3. **`/proc` fallback**：依赖 `/proc` 已挂载、fd 3 未被覆盖/复用；未定义 `/proc` 不可用、hidepid、fd 3 被工具关闭/重开时的失败分类。
  4. **`Args[0]` 歧义**：`Args[0]` 设为批准绝对路径并称“仅作展示”，但程序可读取 `os.Args[0]` 做资源定位、自校验或安全决策；“仅作展示”不是可强制约束。未规定完整 argv（`Args[1]` 应为 `restore-v1`/`verify-v1`，且不得有额外参数）。
  5. **owner/mode 与执行身份冲突**：`tool` 示例 `owner_uid: 0`、`owner_gid: 0`、`mode: "0600"`；`0600` 没有执行位，`execveat`/`execve` 会因 `EACCES` 失败（即使 root 也要求至少一个执行位）。若改成 root-owned `0700`，非 root 执行身份仍无法执行；若改成执行身份 owned `0500`，`execveat(AT_EMPTY_PATH)` 又可能缺 `CAP_SYS_ADMIN`。设计未说明工具所有权/模式与执行身份的匹配关系，也未禁止 setuid/setgid。
  6. **现有代码**：`openImage` 仍用 `os.Open(t.executable)` 路径打开（`:474`），不是 fd-exec；r61.5 要求改，但 design 的影响面没有明确 `openImage` 的替换点与 fd 生命周期。
- 攻击/失败场景：按示例 `0600` 实施，restore 工具无法执行；按 root-owned `0700` + 非 root 执行身份实施，exec 失败；为绕过而让 root 先 exec，则子进程以 root 运行，违反专用执行身份；`Args[0]` 被工具用于定位资源时，工具可能读取批准路径之外的资源；`/proc` 不可用时 fallback 失败但分类不清。
- 依据：ADR-0031 r07 `framework.md:1134`（fd-exec：`fexecve`、`execveat(AT_EMPTY_PATH)` 或 `/proc/self/fd/N` 等价方式，不得验后按路径重开）、`:1135`（专用执行身份、禁止 shell/su/sudo 包装）、`:1141`（固定 argv、无 shell、无附加 argv）。r61.5 的机制方向符合，但权限/顺序/argv 细节不足。
- 建议：
  1. 首选 `execveat(AT_EMPTY_PATH)`，明确调用者需要 `CAP_SYS_ADMIN`、fd 打开方式（`O_PATH`/`O_RDONLY`）、exec 与降权的顺序；若不能同时满足，明确 fallback。
  2. 对 `/proc/self/fd/3` fallback 写明 `/proc` 挂载、fd 3 稳定性、子进程 fd 关闭策略、失败分类（`ErrAdmissionUnsupported`/`ErrAdmissionIncomplete`）。
  3. 固定完整 argv：`Args[0]` 批准路径（仅 argv0），`Args[1]` 为 `restore-v1`/`verify-v1`，无其他参数；在工具合同中禁止用 argv[0] 做安全/数据定位。
  4. 明确工具 owner/mode 与执行身份：要么执行身份 owned `0500` + `/proc` fallback，要么 root-owned `0700` + root exec 后立即降权并给出窗口证据；禁止 setuid/setgid；示例 `mode: "0600"` 必须改为可执行模式。
  5. 把 `openImage` 从路径打开改为 host 文件返回的已核 fd，明确 fd 生命周期与关闭责任。
- 证据：只读核对 design 行、`recovery_restore.go:470`–`:477`（`os.Open` 路径）、`:592`–`:609`（pipe）；Go `os/exec` 的 `Path`/`Args`/`ExtraFiles` 语义；未跑 Linux fd-exec 实测。

### B13J-S424A-SEC-R01-F007 — NOT RUN 不满足卡面关闭条件 (4)(5)

- 严重级：**BLOCKER**
- 是否阻断：**是**（阻断 S4.24a 关闭）
- 定位：`design.md:4074`–`:4083`、`:3925`、`:4087`；`tasks.md:3196`–`:3207`、`:3208`、`:3209`、`:3215`
- 问题：r61.6 明确把 (d)(e)(f) 在本机 Windows 标为 NOT RUN，并明确“不得用空真断言或断言未 panic 替代”（`:4083`），这一点是诚实的，不构成“伪造 PASS”。但卡面关闭条件 (4) 要求“负向测试证明无 shell、无附加 argv、无自证成功”，(5) 要求“上述 9 条验收命令通过”；Windows 上 `//go:build linux` 或运行时 skip、无匹配测试 exit 0 都不是真实证明。r61 没有提供 Linux 目标/CI 的实际执行证据，也没有定义补证环境与命令。卡面 `tasks.md:3215` 仍为 BLOCKED、未关闭；不得把 NOT RUN 标注当成关闭条件满足。
- 攻击/失败场景：在 Windows 上运行测试得到 exit 0（skip/无匹配），被误读为 (d)(e)(f) 通过；或只跑构造/解析单测，未真实触发 fd-exec、子进程、loader、外部见证，导致“无 shell/无附加 argv/无自证成功/加载器边界”未被验证。若据此关闭 S4.24a，等于用 NOT RUN 换 PASS。
- 依据：卡面 `tasks.md:3208`（负向 (a)–(f) 与 oracle）、`:3209`（关闭条件 7 项，其中 (4)(5) 明确要求负向证明与 9 条命令通过）、`:3196`–`:3207`（9 条验收命令）；r61 `:4079`–`:4081` 的 NOT RUN 标注。r61 的诚实标注不改变“未满足关闭条件”的事实。
- 建议：
  1. 在 Linux 目标环境/CI 上真实补证：
     - (d) 真实 fd-exec + 子进程，证明无 shell、无附加 argv、stdout receipt/exit=0 不被当作成功；
     - (e) 外部见证/批准人核对的等价核验操作与审计字段（见证端点身份、单调序列、读取时间、核验人/系统身份），或真实签名路径；
     - (f) 实际 loader 配置、`PT_INTERP`、`DT_NEEDED`、ELF RPATH/RUNPATH、依赖库内容/挂载与网络边界；
     - 9 条验收命令在 Linux 目标/CI 上真实通过，保留命令日志、JSON 摘要、哈希与退出码。
  2. 在卡面/证据报告首段明确“Windows NOT RUN 不是 PASS”；未补证前 S4.24a 保持 BLOCKED，不得勾选关闭条件。
  3. 若无法取得 Linux 目标环境，登记为未验证边界并保持 closed，不得以 skip/无匹配 exit 0 代替。
- 证据：只读核对 design 负向矩阵、卡面关闭条件与 9 条命令、`tasks.md:3215`（BLOCKED、未关闭）；未跑任何 Linux 测试；本机 Windows 结构性不可达。

### B13J-S424A-SEC-R01-F008 — 规范化/措辞/示例精度不足

- 严重级：**LOW**
- 是否阻断：**否**
- 定位：`design.md:3969`–`:3970`、`:3973`–`:3974`、`:3977`、`:3984`、`:3993`、`:4002`、`:4008`、`:4064`
- 问题：
  1. `approval_digest` 的固定键序只对顶层/source/tool 示例可见；嵌套 `issuer`/`reviewer`/`revocation`/`acl`/`mount`/`receipt` 的键序、字符串转义（HTML escape、Unicode  normalization）、整数格式、数组顺序未全部定义。不同实现可能产生不同摘要。
  2. `system.posix_acl_access` 的“条目排序加十六进制小写”未定义二进制/文本编码、tag 顺序（`ACL_USER_OBJ`/`ACL_USER`/`ACL_GROUP_OBJ`/`ACL_GROUP`/`ACL_MASK`/`ACL_OTHER`）、mask 是否包含、default ACL 是否包含、空 ACL 的表示；`acl.digest` 的可复现性不足。
  3. `reviewed_at` 没有 `reviewed_at >= issued_at`、`reviewed_at <= now` 等时间关系；`valid_from`/`valid_until` 没有最大有效期约束（“不得永久”但可填 100 年）。
  4. `mount.options` 的数组顺序/重复项/等价写法未定义；`DT_RPATH`/`DT_RUNPATH` 写成“ELF 头中”，实际位于动态 section；`st_dev`/`st_ino` 用 JSON number，超过 2^53 时非 Go 生成器可能丢精度，建议十进制字符串或明确 `uint64` 解析。
  5. `tool.mode` 示例 `"0600"` 与可执行性冲突（见 F006）；`tool.receipt` 有 `trust_anchor_ref` 但没有解析/验签/审计字段（`read_at`、`verification_result`、`audit_record`），与卡面关闭条件 (e) 的“等价核验操作与审计字段”不完全对应。
- 攻击/失败场景：不同实现/不同 ACL 工具产生不同摘要，导致合法批准被拒或非法批准被接受；`reviewed_at` 早于 `issued_at` 仍通过；`st_dev`/`st_ino` 精度丢失导致错误匹配；示例 `0600` 被照抄导致工具不可执行。
- 依据：r61.2 的 canonicalization 描述、`:4002` 的有效期/撤销字段、`:4064` 的 host 检查；卡面 `tasks.md:3208` (e) 的审计字段要求。该项不改变 F001/F002/F003/F005 的阻断结论。
- 建议：给出精确的 canonical JSON/ACL 编码算法与测试向量；补时间关系与最大有效期；明确 mount options 顺序/去重；`DT_RPATH`/`DT_RUNPATH` 改称动态 section；`st_dev`/`st_ino` 用十进制字符串或 `uint64`；修正 `mode` 示例；补 `receipt` 验签/审计字段或明确其属于 unknown。
- 证据：只读核对上述 design 行；未跑实现/ACL 实测。

## 3. 逐题证据与未闭合点（补充）

1. **批准记录抗伪造**：`approval_digest` 只覆盖“删除自身键后的记录”，任何能写文件的主体都能重算；`approval_ref` 只是不透明引用，没有独立读取/验证合同；`RestorePlan` 不含 tool approval 锚。否证条件“摘要不符/字节漂移”只能发现不重算的误改，不能发现可重算的伪造。→ F001。
2. **复核人独立性**：唯一比较是 `person_name`；`reviewer` 无 `appointment_ref`；`review_ref` 非空即可；无 executor principal。→ F002。
3. **有效期/撤销 TOCTOU**：`Prepare` 重读 + `Replace`/`VerifyRestored` 经 `prepared()` 的链条依赖 `prepared()` 重读文件，但 design 未明确、现行代码未实现；`now` 不中途重取；只读挂载 vs 原子替换、`:4085` vs `:4006` 矛盾。→ F003。
4. **目标凭据处理**：Go 无法保证清零；默认 `json.Marshal([]byte)` 是 base64；JSON/pipe/子进程/GC 副本；日志/Trace 面基本封闭但错误路径/子进程 stderr/core dump 未完全定义。→ F004。
5. **宿主文件边界**：已覆盖项见 r61.5；缺口包括 schema 字段、`PT_INTERP`、`DT_NEEDED`、`DT_AUDIT`/`DT_DEPAUDIT`、`ld.so.conf`、库内容/挂载、`AT_SECURE`/capabilities、网络隔离、CWD/fd 继承、artifact 挂载、脚本。→ F005。
6. **fd-exec 正确性**：机制方向正确；`execveat`/`/proc` 的前置条件、CAP_SYS_ADMIN、owner/mode 与执行身份、降权顺序、完整 argv、`Args[0]` 歧义未闭合。→ F006。
7. **负向矩阵诚实性**：r61 的 NOT RUN 标注诚实；但卡面 (4)(5) 要求真实负向证明与 9 条命令通过，Windows 无法满足；必须在 Linux 目标/CI 补证，否则保持 BLOCKED。→ F007。
8. **unknown 是否被写成已解决**：r61 `:3925`、`:4087` 与作者报告 §9 明确保留 unknown；未发现越界宣称。但 F001/F004/F005/F008 的措辞仍可能让读者误以为摘要=真实性、`[]byte`=可清零、loader 检查=完整隔离、`receipt` 字段=已验证。→ F008/§4。

## 4. 前置协调（非 r61 增量 finding，但影响能否按 r61 实施/关闭）

### 4.1 ADR-0031 r07 状态冲突

- `framework.md:1129` 仍写“本 ADR 现行修订已是 `adr-0031-r07`，r07 尚未复评，故决策接受级对现行修订尚未达成——在 r07 被架构与安全两路接受之前，`identity-auth-S4.24a` 及任何实施/验证卡不得启动（启动门禁回到 blocked）”。
- `tasks.md:3195` 写“`adr-0031-r07` 为现行修订，已于 2026-11-02 被两路独立复评接受”，并列两路报告：`review-arch-r01` 9964 B / `EC9361004EF31091DC51F2B3F272264C8DC98544D8C42B2F482D18E5BE656CCB` / PASS WITH CONDITIONS；`review-sec-r01` 18884 B / `DA757AD16C73BC3F932371B4FC342DEC32F427E05D86DB211ED4F7123CDC72DC` / PASS WITH CONDITIONS。
- 实际核对：两路报告存在且哈希匹配；两路 verdict 均为 **PASS WITH CONDITIONS**，不是无条件 PASS。安全侧报告明确“F001 阻断 S4.24a 在 r07 两路接受前的启动”，并要求规划角色更新 `tasks.md`/Plan 引用、登记报告路径+SHA+verdict；架构侧报告明确“不是 9(b) 双路门禁的整体完成，也不是人的接纳、生产批准或 S4.24a 解锁”。
- 结论：`tasks.md:3195` 的文字不能当作人的接纳或生产批准；`framework.md:1129` 与 `tasks.md:3195` 的状态不一致，Lead/规划角色必须先对账。对账前，不得以卡面文字主张 S4.24a 已解锁；r61 的评审结论也不替代该对账。

### 4.2 写集冲突

- `tasks.md:3189`–`:3194` 当前写集只有 4 个代码文件：`cmd/han-server/recovery_restore.go`、`recovery_runtime.go`、`main.go`、`recovery_restore_production_integration_test.go`。
- 作者报告 §8 列出 6 个修改 + 13 个新增 = 19 个路径，包括 `recovery_config.go`、`recovery_config_test.go`、`recovery_restore_test.go`、`internal/recovery/restore_source.go`、`recovery_restore_approval.go`/`_test.go`、`recovery_restore_target.go`/`_test.go`、4 个平台文件及测试等。
- `design.md:4087` 只列部分影响面（`recovery_restore.go`、`recovery_runtime.go`、`main.go`、`recovery_config.go` 及对应测试；新增 `internal/recovery/restore_source.go` 与四个平台文件及测试），未列 `recovery_restore_approval.go`、`recovery_restore_target.go` 等批准/目标文件；与 `:4076` 的负向矩阵提到的 `recovery_restore_approval.go` 也不一致。
- 结论：按当前 4 文件写集实施会越界或再次 BLOCKED；Task Planner 必须先按作者 §8/design 影响面更新卡面写集，Lead 再授权实施。设计评审不替 Task Planner 改卡。

### 4.3 证据报告哈希漂移

- `tasks.md:3215` 引用证据报告 `10030 B / 4CC0F4678373E0AEFB540EF54F97025BE70349970368731A998E0CB4F404B7C7`。
- 当前实测 `.spec/wip/recovery-runtime/evidence-b13j-s424a/report.md` 为 `10670 B / 815CC215DB615B77E176FBEA8CA45079235A0082854F500A50682B47C3443763`。
- 作者报告 §1.2 已登记该漂移为 unknown，并说明“漂移原因与漂移前后正文差异为 unknown；若 Lead 需要以给定哈希为唯一基准，须先裁定以哪一份为准”。
- 结论：Lead 需裁定基线；在裁定前，不得把旧哈希或新哈希单独当作唯一关闭证据。该漂移本身不阻断设计评审，但影响证据链可核性。

## 5. 上游依据与核对范围

- ADR-0031 r07：`framework.md:1115`（批准主体最小字段/独立性）、`:1116`（执行主体）、`:1117`（宿主隔离）、`:1118`（原生耐久回执）、`:1133`–`:1147`（不变量 1–6）、`:1160`–`:1169`（约束 8 九条解除条件）。
- 卡面：`tasks.md:3185`–`:3215`（S4.24a 目标、4 文件写集、9 条验收命令、负向 (a)–(f)、7 项关闭条件、BLOCKED 状态）。
- 现行代码只读核对：`cmd/han-server/recovery_restore.go:71`–`:80`、`:196`、`:203`–`:207`、`:216`、`:247`–`:255`、`:318`–`:363`、`:385`–`:397`、`:470`–`:477`、`:592`–`:609`；`cmd/han-server/recovery_config.go:20`–`:83`；`internal/recovery/runtime.go:47`–`:52`。
- 两路 ADR r07 报告：路径/字节/哈希/verdict 见 §4.1；未逐条复核其 findings 全文。

## 6. Verdict 与未验证边界

### Verdict

**CHANGES REQUIRED / BLOCKED**

r61 是一份有用的“可编码合同形状”，且对 unknown 的标注基本诚实；但按当前文本不能作为 S4.24a 实施或关闭的依据。F001/F002/F003/F005/F007 为 BLOCKER；F004/F006 为 HIGH，须在实施前修正；F008 为 LOW。任何一项 BLOCKER 未闭合前，不得写代码、不得派发实施、不得勾选 S4.24a 关闭条件；生产 restore、生产批准主体、专用执行身份、POSIX 隔离实测、原生耐久回执、Windows 生产继续 closed。

### 未验证边界

- 未执行 `go test`、`go build`、`go vet`、`gofmt`；未在 Linux 运行；未实测 `execveat`、`/proc/self/fd/3`、ACL、mountinfo、loader、fd-exec、进程组、网络隔离。
- 无真实批准角色/任命/两条批准记录原件；无真实 restore 工具二进制、SHA256、owner/ACL、mount、`st_dev`/`st_ino`；无真实旧备份、source installation/schema digest、`VerifyRestored` 真实一致性证据。
- 无原生耐久回执、独立读取路径、签名/外部见证、验签信任锚、审计记录；无真实工具/备份系统样本。
- 未读取 design.md 全文 5109 行；只读 r61 增量与相关锚点。未读取工作区 293 行并行改动的全部内容；未核对其是否影响本评审对象。
- 未逐条复核两路 ADR r07 评审报告的 findings 全文；只核路径、字节、SHA256 与 verdict 行。
- 未验证作者报告 §8 的 19 个路径是否与当前工作区实际文件一致；未验证 `tasks.md:3215` 旧证据哈希与新证据哈希的正文差异。
- 本报告是安全侧独立评审结论，不是人的接纳、不是生产批准、不替代 Lead 复核或后续授权；不构成对任何实现、工具、备份、隔离或生产可用性的证明。