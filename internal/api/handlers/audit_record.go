package handlers

import (
	"net/http"

	"github.com/ARCOOON/arx-ca/internal/api/middleware"
)

func auditFromRequest(r *http.Request) *middleware.AuditContext {
	return middleware.AuditFromContext(r.Context())
}

func recordAuditAction(r *http.Request, action string) {
	if ac := auditFromRequest(r); ac != nil {
		ac.SetAction(action)
	}
}

func recordCertAudit(r *http.Request, action, provisioner, certPEM string) {
	ac := auditFromRequest(r)
	if ac == nil {
		return
	}
	ac.SetAction(action)
	if provisioner != "" {
		ac.SetProvisioner(provisioner)
	}
	if certPEM != "" {
		ac.SetFingerprintFromPEM(certPEM)
	}
}
