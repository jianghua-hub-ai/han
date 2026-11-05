# R10 作者交接（2026-11-04 12:31Z）

SOURCE / LOCAL STOP。本阶段未授权、未执行任何 PG 或凭据读取；不以此前 PG 结果代表 R10。Main 可立即接手检查和原 Reviewer delta 评审。

G0：固定 Han/main/6da8abbceb0fa197364810a92a8fa2d97b38ddca。已消费 Backend Role、task-implementation、必要规则/profile，实读 C069 最终计划和采用记录 43B7F52CC1200D584E1DF87FA072AC19863CEC0B26F9AB1917D1504EEFA5C8EB。H 前像 C3DFDC66FDBFB025760ED78B0E0061CC49BD7AFB2DA0BF76685D94D0D89B8B99；五文件前像及旧资源哈希在 before/，没有将原实现算作本轮新建。

唯一源码变更 H：directed initial-drain-10000/r2n3b 使用 small 局部副本 seed240/root300，指纹记录默认120、有效240、profile、采用 ref/SHA 和真正 root/seed 开始与截止。正式三档矩阵和 policy 未变。每1000行同步准确 guard、COPY 数据量、业务动作及断言未变；directed 专用 exclusive seed-progress 记录 guard/COPY 返回行数（未提交）、实际 commit、analyze 和退出，失败亦保留。

directed moderation-race/delete-race 专用 outcome 证据：双方实际结果、安全错误类型/分类、Precondition/Unavailable、输入版本/动作映射、同 ID 最终行/operations/audit 单条只读查询、锁图路径/PID/run/schema/context 时点。先确认双方 worker Done，使用原上下文读取；过期、晚结果、读取/JSON/ID/写入错误使测试失败。正式 W 默认不读写该快照。未改另三 Go/docs。

冻结入口：[manifest](source-freeze-r10/manifest.json)、[小差分](r09-to-r10.patch)、[binary](binary.json)。H=1C90DF0DF6726346F7987D11E430C5D2083C0DA3D3A205466A9DD560198AB7C5，250417B。新 nonrace exe=318EBF7A70220B4319C420A1643CFDFE9B1E1BE38122E97E07BBD0118CF52EF0，20086784B，buildinfo 实读 Go1.27.0/windows/amd64/integration，无 race flag；源前后和真实构建结果见 build-nonrace-r10/result.json。

| 实物目录 | 实际结果/范围 |
|---|---|
| red-profile-r01 | exit1，明确120而非批准240的行为失败，非编译失败 |
| green-profile-r01 | exit0，中间版预算修复，不冒最终版 |
| race-final-r01 | 最终同版 race ×3 exit0，R10预算/快照取消、晚结果、读取错、JSON/ID拒绝及R09/R08/R06/相关R05回归 |
| core-final-r10 | 最终 nonrace binary 核心与R10 ×3 exit0，隔离 CWD，不写旧 fixture |
| build-nonrace-r10 / vet-final-r10 | 同版构建/定向 vet 均 exit0 |
| format-final-r10 / buildinfo-r10 | exit0；格式输出为空 / 实读 build flags |

每条 result.json 均含 argv/CWD/UTC/耗时/owned PID/真实 wait exit/sourceBefore/After/raw SHA。最终六命令与日志 SHA、另四源及旧四资源/policy SHA 均由 freeze.py 核一致；git diff --check 针对 H 为 exit0。

final-process-probe.json 无 go.exe 或 *.test.exe；本轮 owned 子命令已全部 wait。历史未知归属 psql PID34364/PPID31972 仍可见，未触碰，不能声称全主机/全 PG 进程为0。本轮从未持有 PG 客户端或 backend，不为交接另查库。

边界：快照 SQL/exclusive 输出在真实 E02/D02 上仍未运行；离线快照边界回归不是双方真实 PG outcome 原物证，也没有倒造 R09 结果。预算 profile 有行为 RED；快照边界回归未另构造旧版行为 RED。SEM-R09-EV001 和 finding 状态由原 Reviewer 独立判断，不作者核销。R10 整组准入12:15:37已过，整组永久 NOT RUN；本 attempt calibration/small/medium 及 large 均 NOT RUN。原 Tnew10:22:37.932710、145/150、资源 origin/global1406932671、旧 small80B、所有旧证据和失败不变。240为有限准备预算，不保证未来 drain 成功。
