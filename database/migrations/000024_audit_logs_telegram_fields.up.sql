-- Add Telegram-native ID columns that were in the model but missing from the schema.
ALTER TABLE audit_logs
    ADD COLUMN IF NOT EXISTS actor_telegram_id BIGINT,
    ADD COLUMN IF NOT EXISTS chat_telegram_id   BIGINT,
    ADD COLUMN IF NOT EXISTS target_telegram_id BIGINT,
    ADD COLUMN IF NOT EXISTS detail             TEXT;

CREATE INDEX IF NOT EXISTS idx_audit_logs_actor_tg   ON audit_logs (actor_telegram_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_chat_tg    ON audit_logs (chat_telegram_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_target_tg  ON audit_logs (target_telegram_id);

-- Migrate old regular-group ID to the supergroup ID it was upgraded to.
-- This covers lotteries and any other chat_id (bigint) columns still referencing -5103470238.
UPDATE lotteries              SET chat_id = -1004402373057 WHERE chat_id = -5103470238;
UPDATE ban_logs               SET chat_id = -1004402373057 WHERE chat_id = -5103470238;
UPDATE violation_records      SET chat_id = -1004402373057 WHERE chat_id = -5103470238;
UPDATE seen_users             SET chat_id = -1004402373057 WHERE chat_id = -5103470238;
UPDATE invite_links           SET chat_id = -1004402373057 WHERE chat_id = -5103470238;
UPDATE warn_records           SET chat_id = -1004402373057 WHERE chat_id = -5103470238;
UPDATE auto_replies           SET chat_id = -1004402373057 WHERE chat_id = -5103470238;
UPDATE message_templates      SET chat_id = -1004402373057 WHERE chat_id = -5103470238;
UPDATE level_configs          SET chat_id = -1004402373057 WHERE chat_id = -5103470238;

-- For the configs that already have a row for the new supergroup ID,
-- only update the old-ID row if the new-ID row does NOT exist yet.
UPDATE chat_admin_configs SET chat_id = -1004402373057
    WHERE chat_id = -5103470238
      AND NOT EXISTS (SELECT 1 FROM chat_admin_configs WHERE chat_id = -1004402373057);

UPDATE keyword_filters SET chat_id = -1004402373057
    WHERE chat_id = -5103470238
      AND NOT EXISTS (SELECT 1 FROM keyword_filters WHERE chat_id = -1004402373057);

UPDATE chat_moderation_configs SET chat_id = -1004402373057
    WHERE chat_id = -5103470238
      AND NOT EXISTS (SELECT 1 FROM chat_moderation_configs WHERE chat_id = -1004402373057);
