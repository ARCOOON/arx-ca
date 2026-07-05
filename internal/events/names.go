package events

// Dot-notation event names for the centralized pub/sub bus.
const (
	EventSystemStarted       = "ca.system.started"
	EventSystemConfigUpdated = "ca.config.updated"
	EventSystemUpdateAvail   = "ca.system.update_available"
	EventSystemUpdateApplied = "ca.system.update_applied"

	EventAuthLoginSuccess = "ca.auth.login.success"
	EventAuthLoginFailed  = "ca.auth.login.failed"
	EventAuthLogout       = "ca.auth.logout"

	EventCertIssuedNative = "ca.cert.issued.native"
	EventCertIssuedCSR    = "ca.cert.issued.csr"
	EventCertIssuedAuto   = "ca.cert.issued.auto"
	EventCertRevoked      = "ca.cert.revoked"
	EventCertRenewed      = "ca.cert.renewed"
	EventCertRekeyed      = "ca.cert.rekeyed"

	EventAuditRecorded = "ca.audit.recorded"

	EventWebhookCreated = "ca.webhook.created"
	EventWebhookUpdated = "ca.webhook.updated"
	EventWebhookDeleted = "ca.webhook.deleted"

	EventSSHCertUserIssued = "ca.ssh.cert.user.issued"
	EventSSHCertHostIssued = "ca.ssh.cert.host.issued"

	EventEABGenerated = "ca.acme.eab.generated"
)
