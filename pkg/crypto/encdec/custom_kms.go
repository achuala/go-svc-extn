package encdec

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"sync"

	"github.com/tink-crypto/tink-go/v2/aead"
	"github.com/tink-crypto/tink-go/v2/core/registry"
	"github.com/tink-crypto/tink-go/v2/keyset"
	"github.com/tink-crypto/tink-go/v2/tink"
)

const (
	customKmsPrefix = "custom-kms://"
	keyLength       = 32 // 256 bits
)

// CustomKMSClient implements the registry.KMSClient interface
type CustomKMSClient struct {
	keys     map[string][]byte
	keyMutex sync.RWMutex
}

// NewCustomKMSClient creates a new instance of CustomKMSClient
func NewCustomKMSClient() *CustomKMSClient {
	return &CustomKMSClient{
		keys: make(map[string][]byte),
	}
}

// Supported checks if the given key URI is supported by this KMS client
func (c *CustomKMSClient) Supported(keyURI string) bool {
	return strings.HasPrefix(keyURI, customKmsPrefix)
}

// GetAEAD returns an AEAD primitive for the given key URI
func (c *CustomKMSClient) GetAEAD(keyURI string) (tink.AEAD, error) {
	if !c.Supported(keyURI) {
		return nil, fmt.Errorf("unsupported key URI: %s", keyURI)
	}

	keyID := strings.TrimPrefix(keyURI, customKmsPrefix)

	c.keyMutex.RLock()
	_, exists := c.keys[keyID]
	c.keyMutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("key not found: %s", keyID)
	}

	kt := aead.AES256GCMKeyTemplate()
	kh, err := keyset.NewHandle(kt)
	if err != nil {
		return nil, fmt.Errorf("failed to create key handle: %v", err)
	}

	return aead.New(kh)
}

// CreateKey generates a new key and returns its URI
func (c *CustomKMSClient) CreateKey() (string, error) {
	keyMaterial := make([]byte, keyLength)
	_, err := rand.Read(keyMaterial)
	if err != nil {
		return "", fmt.Errorf("failed to generate random key: %v", err)
	}

	keyID := base64.RawURLEncoding.EncodeToString(keyMaterial)

	c.keyMutex.Lock()
	c.keys[keyID] = keyMaterial
	c.keyMutex.Unlock()

	return customKmsPrefix + keyID, nil
}

// RegisterCustomKMS registers the custom KMS client with Tink's registry
func RegisterCustomKMS() error {
	kmsClient := NewCustomKMSClient()
	registry.RegisterKMSClient(kmsClient)
	return nil
}
