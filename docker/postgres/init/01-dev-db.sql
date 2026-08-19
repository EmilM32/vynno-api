-- Runs only on a brand-new volume. ensure_postgres also creates vynno_dev
-- for machines that already have pgdata.
CREATE DATABASE vynno_dev OWNER vynno;
