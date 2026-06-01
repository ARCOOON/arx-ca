package config

import (
	"context"
	"os"
	"strings"
)

// AWSConfig holds optional AWS KMS / IAM integration settings (placeholder).
type AWSConfig struct {
	Enabled         bool
	Region          string
	KMSKeyARN       string
	RoleARN         string
	CredentialsFile string
}

// LoadAWSFromEnv reads AWS integration environment variables.
func LoadAWSFromEnv() AWSConfig {
	enabled := strings.EqualFold(os.Getenv("CA_API_AWS_ENABLED"), "true") ||
		strings.TrimSpace(os.Getenv("CA_API_AWS_KMS_KEY_ARN")) != ""
	return AWSConfig{
		Enabled:         enabled,
		Region:          strings.TrimSpace(os.Getenv("CA_API_AWS_REGION")),
		KMSKeyARN:       strings.TrimSpace(os.Getenv("CA_API_AWS_KMS_KEY_ARN")),
		RoleARN:         strings.TrimSpace(os.Getenv("CA_API_AWS_ROLE_ARN")),
		CredentialsFile: strings.TrimSpace(os.Getenv("CA_API_AWS_CREDENTIALS_FILE")),
	}
}

// AWSKMSClient is the interface a future AWS KMS CAS plugin would implement.
type AWSKMSClient interface {
	Healthy(ctx context.Context) error
	SignDigest(ctx context.Context, keyARN string, digest []byte) ([]byte, error)
}

// StubAWSKMSClient is a no-op placeholder until AWS KMS integration is implemented.
type StubAWSKMSClient struct{}

func (StubAWSKMSClient) Healthy(context.Context) error {
	return ErrIntegrationDisabled
}

func (StubAWSKMSClient) SignDigest(context.Context, string, []byte) ([]byte, error) {
	return nil, ErrIntegrationDisabled
}

// NewAWSKMSClient returns a client implementation based on configuration.
func NewAWSKMSClient(cfg AWSConfig) AWSKMSClient {
	if !cfg.Enabled {
		return StubAWSKMSClient{}
	}
	return StubAWSKMSClient{}
}
