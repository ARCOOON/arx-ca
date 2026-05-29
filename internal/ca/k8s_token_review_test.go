package ca

import (
	"testing"

	"github.com/your-org/arx-ca/internal/config"
)

func TestParseK8sSubject(t *testing.T) {
	parsed, err := parseK8sSubject("system:serviceaccount:prod:api")
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Namespace != "prod" || parsed.ServiceAccountName != "api" {
		t.Fatalf("unexpected parse: %+v", parsed)
	}
}

func TestK8sTokenReviewerLocalRequiresKeys(t *testing.T) {
	r, err := NewK8sTokenReviewer(config.K8sConfig{
		Enabled:    true,
		ReviewMode: config.K8sReviewLocal,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Review(t.Context(), "not-a-jwt"); err == nil {
		t.Fatal("expected error for invalid token")
	}
}
