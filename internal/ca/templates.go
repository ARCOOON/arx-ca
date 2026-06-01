package ca

import (
	"bytes"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/smallstep/certificates/authority/provisioner"

	"github.com/your-org/arx-ca/internal/models"
)

const templateRegistryFile = "registry.json"

// templateApplyResult is the JSON document rendered by a certificate template body.
type templateApplyResult struct {
	DNSSANs    []string            `json:"dns_sans"`
	IPSANs     []string            `json:"ip_sans"`
	EmailSANs  []string            `json:"email_sans"`
	URISANs    []string            `json:"uri_sans"`
	Extensions []templateExtension `json:"extensions"`
}

type templateExtension struct {
	OID           string `json:"oid"`
	Critical      bool   `json:"critical"`
	Value         string `json:"value"`
	ValueEncoding string `json:"value_encoding"`
}

type templateRenderContext struct {
	Metadata   map[string]any
	CommonName string
	CSR        *csrTemplateView
}

type csrTemplateView struct {
	Subject        string   `json:"subject"`
	CommonName     string   `json:"common_name"`
	DNSNames       []string `json:"dns_names"`
	IPAddresses    []string `json:"ip_addresses"`
	EmailAddresses []string `json:"email_addresses"`
	URIs           []string `json:"uris"`
}

// TemplateStore persists certificate issuance templates under the PKI base path.
type TemplateStore struct {
	mu       sync.RWMutex
	basePath string
	byID     map[string]models.CertificateTemplate
}

func newTemplateStore(basePath string) (*TemplateStore, error) {
	store := &TemplateStore{
		basePath: filepath.Join(basePath, "templates"),
		byID:     make(map[string]models.CertificateTemplate),
	}
	if err := os.MkdirAll(store.basePath, 0o700); err != nil {
		return nil, fmt.Errorf("create templates directory: %w", err)
	}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *TemplateStore) registryPath() string {
	return filepath.Join(s.basePath, templateRegistryFile)
}

func (s *TemplateStore) load() error {
	data, err := os.ReadFile(s.registryPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read template registry: %w", err)
	}

	var templates []models.CertificateTemplate
	if err := json.Unmarshal(data, &templates); err != nil {
		return fmt.Errorf("parse template registry: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.byID = make(map[string]models.CertificateTemplate, len(templates))
	for _, tpl := range templates {
		s.byID[tpl.ID] = tpl
	}
	return nil
}

func (s *TemplateStore) persistLocked() error {
	templates := make([]models.CertificateTemplate, 0, len(s.byID))
	for _, tpl := range s.byID {
		templates = append(templates, tpl)
	}
	data, err := json.MarshalIndent(templates, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal template registry: %w", err)
	}
	tmp := s.registryPath() + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write template registry: %w", err)
	}
	return os.Rename(tmp, s.registryPath())
}

// CreateTemplate registers a new issuance template.
func (s *TemplateStore) CreateTemplate(req models.CreateCertificateTemplateRequest) (*models.CertificateTemplate, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, errors.New("name is required")
	}
	body := strings.TrimSpace(req.Body)
	if body == "" {
		return nil, errors.New("body is required")
	}
	if err := validateTemplateBody(body); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	tpl := models.CertificateTemplate{
		ID:          uuid.NewString(),
		Name:        name,
		Description: strings.TrimSpace(req.Description),
		Body:        body,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.byID {
		if strings.EqualFold(existing.Name, name) {
			return nil, fmt.Errorf("template name %q already exists", name)
		}
	}
	s.byID[tpl.ID] = tpl
	if err := s.persistLocked(); err != nil {
		delete(s.byID, tpl.ID)
		return nil, err
	}
	return &tpl, nil
}

// ListTemplates returns all registered templates.
func (s *TemplateStore) ListTemplates() (*models.ListCertificateTemplatesResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	templates := make([]models.CertificateTemplate, 0, len(s.byID))
	for _, tpl := range s.byID {
		templates = append(templates, tpl)
	}
	return &models.ListCertificateTemplatesResponse{
		Templates: templates,
		Total:     len(templates),
	}, nil
}

// GetTemplate returns a template by identifier.
func (s *TemplateStore) GetTemplate(id string) (*models.CertificateTemplate, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, errors.New("template_id is required")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	tpl, ok := s.byID[id]
	if !ok {
		return nil, fmt.Errorf("template %q not found", id)
	}
	out := tpl
	return &out, nil
}

func validateTemplateBody(body string) error {
	if _, err := template.New("validate").Parse(body); err != nil {
		return fmt.Errorf("invalid template body: %w", err)
	}
	return nil
}

func validateTemplateOutput(raw []byte) error {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return errors.New("template body must render non-empty JSON")
	}
	var result templateApplyResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return fmt.Errorf("template body must render JSON: %w", err)
	}
	for _, ext := range result.Extensions {
		if strings.TrimSpace(ext.OID) == "" {
			return errors.New("extension oid is required")
		}
		if strings.TrimSpace(ext.Value) == "" {
			return errors.New("extension value is required")
		}
	}
	return nil
}

