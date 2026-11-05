# 运行环境能力探测记录（env-capability-r01）

日期：2026-11-02 22:40 前后（Asia/Shanghai）
HEAD：`6da8abbceb0fa197364810a92a8fa2d97b38ddca`
执行方式：Lead 直接执行只读探测（无 Role 产出、无代码改动、无 Git 写、未建删改任何跟踪文件）
探测动机：`tasks.md` 中 11 张 todo 卡分别受阻于目标环境、真实部署、独立故障域、POSIX 能力与双人保管；本记录把「POSIX 能力是否存在」从 unknown 变为实测事实。

## 1. 实测结果

| 探测项 | 命令 | 结果 |
|---|---|---|
| WSL 发行版 | `wsl.exe -l -v` | `Ubuntu`，State `Running`，Version 2 |
| 内核 | `uname -a` | `Linux DESKTOP-FGO5CAD 6.18.54.1-microsoft-standard-WSL2 x86_64` |
| 当前用户 | `id` | `uid=1000(jianghua)`，`gid=1000`，属组含 `sudo` |
| `/tmp` 文件系统 | `stat -f -c %T /tmp` | `ext2/ext3`（WSL2 虚拟盘上的 ext4，返回通用类型名） |
| 仓库所在盘文件系统 | `stat -f -c %T /mnt/d` | `v9fs`（Windows 盘以 9p 挂载） |
| POSIX 工具 | `command -v flock setpriv capsh getfacl` | `flock`、`setpriv`、`capsh` 存在；`getfacl` 缺失 |
| 宿主 PostgreSQL 5433（Windows 侧） | `Test-NetConnection 127.0.0.1:5433` | 可达（True） |
| 宿主 PostgreSQL 5433（WSL 侧） | `/dev/tcp/127.0.0.1/5433`、`/dev/tcp/172.19.128.1/5433` | 均不可达 |
| WSL 到宿主其他服务 | `/dev/tcp/172.19.128.1/22` | 不可达 |

## 2. 结论

- **存在可用的 POSIX 运行时**：WSL2 + ext4（`/tmp` 所在盘）具备 `flock`、`setpriv`、`capsh`，可作为 POSIX-only 能力的取证环境。
- **仓库当前所在挂载不满足 POSIX 取证前提**：`/mnt/d` 为 `v9fs`，`flock`/`O_CLOEXEC`/`st_dev`/`st_ino` 语义与本地 ext4 不同，POSIX 锁与原子语义不得在 `/mnt/d` 上取证；如需取证，必须在 ext4 工作副本上运行。
- **ACL 取证工具缺失**：`getfacl`/`setfacl` 未安装，ADR-0032 的 ACL/GRANT 矩阵取证前需先安装 `acl` 包（属环境变更，需授权）。
- **WSL 无法访问宿主 PostgreSQL 5433**：集成测试若要在 POSIX 侧运行，需改 Windows 侧 `listen_addresses`/`pg_hba.conf`/防火墙（属配置变更，需授权）；当前不可用。
- **故障域仍不独立**：WSL2 与 Windows 同属一台物理机、同一主机、同一电源与磁盘故障域，不构成独立故障域。

## 3. 不宣称边界

- 不表示 `identity-auth-S4.23`/`S4.24a`/`S4.24b`/`S4.22` 任一已解锁、已实施、可关闭。
- 不表示「独立故障域」成立，不表示生产 `enter`/`install`/`normal`/`recovery`/`requeue` 解锁，不表示生产可用或灾备就绪。
- 不表示任何卡片的 G0 已通过、任何评审已完成、任何人已签收。

## 4. 需要人拍板的四项（含建议值）

