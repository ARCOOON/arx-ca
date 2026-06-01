package config

import (
	"os"
	"strings"
)

// K8sTokenReviewMode selects how Kubernetes service account tokens are validated.
type K8sTokenReviewMode string

const (
	// K8sReviewLocal validates JWT signatures using configured cluster public keys (no API call).
	K8sReviewLocal K8sTokenReviewMode = "local"
	// K8sReviewAPI calls the Kubernetes TokenReview API.
	K8sReviewAPI K8sTokenReviewMode = "api"
	// K8sReviewAuto uses the API when configured, otherwise local public keys.
	K8sReviewAuto K8sTokenReviewMode = "auto"
)

// K8sConfig holds Kubernetes Service Account (KSA) provisioner settings.
type K8sConfig struct {
	Enabled       bool
	Provisioner   string
	PublicKeysPEM []byte
	ReviewMode    K8sTokenReviewMode
	APIServer     string
	CAFile        string
	BearerToken   string
	Namespaces    []string
}

// LoadK8sFromEnv reads Kubernetes provisioner environment variables.
func LoadK8sFromEnv() K8sConfig {
	enabled := strings.EqualFold(os.Getenv("CA_API_K8S_ENABLED"), "true")
	name := strings.TrimSpace(os.Getenv("CA_API_K8S_PROVISIONER_NAME"))
	if name == "" {
		name = "k8s-sa"
	}

	mode := K8sTokenReviewMode(strings.ToLower(strings.TrimSpace(os.Getenv("CA_API_K8S_TOKEN_REVIEW"))))
	if mode == "" {
		mode = K8sReviewAuto
	}

	var pubKeys []byte
	if path := strings.TrimSpace(os.Getenv("CA_API_K8S_PUBLIC_KEYS_FILE")); path != "" {
		if data, err := os.ReadFile(path); err == nil {
			pubKeys = data
		}
	} else if inline := strings.TrimSpace(os.Getenv("CA_API_K8S_PUBLIC_KEYS")); inline != "" {
		pubKeys = []byte(inline)
	}

	var namespaces []string
	if raw := strings.TrimSpace(os.Getenv("CA_API_K8S_ALLOWED_NAMESPACES")); raw != "" {
		for _, part := range strings.Split(raw, ",") {
			if ns := strings.TrimSpace(part); ns != "" {
				namespaces = append(namespaces, ns)
			}
		}
	}

	return K8sConfig{
		Enabled:       enabled,
		Provisioner:   name,
		PublicKeysPEM: pubKeys,
		ReviewMode:    mode,
		APIServer:     strings.TrimSpace(os.Getenv("CA_API_K8S_API_SERVER")),
		CAFile:        strings.TrimSpace(os.Getenv("CA_API_K8S_CA_FILE")),
		BearerToken:   strings.TrimSpace(os.Getenv("CA_API_K8S_BEARER_TOKEN")),
		Namespaces:    namespaces,
	}
}

// UsesTokenReviewAPI reports whether the Kubernetes API TokenReview path should be attempted.
func (k K8sConfig) UsesTokenReviewAPI() bool {
	switch k.ReviewMode {
	case K8sReviewAPI:
		return true
	case K8sReviewAuto:
		return k.APIServer != "" || k.hasInClusterConfig()
	default:
		return false
	}
}

func (k K8sConfig) hasInClusterConfig() bool {
	_, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount/token")
	return err == nil
}
