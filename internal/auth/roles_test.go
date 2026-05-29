package auth

import "testing"

func TestHasPermission(t *testing.T) {
	if !HasPermission([]Role{RoleCAAdmin}, PermCertificatesIssue) {
		t.Fatal("CA-Admin should issue certificates")
	}
	if HasPermission([]Role{RoleReadOnly}, PermCertificatesIssue) {
		t.Fatal("Read-Only should not issue certificates")
	}
	if !HasPermission([]Role{RoleRevocationManager}, PermCertificatesRevoke) {
		t.Fatal("Revocation-Manager should revoke certificates")
	}
	if !HasPermission([]Role{RoleSuperAdmin}, PermServiceAccounts) {
		t.Fatal("SuperAdmin should manage service accounts")
	}
}

func TestParseRoles(t *testing.T) {
	roles := ParseRoles("SuperAdmin, CA-Admin, invalid")
	if len(roles) != 2 {
		t.Fatalf("expected 2 roles, got %d", len(roles))
	}
}
