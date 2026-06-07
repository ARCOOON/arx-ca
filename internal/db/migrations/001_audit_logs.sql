CREATE TABLE IF NOT EXISTS audit_logs (
	id TEXT PRIMARY KEY,
	timestamp TEXT NOT NULL,
	request_id TEXT NOT NULL,
	ip_address TEXT NOT NULL,
	http_method TEXT NOT NULL,
	endpoint TEXT NOT NULL,
	status_code INTEGER NOT NULL,
	actor_type TEXT NOT NULL,
	actor_id TEXT NOT NULL,
	actor_roles TEXT NOT NULL,
	action TEXT NOT NULL,
	provisioner TEXT,
	fingerprint TEXT,
	metadata TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_audit_logs_timestamp ON audit_logs(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_request_id ON audit_logs(request_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action);
