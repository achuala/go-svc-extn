package crypto

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

// MockAccessSecretProvider implements AccessSecretProvider for testing
type MockAccessSecretProvider struct {
	secrets map[string]string
}

func NewMockAccessSecretProvider() *MockAccessSecretProvider {
	return &MockAccessSecretProvider{
		secrets: map[string]string{
			"test-key-id": "test-secret-key",
		},
	}
}

func (m *MockAccessSecretProvider) GetAccessSecret(accessKeyId string) (string, error) {
	if secret, ok := m.secrets[accessKeyId]; ok {
		return secret, nil
	}
	return "", nil
}

func formatSignedHeader(headers map[string]string) string {
	var result string
	for k, v := range headers {
		if result != "" {
			result += "/"
		}
		result += k + "=" + v
	}
	return result
}

func TestComputeSignature(t *testing.T) {
	tests := []struct {
		name         string
		accessSecret string
		payload      string // Raw payload
		headers      map[string]string
		wantSignLen  int
	}{
		{
			name:         "Valid signature computation",
			accessSecret: "test-secret-key",
			payload:      `{"data":"test"}`,
			headers: map[string]string{
				"ts":    "2024-03-14T12:00:00Z",
				"api":   "test-api",
				"ver":   "v1",
				"chnl":  "web",
				"usrid": "test-user",
			},
			wantSignLen: 64,
		},
		{
			name:         "Empty payload",
			accessSecret: "test-secret-key",
			payload:      "",
			headers: map[string]string{
				"ts":    "2024-03-14T12:00:00Z",
				"api":   "test-api",
				"ver":   "v1",
				"chnl":  "web",
				"usrid": "test-user",
			},
			wantSignLen: 64,
		},
		{
			name:         "Different timestamp",
			accessSecret: "test-secret-key",
			payload:      `{"data":"test2"}`,
			headers: map[string]string{
				"ts":    "2024-03-14T13:00:00Z",
				"api":   "test-api",
				"ver":   "v1",
				"chnl":  "web",
				"usrid": "test-user",
			},
			wantSignLen: 64,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// First compute SHA256 of payload
			hasher := sha256.New()
			hasher.Write([]byte(tt.payload))
			payloadHash := hex.EncodeToString(hasher.Sum(nil))

			// Now compute signature using the hash
			signature := ComputeSignature(tt.accessSecret, payloadHash, tt.headers)

			// Check if signature is not empty
			if signature == "" {
				t.Error("Expected non-empty signature")
			}

			// Check signature length
			if len(signature) != tt.wantSignLen {
				t.Errorf("Expected signature length %d, got %d", tt.wantSignLen, len(signature))
			}

			// Verify the signature can be used in verification
			authHeader := "alg=HMAC-SHA256/creds=access-key:test-key-id/sign=" + signature
			signedHeader := formatSignedHeader(tt.headers)
			mockProvider := NewMockAccessSecretProvider()
			mockProvider.secrets["test-key-id"] = tt.accessSecret

			valid, err := VerifySignature(authHeader, signedHeader, payloadHash, mockProvider)
			if !valid || err != nil {
				t.Errorf("Signature verification failed: valid=%v, err=%v", valid, err)
			}
		})
	}
}

func TestVerifySignature(t *testing.T) {
	mockProvider := NewMockAccessSecretProvider()

	tests := []struct {
		name          string
		authHeader    string
		signedHeader  string
		payload       string
		expectedValid bool
		expectedError string
	}{
		{
			name:          "Valid signature verification",
			authHeader:    "alg=HMAC-SHA256/creds=access-key:test-key-id/sign=",
			signedHeader:  "ts=2024-03-14T12:00:00Z/api=test-api/ver=v1/chnl=web/usrid=test-user",
			payload:       `{"test":"data"}`,
			expectedValid: true,
			expectedError: "",
		},
		{
			name:          "Invalid authorization header",
			authHeader:    "invalid-header",
			signedHeader:  "ts=2024-03-14T12:00:00Z/api=test-api/ver=v1/chnl=web/usrid=test-user",
			payload:       `{"test":"data"}`,
			expectedValid: false,
			expectedError: "INVALID_AUTHORIZATION_HEADER",
		},
		{
			name:          "Invalid algorithm",
			authHeader:    "alg=MD5/creds=access-key:test-key-id/sign=test-signature",
			signedHeader:  "ts=2024-03-14T12:00:00Z/api=test-api/ver=v1/chnl=web/usrid=test-user",
			payload:       `{"test":"data"}`,
			expectedValid: false,
			expectedError: "INVALID_ALGORITHM",
		},
		{
			name:          "Missing signature",
			authHeader:    "alg=HMAC-SHA256/creds=access-key:test-key-id",
			signedHeader:  "ts=2024-03-14T12:00:00Z/api=test-api/ver=v1/chnl=web/usrid=test-user",
			payload:       `{"test":"data"}`,
			expectedValid: false,
			expectedError: "INVALID_AUTHORIZATION_HEADER",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For valid signature test, compute and append the actual signature
			if tt.expectedValid {
				headers := splitKeyValue(tt.signedHeader, "/", "=")
				signature := ComputeSignature(mockProvider.secrets["test-key-id"], tt.payload, headers)
				tt.authHeader = tt.authHeader + signature
			}

			valid, err := VerifySignature(tt.authHeader, tt.signedHeader, tt.payload, mockProvider)

			// Check error
			if tt.expectedError != "" {
				if err == nil || err.Error() != tt.expectedError {
					t.Errorf("Expected error %v, got %v", tt.expectedError, err)
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Check validity
			if valid != tt.expectedValid {
				t.Errorf("Expected valid=%v, got %v", tt.expectedValid, valid)
			}
		})
	}
}