func renderTemplate(tpl *models.CertificateTemplate, metadata map[string]any, csr *x509.CertificateRequest, commonName string) (*templateApplyResult, error) {
	tmpl, err := template.New(tpl.ID).Parse(tpl.Body)
	if err != nil {
		return nil, fmt.Errorf("parse template: %w", err)
	}

	if metadata == nil {
		metadata = map[string]any{}
	}

	ctx := templateRenderContext{
		Metadata:   metadata,
		CommonName: commonName,
	}
	if csr != nil {
		ctx.CSR = csrToTemplateView(csr)
		if ctx.CommonName == "" {
			ctx.CommonName = csr.Subject.CommonName
		}
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, ctx); err != nil {
		return nil, fmt.Errorf("execute template: %w", err)
	}
	if err := validateTemplateOutput(buf.Bytes()); err != nil {
		return nil, err
	}

	var result templateApplyResult
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("parse template output: %w", err)
	}
	return &result, nil
}

func csrToTemplateView(csr *x509.CertificateRequest) *csrTemplateView {
	view := &csrTemplateView{
		Subject:        csr.Subject.String(),
		CommonName:     csr.Subject.CommonName,
		DNSNames:       append([]string(nil), csr.DNSNames...),
		EmailAddresses: append([]string(nil), csr.EmailAddresses...),
	}
	for _, ip := range csr.IPAddresses {
		view.IPAddresses = append(view.IPAddresses, ip.String())
	}
	for _, u := range csr.URIs {
		if u != nil {
			view.URIs = append(view.URIs, u.String())
		}
	}
	return view
}

func (e *PKIEngine) templateSignOptions(templateID string, metadata map[string]any, csr *x509.CertificateRequest, commonName string) ([]provisioner.SignOption, error) {
	if e == nil || e.templates == nil {
		return nil, errors.New("template store is not initialized")
	}
	templateID = strings.TrimSpace(templateID)
	if templateID == "" {
		return nil, nil
	}

	tpl, err := e.templates.GetTemplate(templateID)
	if err != nil {
		return nil, err
	}

	result, err := renderTemplate(tpl, metadata, csr, commonName)
	if err != nil {
		return nil, err
	}

	modifier, err := certificateModifierFromResult(result)
	if err != nil {
		return nil, err
	}
	return []provisioner.SignOption{modifier}, nil
}

func certificateModifierFromResult(result *templateApplyResult) (provisioner.SignOption, error) {
	dnsNames := uniqueStrings(result.DNSSANs)
	ips, err := parseIPAddresses(result.IPSANs)
	if err != nil {
		return nil, err
	}
	emails := uniqueStrings(result.EmailSANs)
	uris, err := parseURIs(result.URISANs)
	if err != nil {
		return nil, err
	}
	extensions, err := parseTemplateExtensions(result.Extensions)
	if err != nil {
		return nil, err
	}

	return provisioner.CertificateModifierFunc(func(cert *x509.Certificate, _ provisioner.SignOptions) error {
		cert.DNSNames = mergeStrings(cert.DNSNames, dnsNames)
		cert.IPAddresses = mergeIPs(cert.IPAddresses, ips)
		cert.EmailAddresses = mergeStrings(cert.EmailAddresses, emails)
		cert.URIs = mergeURIs(cert.URIs, uris)
		cert.ExtraExtensions = append(cert.ExtraExtensions, extensions...)
		return nil
	}), nil
}

