-- Роли приложения. Создаются один раз при инициализации тома Postgres.
-- pms_app — рабочая роль с ПРИНУДИТЕЛЬНЫМ RLS (не владелец таблиц → политики применяются).
-- pms_jobs — фоновые джобы с обходом RLS (purge, кросс-тенантные операции).
-- Миграции выполняются ролью postgres (владелец таблиц); приложение коннектится как pms_app.

CREATE ROLE pms_app  LOGIN PASSWORD 'pms_app';
CREATE ROLE pms_jobs LOGIN PASSWORD 'pms_jobs' BYPASSRLS;

GRANT CONNECT ON DATABASE pms TO pms_app, pms_jobs;
GRANT USAGE  ON SCHEMA public TO pms_app, pms_jobs;

-- Привилегии на будущие объекты схемы (таблицы/последовательности создаются миграциями от postgres).
ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA public
  GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO pms_app, pms_jobs;
ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA public
  GRANT USAGE, SELECT ON SEQUENCES TO pms_app, pms_jobs;
