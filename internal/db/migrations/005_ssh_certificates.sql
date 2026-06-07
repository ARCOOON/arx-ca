CREATE TABLE IF NOT EXISTS ssh_certificates (
	id TEXT PRIMARY KEY,
	serial TEXT NOT NULL,
	cert_type TEXT NOT NULL,
	principals TEXT NOT NULL,
	fingerprint TEXT NOT NULL,
	valid_after TEXT NOT NULL,
	valid_before TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_ssh_certificates_valid_before ON ssh_certificates(valid_before DESC);
CREATE INDEX IF NOT EXISTS idx_ssh_certificates_cert_type ON ssh_certificates(cert_type);
CREATE INDEX IF NOT EXISTS idx_ssh_certificates_serial ON ssh_certificates(serial);
