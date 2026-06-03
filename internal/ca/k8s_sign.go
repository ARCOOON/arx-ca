package ca

import (
	"context"
	"crypto/x509"
	"fmt"
	"strings"

	"github.com/ARCOOON/arx-ca/internal/models"
)

// prepareEnrollmentToken validates Kubernetes service account tokens when configured
// and bridges them to a JWK provisioner token when the K8sSA provisioner is not present.
func (e *PKIEngine) prepareEnrollmentToken(ctx context.Context, csr *x509.CertificateRequest, token string) (string, error) {
	if e == nil || e.k8sReviewer == nil || !isKubernetesServiceAccountToken(token) {
		return token, nil
	}

	identity, err := e.k8sReviewer.Review(ctx, token)
	if err != nil {
		return "", fmt.Errorf("kubernetes service account authentication failed: %w", err)
	}

	if e.hasK8sSAProvisioner() {
		return token, nil
	}

	cn := csr.Subject.CommonName
	if cn == "" {
		cn = identity.ServiceAccountName
	}
	sans := collectCSRSubjectAlternativeNames(csr)
	if len(sans) == 0 && cn != "" {
		sans = []string{cn}
	}

	bridged, _, _, err := e.createProvisionerSignToken(defaultProvisioner, cn, sans, defaultTokenTTL)
	if err != nil {
		return "", fmt.Errorf("bridge kubernetes token to provisioner token: %w", err)
	}
	return bridged, nil
}

func (e *PKIEngine) hasK8sSAProvisioner() bool {
	if e == nil || e.auth == nil || !e.appConfig.K8s.Enabled {
		return false
	}
	name := e.appConfig.K8s.Provisioner
	if name == "" {
		name = defaultK8sProvisionerName
	}
	_, err := e.loadProvisionerByName(name)
	return err == nil
}

// K8sProvisionerStatus exposes Kubernetes provisioner configuration for operators.
func (e *PKIEngine) K8sProvisionerStatus() models.K8sProvisionerStatusResponse {
	if e == nil || !e.appConfig.K8s.Enabled {
		return models.K8sProvisionerStatusResponse{Enabled: false}
	}
	name := strings.TrimSpace(e.appConfig.K8s.Provisioner)
	if name == "" {
		name = defaultK8sProvisionerName
	}
	mode := string(e.appConfig.K8s.ReviewMode)
	if mode == "" {
		mode = "auto"
	}
	return models.K8sProvisionerStatusResponse{
		Enabled:     true,
		Provisioner: name,
		ReviewMode:  mode,
		HasPubKeys:  len(e.appConfig.K8s.PublicKeysPEM) > 0,
		UsesAPI:     e.appConfig.K8s.UsesTokenReviewAPI(),
	}
}
