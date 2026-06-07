CREATE TABLE IF NOT EXISTS webhooks (
	id TEXT PRIMARY KEY,
	url TEXT NOT NULL,
	name TEXT NOT NULL,
	secret_token TEXT,
	active INTEGER NOT NULL DEFAULT 1,
	subscribed_events TEXT NOT NULL,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_webhooks_active ON webhooks(active);
