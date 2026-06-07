package db

// Canonical audit action identifiers for forensic logs and webhook subscriptions.
const (
	ActionSysStart             = "SYS_START"
	ActionSysConfigUpdate      = "SYS_CONFIG_UPDATE"
	ActionAuthLoginSuccess     = "AUTH_LOGIN_SUCCESS"
	ActionAuthLoginFailed      = "AUTH_LOGIN_FAILED"
	ActionCertIssueNative      = "CERT_ISSUE_NATIVE"
	ActionCertIssueCSR         = "CERT_ISSUE_CSR"
	ActionCertRevoke           = "CERT_REVOKE"
	ActionCertRenew            = "CERT_RENEW"
	ActionEABGenerate          = "EAB_GENERATE"
	ActionEABRevoke            = "EAB_REVOKE"
	ActionSCEPChallengeRotated = "SCEP_CHALLENGE_ROTATED"
	ActionWebhookCreated       = "WEBHOOK_CREATED"
	ActionWebhookDeleted       = "WEBHOOK_DELETED"
	ActionWebhookUpdated       = "WEBHOOK_UPDATED"
	ActionSSHUserCertIssue     = "SSH_USER_CERT_ISSUE"
	ActionSSHHostCertIssue     = "SSH_HOST_CERT_ISSUE"
)
