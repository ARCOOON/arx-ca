package models

// CertificateStatsResponse summarizes X.509 certificate inventory metrics.
type CertificateStatsResponse struct {
	TotalIssued  int `json:"total_issued"`
	Expiring30d  int `json:"expiring_30d"`
	TotalRevoked int `json:"total_revoked"`
}

// SSHStatsResponse summarizes SSH certificate inventory metrics.
type SSHStatsResponse struct {
	TotalUserCerts int `json:"total_user_certs"`
	TotalHostCerts int `json:"total_host_certs"`
	ActiveNow      int `json:"active_now"`
}
