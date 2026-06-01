package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	apiKeyPrefix     = "arx_sa_"
	apiKeyRandomSize = 32
)

var (
	// ErrAPIKeyNotFound is returned when the presented API key does not match any account.
	ErrAPIKeyNotFound = errors.New("api key not found or revoked")
	// ErrDuplicateServiceAccount is returned when a service account name already exists.
	ErrDuplicateServiceAccount = errors.New("service account name already exists")
	// ErrInvalidServiceAccountName is returned when the name is empty or too long.
	ErrInvalidServiceAccountName = errors.New("service account name must be between 1 and 128 characters")
)

// ServiceAccount represents a registered service account with a hashed API key.
type ServiceAccount struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Roles     []Role    `json:"roles"`
	KeyHash   string    `json:"-"`
	CreatedAt time.Time `json:"created_at"`
}

// APIKeyStore holds active service accounts and their key hashes in memory.
type APIKeyStore struct {
	mu       sync.RWMutex
	accounts map[string]*ServiceAccount
	byHash   map[string]string
	byName   map[string]string
}

// NewAPIKeyStore creates an empty, concurrent-safe API key store.
func NewAPIKeyStore() *APIKeyStore {
	return &APIKeyStore{
		accounts: make(map[string]*ServiceAccount),
		byHash:   make(map[string]string),
		byName:   make(map[string]string),
	}
}

// GenerateAPIKey creates a cryptographically secure API key and its storage hash.
// The plaintext key is returned once to the caller; only the hash is persisted.
func GenerateAPIKey() (plaintext string, hash string, err error) {
	random := make([]byte, apiKeyRandomSize)
	if _, err := rand.Read(random); err != nil {
		return "", "", fmt.Errorf("auth: generate api key entropy: %w", err)
	}
	encoded := base64.RawURLEncoding.EncodeToString(random)
	plaintext = apiKeyPrefix + encoded
	hash = hashAPIKey(plaintext)
	return plaintext, hash, nil
}

// hashAPIKey returns a hex-encoded SHA-256 digest of the API key for map lookup.
func hashAPIKey(plaintext string) string {
	sum := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(sum[:])
}

// CreateServiceAccount registers a new service account and returns the account plus plaintext key.
func (s *APIKeyStore) CreateServiceAccount(name string, roles []Role) (*ServiceAccount, string, error) {
	if len(name) == 0 || len(name) > 128 {
		return nil, "", ErrInvalidServiceAccountName
	}

	roles = NormalizeRoles(roles)
	if len(roles) == 0 {
		roles = DefaultServiceAccountRoles()
	}

	plaintext, keyHash, err := GenerateAPIKey()
	if err != nil {
		return nil, "", err
	}

	id := uuid.NewString()
	account := &ServiceAccount{
		ID:        id,
		Name:      name,
		Roles:     roles,
		KeyHash:   keyHash,
		CreatedAt: time.Now().UTC(),
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.byName[name]; exists {
		return nil, "", ErrDuplicateServiceAccount
	}

	s.accounts[id] = account
	s.byHash[keyHash] = id
	s.byName[name] = id

	return account, plaintext, nil
}

// ValidateAPIKey checks the plaintext key against stored hashes.
func (s *APIKeyStore) ValidateAPIKey(plaintext string) (*ServiceAccount, error) {
	if plaintext == "" {
		return nil, ErrAPIKeyNotFound
	}

	digest := hashAPIKey(plaintext)

	s.mu.RLock()
	defer s.mu.RUnlock()

	id, ok := s.byHash[digest]
	if !ok {
		return nil, ErrAPIKeyNotFound
	}

	account, ok := s.accounts[id]
	if !ok {
		return nil, ErrAPIKeyNotFound
	}
	return account, nil
}

// RevokeServiceAccount removes a service account and its key from the store.
func (s *APIKeyStore) RevokeServiceAccount(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	account, ok := s.accounts[id]
	if !ok {
		return ErrAPIKeyNotFound
	}

	delete(s.byHash, account.KeyHash)
	delete(s.byName, account.Name)
	delete(s.accounts, id)
	return nil
}
