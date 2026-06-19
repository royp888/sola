-- Fix remaining tables that still reference the old regular-group ID after
-- the group was upgraded to a supergroup (-5103470238 → -1004402373057).
-- Migration 000024 already covered most tables; this covers what was missed.

UPDATE chat_point_configs SET chat_id = -1004402373057 WHERE chat_id = -5103470238;

UPDATE user_points SET chat_id = -1004402373057 WHERE chat_id = -5103470238;

UPDATE point_logs SET chat_id = -1004402373057 WHERE chat_id = -5103470238;

UPDATE scheduled_posts SET chat_id = -1004402373057 WHERE chat_id = -5103470238;

-- telegram_chats uses telegram_chat_id column; also correct the type to supergroup.
UPDATE telegram_chats
    SET telegram_chat_id = -1004402373057,
        type             = 'supergroup'
    WHERE telegram_chat_id = -5103470238;
