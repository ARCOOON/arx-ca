CREATE TABLE IF NOT EXISTS notifications (
	id TEXT PRIMARY KEY,
	timestamp TEXT NOT NULL,
	action TEXT NOT NULL,
	level TEXT NOT NULL DEFAULT 'info',
	message TEXT NOT NULL,
	is_read INTEGER NOT NULL DEFAULT 0,
	metadata TEXT NOT NULL DEFAULT '{}'
);
CREATE INDEX IF NOT EXISTS idx_notifications_timestamp ON notifications(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_notifications_is_read ON notifications(is_read);
