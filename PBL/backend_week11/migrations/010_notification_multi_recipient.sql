ALTER TABLE notifications
    DROP CONSTRAINT IF EXISTS notifications_outbox_unique;

ALTER TABLE notifications
    ADD CONSTRAINT notifications_outbox_user_unique UNIQUE (outbox_id, user_id);