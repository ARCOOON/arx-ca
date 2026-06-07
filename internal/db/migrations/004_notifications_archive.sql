ALTER TABLE notifications ADD COLUMN is_archived INTEGER NOT NULL DEFAULT 0;
CREATE INDEX IF NOT EXISTS idx_notifications_is_archived ON notifications(is_archived);