| # | 决策 | 建议值 | 解锁范围 |
|---|---|---|---|
| A | 指定 WSL2/ext4 为 POSIX 取证运行时 | 采纳；但只用于能力取证，不作为独立故障域 | 为 `S4.24b` 等 POSIX 实现提供可测环境（仍受 `S4.23` 门禁） |
| B | 第二具名保管人 | 须为真实第二自然人，AI 不可代任 | 解除 ADR-0030 约束 9(b)，是恢复链四卡的唯一结构前提 |
| C | 独立主机/故障域 | 一台与宿主不同物理机或云 VM 的 POSIX 主机 | 解除同主机同故障域限制，支撑 `S4.5` 与恢复演练取证 |
| D | 容量/延迟/RPO/RTO/retention 数值 | 测试项目可给最小合理值（如 RPO≤24h、RTO≤4h、retention 30 天），但须人确认后才写入 | 解锁 `S2.2`、`S4.5`、`S4.6` 的验收判据 |

## 5. 待决项处置（2026-11-02，standing 授权采用建议值）

用户持续授权「对待确认或决策项采用建议值并继续」。处置如下；本节只作记录，未改代码、未改 `tasks.md`、未改 ADR、未做 Git 写。

| # | 处置 | 落地证据 / 限制 |
|---|---|---|
| A 指定 WSL2/ext4 为 POSIX 取证运行时 | **已采纳** | 已在 `~/han-posix-r01`（ext4）完成构建、`vet`、umask 022/077 两轮全量测试并通过，见 `linux-firstrun-r01.md`；仅作能力取证，不据此声明独立故障域 |
| B 第二具名保管人 | **不可由建议值代决，保持 blocked** | 必须是真实第二自然人；AI 代任会实质架空 ADR-0030 约束 9(b)。恢复链四卡（`S4.23`/`S4.24a`/`S4.24b`/`S4.22`）继续 `blocked-upstream` |
| C 独立主机/故障域 | **不可由建议值代决，保持 blocked** | 需一台与宿主不同物理机或云 VM；WSL2 与 Windows 同物理机、同电源与磁盘，不满足；`S4.5` 与恢复演练取证继续 NOT RUN |
| D 容量/延迟/RPO/RTO/retention 数值 | **已采纳为测试项目暂定值** | `RPO ≤ 24h`、`RTO ≤ 4h`、`retention 30 天`；仅作验收判据的暂定输入，属测试项目尺度，不得用于生产容量或 SLA 声明；`S2.2` 的目标部署环境仍缺，故该卡仍 `blocked-for-execution` |

**E（Linux `-race`，已闭环）**：2026-11-02 探测 WSL 侧 `gcc`/`cc`/`clang` 均 `MISSING`（仅 `ld` 存在），`sudo` 又需密码，无法安装系统包。改用**无 root 路径**：`apt-get download` 取 `gcc-11`、`cpp-11`、`libgcc-11-dev`、`libc6-dev` 等 13 个 deb，`dpkg-deb -x` 解到 `~/tc/root`，重写该树内的 `libc.so` 链接脚本指向解包路径，再以 `CC`/`CGO_CFLAGS`/`CGO_LDFLAGS`/`LD_LIBRARY_PATH` 指向该树。
- 结果：`go test -race ./... -count=1` 于 23:08–23:10 在 `~/han-posix-r01` 执行，**exit 0**，12 个包 `ok`、2 个无测试用例（`cmd/han-server` 116.5s、`internal/recovery` 9.3s、`internal/mailer` 7.0s），无 data race 报告。2026-11-03 用还原后的 `go.mod`（`GOFLAGS=-mod=readonly`）复跑，`cmd/han-server` 132.4s，结果一致。
- 边界：工具链完全位于 WSL 用户主目录，未 `apt install`、未改系统包库与 `PATH`；不是“已安装 gcc”的系统变更，换机器或重装 WSL 即失效。
- 仍未运行：迁移、真实 PG 集成、E2E 旅程、恢复演练。
- 更新（2026-11-03）：真实 PG 集成已首次执行，11 个包通过、6 个包失败，失败分为“迁移漂移 / 缺专用库角色 / 缺证据目录 / 未定位超时”四类，详见 `integration-run-2026-11-03.md`。

本文件位于 `.spec/wip`，属临时交接物，按清理清单确认流程处置，不自动删除。
