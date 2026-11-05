# Probe-error R06 verdict

**PASS，仅限固定R06代码delta的有界静态Gate。PROBE-R05-F001 CLOSED，无新增confirmed finding。**

objectType=code；Han/main/HEAD6da8abbceb0fa197364810a92a8fa2d97b38ddca。H **214370B / 550B16614224A13F31CEE931C016C3B6841F59E80F27CE2A61F24720219392E7**；五snapshot/before与旧资源摘要已自行核，8个patch hunk重建一致。

Round1核窄修复及新测试；Round2核真实reader两调用点→helper→classifier/collector Closing→诊断/failedOwn→closeResources。noncontext/mixed不再被ctx取消覆盖，安全类型保留SQLSTATE/type但不泄漏原文，纯context关闭不误报，晚成功不发布。原F001实现/回归关闭条件满足。

9条result/raw独立核；5条最终同五hash且exit0。普通16pass、定向race count3合计87pass，均0fail/skip；另build/vet/format通过。新R06 binary **99F5004989B701191EEB2AF0D363410026A993A93A544442D796BFE792958A5D**，构建/路径/源码及新sandbox链已核；只读入口body与R05相同，依赖已变，未来不得用旧binary替代。

中间测试越界向R05新增6目录/30文件实际保留、摘要匹配，未清理/回滚。旧已固定manifest/快照/raw/Review/资源及关联证据摘要未变，最终回归在R06隔离；无事前全树inventory，不能无限保证所有未pin旧文件。事件留档由Main现场接收，不改判为新源码HIGH。

**未Go/LOCAL/PG/凭据/Git/清理/源或旧物证改写/委派。P0真实PG/growth/leak/资源归还及原12pending未过；Tnew未产生，旧完整性能NRU、large/canonical NRU、任务active保持。** 本PASS不是PG/性能/整卡/E2E PASS或下一运行授权。

详见[report.md](report.md)、evidence-audit.json及artifact-sha256.json。完成 **STOP**。
