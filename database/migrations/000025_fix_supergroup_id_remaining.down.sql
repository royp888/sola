-- Reverse: restore old regular-group ID (data loss if new rows were inserted).
UPDATE chat_point_configs SET chat_id = -5103470238 WHERE chat_id = -1004402373057;
UPDATE user_points         SET chat_id = -5103470238 WHERE chat_id = -1004402373057;
UPDATE point_logs          SET chat_id = -5103470238 WHERE chat_id = -1004402373057;
UPDATE scheduled_posts     SET chat_id = -5103470238 WHERE chat_id = -1004402373057;
UPDATE telegram_chats
    SET telegram_chat_id = -5103470238,
        type             = 'group'
    WHERE telegram_chat_id = -1004402373057;