func parseTemplateExtensions(exts []templateExtension) ([]pkix.Extension, error) {
	out := make([]pkix.Extension, 0, len(exts))
	for _, ext := range exts {
		oid, err := parseOID(ext.OID)
		if err != nil {
			return nil, err
		}
		value, err := decodeExtensionValue(ext.Value, ext.ValueEncoding)
		if err != nil {
			return nil, fmt.Errorf("extension %s: %w", ext.OID, err)
		}
		out = append(out, pkix.Extension{
			Id:       oid,
			Critical: ext.Critical,
			Value:    value,
		})
	}
	return out, nil
}

func parseOID(raw string) (asn1.ObjectIdentifier, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, errors.New("oid is required")
	}
	parts := strings.Split(raw, ".")
	oid := make(asn1.ObjectIdentifier, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, fmt.Errorf("invalid oid %q", raw)
		}
		var n int
		if _, err := fmt.Sscanf(part, "%d", &n); err != nil || n < 0 {
			return nil, fmt.Errorf("invalid oid component %q in %q", part, raw)
		}
		oid = append(oid, n)
	}
	return oid, nil
}

func decodeExtensionValue(value, encoding string) ([]byte, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, errors.New("value is required")
	}
	switch strings.ToLower(strings.TrimSpace(encoding)) {
	case "", "base64":
		return base64.StdEncoding.DecodeString(value)
	case "hex":
		return hex.DecodeString(value)
	case "utf8", "text":
		return []byte(value), nil
	default:
		return nil, fmt.Errorf("unsupported value_encoding %q", encoding)
	}
}

func parseIPAddresses(values []string) ([]net.IP, error) {
	out := make([]net.IP, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		ip := net.ParseIP(value)
		if ip == nil {
			return nil, fmt.Errorf("invalid ip_sans entry %q", value)
		}
		out = append(out, ip)
	}
	return out, nil
}

func parseURIs(values []string) ([]*url.URL, error) {
	out := make([]*url.URL, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		u, err := url.Parse(value)
		if err != nil {
			return nil, fmt.Errorf("invalid uri_sans entry %q: %w", value, err)
		}
		out = append(out, u)
	}
	return out, nil
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func mergeStrings(existing, extra []string) []string {
	if len(extra) == 0 {
		return existing
	}
	seen := make(map[string]struct{}, len(existing)+len(extra))
	out := make([]string, 0, len(existing)+len(extra))
	for _, value := range existing {
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	for _, value := range extra {
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func mergeIPs(existing, extra []net.IP) []net.IP {
	if len(extra) == 0 {
		return existing
	}
	seen := make(map[string]struct{}, len(existing)+len(extra))
	out := make([]net.IP, 0, len(existing)+len(extra))
	for _, ip := range existing {
		key := ip.String()
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, ip)
	}
	for _, ip := range extra {
		key := ip.String()
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, ip)
	}
	return out
}

func mergeURIs(existing, extra []*url.URL) []*url.URL {
	if len(extra) == 0 {
		return existing
	}
	seen := make(map[string]struct{}, len(existing)+len(extra))
	out := make([]*url.URL, 0, len(existing)+len(extra))
	for _, u := range existing {
		if u == nil {
			continue
		}
		key := u.String()
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, u)
	}
	for _, u := range extra {
		if u == nil {
			continue
		}
		key := u.String()
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, u)
	}
	return out
}

// CreateCertificateTemplate registers a template via the engine.
func (e *PKIEngine) CreateCertificateTemplate(req models.CreateCertificateTemplateRequest) (*models.CertificateTemplate, error) {
	if e == nil || e.templates == nil {
		return nil, errors.New("template store is not initialized")
	}
	return e.templates.CreateTemplate(req)
}

// ListCertificateTemplates returns all templates via the engine.
func (e *PKIEngine) ListCertificateTemplates() (*models.ListCertificateTemplatesResponse, error) {
	if e == nil || e.templates == nil {
		return nil, errors.New("template store is not initialized")
	}
	return e.templates.ListTemplates()
}
