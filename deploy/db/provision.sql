-- 数据库身份与 schema 归属（以集群管理员身份执行一次）。
--
-- 必填 psql 变量（从部署密钥管理系统注入，不写入仓库、不写入 shell 历史）：
--   -v database_name=han
--   -v schema_name=public
--   -v migrator_role=han_migrator
--   -v migrator_password=...
--   -v application_role=han_app
--   -v application_password=...
--
-- 迁移身份与运行身份必须是两个不同的 LOGIN 角色：
--   * 迁移身份持有 schema 与对象所有权，只在启动迁移步骤使用；
--   * 运行身份只有业务 DML 权限，不得拥有 schema 或表。

\set ON_ERROR_STOP on

-- psql 不会在 dollar-quoted 块（DO $$ ... $$）内替换 :'var'，因此角色创建用
-- SELECT + \gexec：查询生成 DDL 文本，再由 psql 逐行执行。
SELECT format('CREATE ROLE %I', :'migrator_role')
WHERE NOT EXISTS (
	SELECT 1 FROM pg_catalog.pg_roles WHERE rolname = :'migrator_role'
);
\gexec

SELECT format('CREATE ROLE %I', :'application_role')
WHERE NOT EXISTS (
	SELECT 1 FROM pg_catalog.pg_roles WHERE rolname = :'application_role'
);
\gexec

ALTER ROLE :"migrator_role" WITH LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE PASSWORD :'migrator_password';
ALTER ROLE :"application_role" WITH LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE PASSWORD :'application_password';

GRANT CONNECT ON DATABASE :"database_name" TO :"migrator_role";
GRANT CONNECT ON DATABASE :"database_name" TO :"application_role";

CREATE SCHEMA IF NOT EXISTS :"schema_name" AUTHORIZATION :"migrator_role";
ALTER SCHEMA :"schema_name" OWNER TO :"migrator_role";
REVOKE CREATE ON SCHEMA :"schema_name" FROM PUBLIC;
REVOKE ALL ON SCHEMA :"schema_name" FROM :"application_role";
GRANT USAGE ON SCHEMA :"schema_name" TO :"application_role";

-- 迁移身份新建的业务对象默认授予运行身份 DML。服务启动迁移完成后会在
-- 关闭迁移池之前，用迁移身份对 append-only 表执行与 grant-application.sql
-- 等价的 REVOKE；因此 fresh install 不需要先手工执行 grant-application.sql。
ALTER DEFAULT PRIVILEGES FOR ROLE :"migrator_role" IN SCHEMA :"schema_name"
	GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO :"application_role";
ALTER DEFAULT PRIVILEGES FOR ROLE :"migrator_role" IN SCHEMA :"schema_name"
	GRANT USAGE, SELECT ON SEQUENCES TO :"application_role";
