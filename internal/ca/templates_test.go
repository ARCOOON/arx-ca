package ca

import (
	"crypto/x509"
	"encoding/json"
	"testing"

	"github.com/smallstep/certificates/authority/provisioner"

	"github.com/your-org/arx-ca/internal/models"
)

func TestRenderTemplateAddsSANs(t *testing.T) {
	body := `{"dns_sans":["{{.Metadata.host}}.example.com"],"ip_sans":[],"email_sans":[],"uri_sans":[],"extensions":[]}`
	tpl := &models.CertificateTemplate{ID: "test", Body: body}

	result, err := renderTemplate(tpl, map[string]any{"host": "app"}, nil, "app.example.com")
	if err != nil {
		t.Fatalf("renderTemplate: %v", err)
	}
	if len(result.DNSSANs) != 1 || result.DNSSANs[0] != "app.example.com" {
		t.Fatalf("unexpected dns sans: %#v", result.DNSSANs)
	}

	modifier, err := certificateModifierFromResult(result)
	if err != nil {
		t.Fatalf("certificateModifierFromResult: %v", err)
	}

	cert := &x509.Certificate{}
	mod, ok := modifier.(interface {
		Modify(*x509.Certificate, provisioner.SignOptions) error
	})
	if !ok {
		t.Fatal("modifier does not implement Modify")
	}
	if err := mod.Modify(cert, provisioner.SignOptions{}); err != nil {
		t.Fatalf("Modify: %v", err)
	}
	if len(cert.DNSNames) != 1 || cert.DNSNames[0] != "app.example.com" {
		t.Fatalf("unexpected cert DNS names: %#v", cert.DNSNames)
	}
}

func TestValidateTemplateOutput(t *testing.T) {
	raw, err := json.Marshal(templateApplyResult{DNSSANs: []string{"a.example.com"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := validateTemplateOutput(raw); err != nil {
		t.Fatalf("validateTemplateOutput: %v", err)
	}
}
