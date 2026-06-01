package auth

import (
	"os"
	"strings"
)

// Role identifies an RBAC role assigned to admin users and service accounts.
type Role string

const (
	RoleSuperAdmin        Role = "SuperAdmin"
	RoleCAAdmin           Role = "CA-Admin"
	RoleRevocationManager Role = "Revocation-Manager"
	RoleReadOnly          Role = "Read-Only"
)

// Permission names an API capability enforced by RBAC middleware.
type Permission string

const (
	PermCertificatesIssue  Permission = "certificates:issue"
	PermCertificatesRevoke Permission = "certificates:revoke"
	PermCertificatesRead   Permission = "certificates:read"
	PermCertificatesLint   Permission = "certificates:lint"
	PermCertificatesRenew  Permission = "certificates:renew"
	PermProvisionersToken  Permission = "provisioners:token"
	PermTemplatesManage    Permission = "templates:manage"
	PermTemplatesRead      Permission = "templates:read"
	PermServiceAccounts    Permission = "service_accounts:manage"
	PermACMEEAB            Permission = "acme:eab"
	PermEnrollmentStatus   Permission = "enrollment:status"
	PermSSHSignHost        Permission = "ssh:sign_host"
	PermSSHInspect         Permission = "ssh:inspect"
)

var rolePermissions = map[Role][]Permission{
	RoleSuperAdmin: {
		PermCertificatesIssue,
		PermCertificatesRevoke,
		PermCertificatesRead,
		PermCertificatesLint,
		PermCertificatesRenew,
		PermProvisionersToken,
		PermTemplatesManage,
		PermTemplatesRead,
		PermServiceAccounts,
		PermACMEEAB,
		PermEnrollmentStatus,
		PermSSHSignHost,
		PermSSHInspect,
	},
	RoleCAAdmin: {
		PermCertificatesIssue,
		PermCertificatesRead,
		PermCertificatesLint,
		PermCertificatesRenew,
		PermProvisionersToken,
		PermTemplatesManage,
		PermTemplatesRead,
		PermACMEEAB,
		PermEnrollmentStatus,
		PermSSHSignHost,
		PermSSHInspect,
	},
	RoleRevocationManager: {
		PermCertificatesRevoke,
		PermCertificatesRead,
		PermCertificatesLint,
		PermEnrollmentStatus,
	},
	RoleReadOnly: {
		PermCertificatesRead,
		PermTemplatesRead,
		PermEnrollmentStatus,
	},
}

// ValidRole reports whether role is a known RBAC role.
func ValidRole(role Role) bool {
	_, ok := rolePermissions[role]
	return ok
}

// ParseRoles splits a comma-separated role list and returns only valid roles.
func ParseRoles(raw string) []Role {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]Role, 0, len(parts))
	seen := make(map[Role]struct{})
	for _, part := range parts {
		role := Role(strings.TrimSpace(part))
		if !ValidRole(role) {
			continue
		}
		if _, exists := seen[role]; exists {
			continue
		}
		seen[role] = struct{}{}
		out = append(out, role)
	}
	return out
}

// RolesForAdmin returns roles assigned to the given admin email when not stored in the database.
func RolesForAdmin(email string) []Role {
	_ = email
	if roles := ParseRoles(os.Getenv("CA_API_ADMIN_ROLES")); len(roles) > 0 {
		return roles
	}
	if roles := ParseRoles(os.Getenv("CA_API_BOOTSTRAP_ROLES")); len(roles) > 0 {
		return roles
	}
	return []Role{RoleSuperAdmin}
}

// DefaultServiceAccountRoles returns roles for newly created service accounts.
func DefaultServiceAccountRoles() []Role {
	if roles := ParseRoles(os.Getenv("CA_API_SERVICE_ACCOUNT_ROLES")); len(roles) > 0 {
		return roles
	}
	return []Role{RoleCAAdmin}
}

// HasPermission reports whether any of the given roles grants permission.
func HasPermission(roles []Role, perm Permission) bool {
	for _, role := range roles {
		for _, p := range rolePermissions[role] {
			if p == perm {
				return true
			}
		}
	}
	return false
}

// NormalizeRoles deduplicates and drops unknown roles.
func NormalizeRoles(roles []Role) []Role {
	if len(roles) == 0 {
		return nil
	}
	seen := make(map[Role]struct{}, len(roles))
	out := make([]Role, 0, len(roles))
	for _, role := range roles {
		if !ValidRole(role) {
			continue
		}
		if _, exists := seen[role]; exists {
			continue
		}
		seen[role] = struct{}{}
		out = append(out, role)
	}
	return out
}
