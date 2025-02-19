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
		wantErr      bool
	}{
		{
			name:         "Valid signature computation",
			accessSecret: "test-secret-key",
			payload:      `{"data":"test"}`,
			headers: map[string]string{
				HeaderTimestamp:  "2024-03-14T12:00:00Z",
				HeaderAPIName:    "test-api",
				HeaderAPIVersion: "v1",
				HeaderChannel:    "web",
				HeaderUserID:     "test-user",
			},
			wantSignLen: 64,
			wantErr:     false,
		},
		{
			name:         "Empty payload",
			accessSecret: "test-secret-key",
			payload:      "",
			headers: map[string]string{
				HeaderTimestamp:  "2024-03-14T12:00:00Z",
				HeaderAPIName:    "test-api",
				HeaderAPIVersion: "v1",
				HeaderChannel:    "web",
				HeaderUserID:     "test-user",
			},
			wantSignLen: 64,
			wantErr:     false,
		},
		{
			name:         "Different timestamp",
			accessSecret: "test-secret-key",
			payload:      `{"data":"test2"}`,
			headers: map[string]string{
				HeaderTimestamp:  "2024-03-14T13:00:00Z",
				HeaderAPIName:    "test-api",
				HeaderAPIVersion: "v1",
				HeaderChannel:    "web",
				HeaderUserID:     "test-user",
			},
			wantSignLen: 64,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// First compute SHA256 of payload
			hasher := sha256.New()
			hasher.Write([]byte(tt.payload))
			payloadHash := hex.EncodeToString(hasher.Sum(nil))

			// Now compute signature using the hash
			signature, err := ComputeSignature(tt.accessSecret, payloadHash, tt.headers)
			if err != nil {
				t.Errorf("ComputeSignature() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Check if signature is not empty
				if signature == "" {
					t.Error("Expected non-empty signature")
				}

				// Check signature length
				if len(signature) != tt.wantSignLen {
					t.Errorf("Expected signature length %d, got %d", tt.wantSignLen, len(signature))
				}

				// Verify the signature can be used in verification
				signedHeaders := formatSignedHeader(tt.headers)
				mockProvider := NewMockAccessSecretProvider()
				mockProvider.secrets["test-key-id"] = tt.accessSecret

				valid, err := VerifySignature(signedHeaders, payloadHash, signature, tt.accessSecret)
				if !valid || err != nil {
					t.Errorf("Signature verification failed: valid=%v, err=%v", valid, err)
				}
			}
		})
	}
}

func TestSignPayload(t *testing.T) {
	mockProvider := NewMockAccessSecretProvider()
	mockProvider.secrets["test-key-id"] = "test-secret-key"

	tests := []struct {
		name     string
		api      string
		ver      string
		chnl     string
		usrid    string
		payload  string
		keyID    string
		provider AccessSecretProvider[string]
		wantErr  bool
	}{
		{
			name:     "Valid payload signing",
			api:      "test-api",
			ver:      "v1",
			chnl:     "web",
			usrid:    "test-user",
			payload:  `{"test":"data"}`,
			keyID:    "test-key-id",
			provider: mockProvider,
			wantErr:  false,
		},
		{
			name:     "Missing required headers",
			api:      "",
			ver:      "v1",
			chnl:     "web",
			usrid:    "test-user",
			payload:  `{"test":"data"}`,
			keyID:    "test-key-id",
			provider: mockProvider,
			wantErr:  true,
		},
		{
			name:  "Invalid key ID",
			api:   "test-api",
			ver:   "v1",
			chnl:  "web",
			usrid: "test-user",

			payload:  `{"test":"data"}`,
			keyID:    "invalid-key",
			provider: mockProvider,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			signature, authHeader, signedHeader, err := SignPayload(tt.api, tt.ver, tt.chnl, tt.usrid, tt.payload, tt.keyID, mockProvider.secrets[tt.keyID])
			if (err != nil) != tt.wantErr {
				t.Errorf("SignPayload() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if signature == "" || authHeader == "" || signedHeader == "" {
					t.Errorf("Expected non-empty signature, authHeader, and signedHeader")
				}
				payloadHash := hex.EncodeToString(sha256Hash(tt.payload))
				// Verify the generated signature
				valid, err := VerifySignature(signedHeader, payloadHash, signature, mockProvider.secrets["test-key-id"])
				if !valid || err != nil {
					t.Errorf("Generated signature verification failed: valid=%v, err=%v", valid, err)
				}
			}
		})
	}
}

func TestParseSignatureHeader(t *testing.T) {
	tests := []struct {
		name          string
		authHeader    string
		wantAlg       string
		wantCreds     string
		wantHeaders   string
		wantSignature string
		expectedError string
	}{
		{
			name:          "Valid header",
			authHeader:    "alg=HMAC-SHA256,access-key=test-key-id,signed-headers=timestamp=2024-03-14T12:00:00Z/api-name=test-api/api-version=v1/,signature=test-signature",
			wantAlg:       "HMAC-SHA256",
			wantCreds:     "test-key-id",
			wantHeaders:   "timestamp=2024-03-14T12:00:00Z/api-name=test-api/api-version=v1/",
			wantSignature: "test-signature",
			expectedError: "",
		},
		{
			name:          "Missing algorithm",
			authHeader:    "access-key=test-key-id,signed-headers=timestamp=2024-03-14T12:00:00Z/api-name=test-api/api-version=v1/,signature=test-signature",
			expectedError: "INVALID_AUTHORIZATION_HEADER",
		},
		{
			name:          "Missing credentials",
			authHeader:    "alg=HMAC-SHA256,signed-headers=timestamp=2024-03-14T12:00:00Z/api-name=test-api/api-version=v1/,signature=test-signature",
			expectedError: "INVALID_AUTHORIZATION_HEADER",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			alg, creds, headers, signature, err := ParseSignatureHeader(tt.authHeader)

			if tt.expectedError != "" {
				if err == nil || err.Error() != tt.expectedError {
					t.Errorf("Expected error %v, got %v", tt.expectedError, err)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if alg != tt.wantAlg {
				t.Errorf("Expected algorithm %v, got %v", tt.wantAlg, alg)
			}
			if creds != tt.wantCreds {
				t.Errorf("Expected credentials %v, got %v", tt.wantCreds, creds)
			}
			if headers != tt.wantHeaders {
				t.Errorf("Expected headers %v, got %v", tt.wantHeaders, headers)
			}
			if signature != tt.wantSignature {
				t.Errorf("Expected signature %v, got %v", tt.wantSignature, signature)
			}
		})
	}
}
