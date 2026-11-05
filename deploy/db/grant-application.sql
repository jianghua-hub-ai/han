-- 迁移后授权（以迁移身份执行）。
--
-- 必填 psql 变量：
--   -v schema_name=public
--   -v application_role=han_app
--
-- 顺序不可颠倒：先授予业务 DML，再对 append-only 表撤销变更与清空权限。
-- TRUNCATE 不在 0001_shared.sql 的 trigger 覆盖范围内，只能靠 ACL 拒绝。
-- 本脚本的 append-only 撤销列表必须与 cmd/han-server/append_only_acl.go 的
-- appendOnlyMutationPrivileges 常量保持同步。服务启动迁移后会自动执行等价
-- REVOKE；本脚本用于 out-of-band 迁移后的重放或人工复核。
-- 表级 REVOKE 不会移除列级授权；gate 会拒绝 append-only 表上的任何列级
-- UPDATE/DELETE ACL，必须由授权方显式撤销。

\set ON_ERROR_STOP on

GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA :"schema_name" TO :"application_role";
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA :"schema_name" TO :"application_role";
REVOKE ALL ON TABLE :"schema_name".schema_migrations FROM :"application_role";
REVOKE UPDATE, DELETE, TRUNCATE ON TABLE :"schema_name".audit_events FROM :"application_role";
REVOKE UPDATE, DELETE, TRUNCATE ON TABLE :"schema_name".processing_gate_records FROM :"application_role";
