DROP INDEX IF EXISTS idx_audit_logs_actor_tg;
DROP INDEX IF EXISTS idx_audit_logs_chat_tg;
DROP INDEX IF EXISTS idx_audit_logs_target_tg;

ALTER TABLE audit_logs
    DROP COLUMN IF EXISTS actor_telegram_id,
    DROP COLUMN IF EXISTS chat_telegram_id,
    DROP COLUMN IF EXISTS target_telegram_id,
    DROP COLUMN IF EXISTS detail;
