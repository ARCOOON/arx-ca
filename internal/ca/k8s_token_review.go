package ca

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"go.step.sm/crypto/jose"
	"go.step.sm/crypto/pemutil"

	"github.com/your-org/arx-ca/internal/config"
)

const k8sServiceAccountIssuer = "kubernetes/serviceaccount"

// K8sServiceAccountIdentity holds validated Kubernetes workload identity claims.
type K8sServiceAccountIdentity struct {
	Subject            string
	Namespace          string
	ServiceAccountName string
	ServiceAccountUID  string
	Username           string
}

// K8sTokenReviewer validates Kubernetes service account JWTs locally and/or via TokenReview.
type K8sTokenReviewer struct {
	cfg        config.K8sConfig
	pubKeys    []any
	httpClient *http.Client
}

// NewK8sTokenReviewer builds a reviewer from configuration.
func NewK8sTokenReviewer(cfg config.K8sConfig) (*K8sTokenReviewer, error) {
	r := &K8sTokenReviewer{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
	if len(cfg.PublicKeysPEM) > 0 {
		keys, err := parseK8sPublicKeys(cfg.PublicKeysPEM)
		if err != nil {
			return nil, err
		}
		r.pubKeys = keys
	}
	return r, nil
}

// Review validates a Kubernetes service account token and returns workload identity.
func (r *K8sTokenReviewer) Review(ctx context.Context, token string) (*K8sServiceAccountIdentity, error) {
	if r == nil {
		return nil, errors.New("kubernetes token reviewer is not configured")
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, errors.New("kubernetes token is required")
	}

	switch r.cfg.ReviewMode {
	case config.K8sReviewAPI:
		return r.reviewViaAPI(ctx, token)
	case config.K8sReviewLocal:
		return r.reviewLocally(token)
	default:
		if r.cfg.UsesTokenReviewAPI() {
			if id, err := r.reviewViaAPI(ctx, token); err == nil {
				return id, nil
			}
		}
		if len(r.pubKeys) > 0 {
			return r.reviewLocally(token)
		}
		return nil, errors.New("kubernetes token could not be validated: configure public keys or TokenReview API access")
	}
}

func (r *K8sTokenReviewer) reviewLocally(token string) (*K8sServiceAccountIdentity, error) {
	if len(r.pubKeys) == 0 {
		return nil, errors.New("kubernetes public keys are not configured for local token review")
	}

	jwt, err := jose.ParseSigned(token)
	if err != nil {
		return nil, fmt.Errorf("parse kubernetes token: %w", err)
	}

	var claims k8sSAClaims
	var valid bool
	for _, pk := range r.pubKeys {
		if err = jwt.Claims(pk, &claims); err == nil {
			valid = true
			break
		}
	}
	if !valid {
		return nil, errors.New("kubernetes token signature verification failed")
	}

	if err = claims.Validate(jose.Expected{Issuer: k8sServiceAccountIssuer}); err != nil {
		return nil, fmt.Errorf("invalid kubernetes token claims: %w", err)
	}
	if claims.Subject == "" {
		return nil, errors.New("kubernetes token subject is empty")
	}

	id := &K8sServiceAccountIdentity{
		Subject:            claims.Subject,
		Namespace:          claims.Namespace,
		ServiceAccountName: claims.ServiceAccountName,
		ServiceAccountUID:  claims.ServiceAccountUID,
		Username:           claims.Subject,
	}
	if err := r.enforceNamespace(id.Namespace); err != nil {
		return nil, err
	}
	return id, nil
}

func (r *K8sTokenReviewer) reviewViaAPI(ctx context.Context, token string) (*K8sServiceAccountIdentity, error) {
	apiServer, bearer, caPEM, err := r.resolveClusterCredentials()
	if err != nil {
		return nil, err
	}

	body := map[string]any{
		"apiVersion": "authentication.k8s.io/v1",
		"kind":       "TokenReview",
		"spec": map[string]any{
			"token": token,
		},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(apiServer, "/")+"/apis/authentication.k8s.io/v1/tokenreviews", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+bearer)

	client := r.httpClient
	if len(caPEM) > 0 {
		client, err = clientWithCA(caPEM)
		if err != nil {
			return nil, err
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("kubernetes TokenReview request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("kubernetes TokenReview returned %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var review tokenReviewResponse
	if err := json.Unmarshal(respBody, &review); err != nil {
		return nil, fmt.Errorf("decode TokenReview response: %w", err)
	}
	if review.Status.Error != "" {
		return nil, fmt.Errorf("kubernetes TokenReview error: %s", review.Status.Error)
	}
	if !review.Status.Authenticated {
		return nil, errors.New("kubernetes token was not authenticated")
	}

	id := &K8sServiceAccountIdentity{
		Subject:  review.Status.User.Username,
		Username: review.Status.User.Username,
	}
	for _, ref := range review.Status.User.Extra["authentication.kubernetes.io/service-account.name"] {
		if id.ServiceAccountName == "" {
			id.ServiceAccountName = ref
		}
	}
	for _, ref := range review.Status.User.Extra["authentication.kubernetes.io/service-account.namespace"] {
		if id.Namespace == "" {
			id.Namespace = ref
		}
	}
	for _, ref := range review.Status.User.Extra["authentication.kubernetes.io/service-account.uid"] {
		if id.ServiceAccountUID == "" {
			id.ServiceAccountUID = ref
		}
	}

	if id.Namespace == "" || id.ServiceAccountName == "" {
		if idParsed, err := parseK8sSubject(id.Username); err == nil {
			id.Namespace = idParsed.Namespace
			id.ServiceAccountName = idParsed.ServiceAccountName
		}
	}

	if err := r.enforceNamespace(id.Namespace); err != nil {
		return nil, err
	}
	return id, nil
}

func (r *K8sTokenReviewer) enforceNamespace(namespace string) error {
	if len(r.cfg.Namespaces) == 0 {
		return nil
	}
	for _, allowed := range r.cfg.Namespaces {
		if namespace == allowed {
			return nil
		}
	}
	return fmt.Errorf("namespace %q is not allowed for kubernetes enrollment", namespace)
}

func (r *K8sTokenReviewer) resolveClusterCredentials() (apiServer, bearer string, caPEM []byte, err error) {
	apiServer = r.cfg.APIServer
	bearer = r.cfg.BearerToken
	if path := r.cfg.CAFile; path != "" {
		caPEM, err = os.ReadFile(path)
		if err != nil {
			return "", "", nil, fmt.Errorf("read kubernetes CA file: %w", err)
		}
	}

	if apiServer == "" {
		if host, port := os.Getenv("KUBERNETES_SERVICE_HOST"), os.Getenv("KUBERNETES_SERVICE_PORT"); host != "" {
			apiServer = "https://" + host + ":" + port
		}
	}
	if bearer == "" {
		if data, readErr := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token"); readErr == nil {
			bearer = strings.TrimSpace(string(data))
		}
	}
	if len(caPEM) == 0 {
		if data, readErr := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"); readErr == nil {
			caPEM = data
		}
	}

	if apiServer == "" {
		return "", "", nil, errors.New("kubernetes API server is not configured (set CA_API_K8S_API_SERVER or run in-cluster)")
	}
	if bearer == "" {
		return "", "", nil, errors.New("kubernetes API bearer token is not configured")
	}
	return apiServer, bearer, caPEM, nil
}

type k8sSAClaims struct {
	jose.Claims
	Namespace          string `json:"kubernetes.io/serviceaccount/namespace,omitempty"`
	ServiceAccountName string `json:"kubernetes.io/serviceaccount/service-account.name,omitempty"`
	ServiceAccountUID  string `json:"kubernetes.io/serviceaccount/service-account.uid,omitempty"`
}

type tokenReviewResponse struct {
	Status struct {
		Authenticated bool   `json:"authenticated"`
		Error         string `json:"error,omitempty"`
		User          struct {
			Username string              `json:"username"`
			Extra    map[string][]string `json:"extra,omitempty"`
		} `json:"user"`
	} `json:"status"`
}

func parseK8sPublicKeys(pemBytes []byte) ([]any, error) {
	var keys []any
	rest := pemBytes
	for len(rest) > 0 {
		block, remaining := pem.Decode(rest)
		if block == nil {
			break
		}
		rest = remaining
		key, err := pemutil.ParseKey(pem.EncodeToMemory(block))
		if err != nil {
			return nil, fmt.Errorf("parse kubernetes public key: %w", err)
		}
		switch k := key.(type) {
		case *rsa.PublicKey, *ecdsa.PublicKey, ed25519.PublicKey:
			keys = append(keys, k)
		default:
			return nil, fmt.Errorf("unsupported kubernetes public key type %T", key)
		}
	}
	if len(keys) == 0 {
		return nil, errors.New("no kubernetes public keys found in PEM data")
	}
	return keys, nil
}

func isKubernetesServiceAccountToken(token string) bool {
	jwt, err := jose.ParseSigned(token)
	if err != nil {
		return false
	}
	var claims jose.Claims
	if err := jwt.UnsafeClaimsWithoutVerification(&claims); err != nil {
		return false
	}
	return claims.Issuer == k8sServiceAccountIssuer
}

type parsedK8sSubject struct {
	Namespace          string
	ServiceAccountName string
}

func parseK8sSubject(subject string) (*parsedK8sSubject, error) {
	// system:serviceaccount:namespace:name
	const prefix = "system:serviceaccount:"
	if !strings.HasPrefix(subject, prefix) {
		return nil, fmt.Errorf("unexpected kubernetes subject %q", subject)
	}
	parts := strings.SplitN(strings.TrimPrefix(subject, prefix), ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("unexpected kubernetes subject %q", subject)
	}
	return &parsedK8sSubject{Namespace: parts[0], ServiceAccountName: parts[1]}, nil
}
