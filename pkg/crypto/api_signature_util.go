package crypto

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

const (
	HeaderTimestamp     = "timestamp"
	HeaderAPIName       = "api-name"
	HeaderAPIVersion    = "api-version"
	HeaderChannel       = "channel"
	HeaderUserID        = "user-id"
	HeaderAlgorithm     = "alg"
	HeaderCredential    = "access-key"
	HeaderSignedHeaders = "signed-headers"
	HeaderSignature     = "signature"
	TerminatorValue     = "@@"
	AlgorithmValue      = "HMAC-SHA256"
)

// Define custom error types
type SignatureError string

func (e SignatureError) Error() string {
	return string(e)
}

const (
	ErrMissingRequiredHeaders SignatureError = "MISSING_REQUIRED_HEADERS"
	ErrInvalidAlgorithm       SignatureError = "INVALID_ALGORITHM"
	ErrInvalidAccessKeyID     SignatureError = "INVALID_ACCESS_KEY_ID"
	ErrSignatureMissing       SignatureError = "SIGNATURE_MISSING"
	ErrInvalidSignedHeaders   SignatureError = "INVALID_SIGNED_HEADERS"
	ErrSignatureMismatch      SignatureError = "SIGNATURE_MISMATCH"
	ErrInvalidAccessSecret    SignatureError = "INVALID_ACCESS_SECRET"
	ErrInvalidAuthHeader      SignatureError = "INVALID_AUTHORIZATION_HEADER"
	ErrEmptyKeyID             SignatureError = "EMPTY_KEY_ID"
	ErrAccessKeyNotFound      SignatureError = "ACCESS_KEY_NOT_FOUND"
	ErrDatabaseError          SignatureError = "DATABASE_ERROR"
)

// AccessSecretProvider is an interface for retrieving access secrets.
// T represents the type of the secret being returned
type AccessSecretProvider[T any] interface {
	GetAccessSecret(accessKeyId string) (T, error)
}

type APIAccessKey struct {
	KeyID           string         `db:"key_id" json:"keyId"`
	Secret          string         `db:"secret" json:"secret"`
	InstitutionID   string         `db:"institution_id" json:"institutionId"`
	ApplicationName string         `db:"application_name" json:"applicationName"`
	Enabled         string         `db:"enabled" json:"enabled"`
	TestEnabled     string         `db:"test_enabled" json:"testEnabled"`
	Version         int16          `db:"version" json:"version"`
	ActiveFrom      time.Time      `db:"active_from" json:"activeFrom"`
	ActiveUntil     *time.Time     `db:"active_until" json:"activeUntil,omitempty"`
	CreatedAt       time.Time      `db:"created_at" json:"createdAt"`
	UpdatedAt       *time.Time     `db:"updated_at" json:"updatedAt,omitempty"`
	DiscardedAt     gorm.DeletedAt `db:"discarded_at" json:"discardedAt,omitempty"`
}

func (a *APIAccessKey) TableName() string {
	return "api_access_keys"
}

func (a *APIAccessKey) BeforeCreate(tx *gorm.DB) error {
	a.CreatedAt = time.Now()
	return nil
}

func (a *APIAccessKey) BeforeUpdate(tx *gorm.DB) error {
	now := time.Now()
	a.UpdatedAt = &now
	return nil
}

// DbAccessSecretProvider manages API access key storage and retrieval
type DbAccessSecretProvider struct {
	db         *gorm.DB
	accessKeys sync.Map
}

// NewDbAccessSecretProvider creates a new DbAccessSecretProvider instance
// with the provided database connection
func NewDbAccessSecretProvider(db *gorm.DB) *DbAccessSecretProvider {
	return &DbAccessSecretProvider{db: db}
}

func (r *DbAccessSecretProvider) GetAccessSecret(ctx context.Context, keyID string) (*APIAccessKey, error) {
	if keyID == "" {
		return nil, ErrEmptyKeyID
	}
	// Check cache first
	if cached, ok := r.accessKeys.Load(keyID); ok {
		return cached.(*APIAccessKey), nil
	}

	var accessKey APIAccessKey
	result := r.db.Where("key_id = ?", keyID).Find(&accessKey)
	if result.Error != nil {
		return nil, SignatureError(fmt.Sprintf("DATABASE_ERROR: %v", result.Error))
	}
	if result.RowsAffected == 0 {
		return nil, ErrAccessKeyNotFound
	}

	// Store in cache
	r.accessKeys.Store(keyID, &accessKey)
	return &accessKey, nil
}

// CreateAccessKey stores a new API access key in the database
// Returns an error if the operation fails
func (r *DbAccessSecretProvider) CreateAccessKey(ctx context.Context, accessKey *APIAccessKey) error {
	return r.db.WithContext(ctx).Create(accessKey).Error
}

func (r *DbAccessSecretProvider) UpdateAccessKey(ctx context.Context, accessKey *APIAccessKey) error {
	return r.db.WithContext(ctx).Save(accessKey).Error
}

func (r *DbAccessSecretProvider) DeleteAccessKey(ctx context.Context, keyID string) error {
	return r.db.WithContext(ctx).Delete(&APIAccessKey{}, "key_id = ?", keyID).Error
}

// Cryptographic Operations
func hmacSha256(data string, key []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return h.Sum(nil)
}

func sha256Hash(input string) []byte {
	h := sha256.New()
	h.Write([]byte(input))
	return h.Sum(nil)
}

func getSignatureKey(accessSecretKey, timeStamp, apiName, apiVersion string) []byte {
	kSecret := []byte(accessSecretKey)
	kDate := hmacSha256(timeStamp, kSecret)
	kVersion := hmacSha256(apiVersion, kDate)
	kApi := hmacSha256(apiName, kVersion)
	return hmacSha256(TerminatorValue, kApi)
}

// Signature Operations
func ComputeSignature(accessSecretKey, payloadHash string, headers map[string]string) (string, error) {
	timestamp := headers[HeaderTimestamp]
	apiName := headers[HeaderAPIName]
	apiVersion := headers[HeaderAPIVersion]
	channel := headers[HeaderChannel]
	userId := headers[HeaderUserID]

	if timestamp == "" || apiName == "" || apiVersion == "" || channel == "" || userId == "" {
		return "", ErrMissingRequiredHeaders
	}

	signingKey := getSignatureKey(accessSecretKey, timestamp, apiName, apiVersion)

	request := channel + userId + payloadHash

	stringToSign := AlgorithmValue + timestamp + hex.EncodeToString(sha256Hash(request))

	return hex.EncodeToString(hmacSha256(stringToSign, signingKey)), nil
}

func VerifySignature(signedHeadersValue, payloadHash, providedSignature, accessSecret string) (bool, error) {
	if providedSignature == "" {
		return false, ErrSignatureMissing
	}
	singedHeaders := splitKeyValue(signedHeadersValue, "/", "=")
	if len(singedHeaders) < 5 {
		return false, ErrInvalidSignedHeaders
	}
	computedSignature, err := ComputeSignature(accessSecret, payloadHash, singedHeaders)
	if err != nil {
		return false, err
	}
	if !strings.EqualFold(computedSignature, providedSignature) {
		return false, ErrSignatureMismatch
	}
	return true, nil
}

// splitKeyValue splits a string into key-value pairs using the provided separators.
func splitKeyValue(s, pairSep, kvSep string) map[string]string {
	result := make(map[string]string)
	pairs := strings.Split(s, pairSep)
	for _, pair := range pairs {
		kv := strings.SplitN(pair, kvSep, 2)
		if len(kv) == 2 {
			result[kv[0]] = kv[1]
		}
	}
	return result
}

// buildSignatureHeader constructs the authorization header string from its components.
//
// Parameters:
//   - credentialStr: The access key ID credential
//   - signedHeadersStr: A string containing all signed headers in format "header1=value1/header2=value2/"
//   - computedSignature: The computed signature in hexadecimal format
//
// Returns:
//   - string: A formatted authorization header string in the format:
//     "alg=HMAC-SHA256,access-key={credentialStr},signed-headers={signedHeadersStr},signature={computedSignature}"
//
// The function uses strings.Builder for efficient string concatenation and pre-allocates
// the required capacity to minimize memory allocations.
func buildSignatureHeader(credentialStr, signedHeadersStr, computedSignature string) string {
	const separator = ","

	var parts strings.Builder
	parts.Grow(len(HeaderAlgorithm) + len(AlgorithmValue) + len(separator) +
		len(HeaderCredential) + len(credentialStr) + len(separator) +
		len(HeaderSignedHeaders) + len(signedHeadersStr) + len(separator) +
		len(HeaderSignature) + len(computedSignature) + 4) // 4 for the separator =
	parts.WriteString(HeaderAlgorithm)
	parts.WriteString("=")
	parts.WriteString(AlgorithmValue)
	parts.WriteString(separator)
	parts.WriteString(HeaderCredential)
	parts.WriteString("=")
	parts.WriteString(credentialStr)
	parts.WriteString(separator)
	parts.WriteString(HeaderSignedHeaders)
	parts.WriteString("=")
	parts.WriteString(signedHeadersStr)
	parts.WriteString(separator)
	parts.WriteString(HeaderSignature)
	parts.WriteString("=")
	parts.WriteString(computedSignature)
	return parts.String()
}

// SignPayload generates a signature and required headers for API request authentication.
//
// Parameters:
//   - apiName: Name of the API being called
//   - apiVersion: Version of the API
//   - channel: Channel identifier for the request
//   - userId: User ID making the request
//   - payload: Request body or payload to be signed
//   - accessKeyId: Access key identifier for authentication
//   - accessSecret: Access secret for signature computation
//
// Returns:
//   - computedSignature: The computed signature for the request
//   - signatureHeader: Complete authorization header string
//   - signedHeaders: String containing all signed headers
//   - err: Error if signature generation fails
//
// The function performs the following steps:
//  1. Generates current timestamp in RFC3339 format
//  2. Validates required parameters
//  3. Computes payload hash and signature
//  4. Builds authorization header with all required components
//
// Possible errors:
//   - MISSING_REQUIRED_HEADERS: If any required header is empty
//   - INVALID_ACCESS_SECRET: If access secret cannot be retrieved
func SignPayload(apiName, apiVersion, channel, userId, payload, accessKeyId, accessSecret string) (computedSignature, signatureHeader, signedHeaders string, err error) {
	timestamp := time.Now().Format(time.RFC3339)
	if apiName == "" || apiVersion == "" || channel == "" || userId == "" {
		return "", "", "", ErrMissingRequiredHeaders
	}
	if accessSecret == "" {
		return "", "", "", ErrInvalidAccessSecret
	}
	headers := map[string]string{
		HeaderTimestamp:  timestamp,
		HeaderAPIName:    apiName,
		HeaderAPIVersion: apiVersion,
		HeaderChannel:    channel,
		HeaderUserID:     userId,
	}
	var signedHeadersBuilder strings.Builder
	for key, value := range headers {
		signedHeadersBuilder.WriteString(key)
		signedHeadersBuilder.WriteString("=")
		signedHeadersBuilder.WriteString(value)
		signedHeadersBuilder.WriteString("/")
	}
	signedHeaders = signedHeadersBuilder.String()
	payloadHash := hex.EncodeToString(sha256Hash(payload))
	computedSignature, err = ComputeSignature(accessSecret, payloadHash, headers)
	if err != nil {
		return "", "", "", err
	}
	signatureHeader = buildSignatureHeader(accessKeyId, signedHeaders, computedSignature)
	return computedSignature, signatureHeader, signedHeaders, nil
}

// ParseSignatureHeader parses the signature header value and returns its components.
// Format: "alg=HMAC-SHA256,access-key=access-key-id,signed-headers=header1=value1/header2=value2/,signature=signature"
// Sample header value
// alg=HMAC-SHA256,access-key=test-key-id,signed-headers=channel=web/user-id=test-user/timestamp=2025-01-05T21:00:02+05:30/api-name=test-api/api-version=v1/,signature=5b15ecf0a5a6cc14c12651f628a9bbc8958dcd8edc9bbe8e9970481bb72668af
// Returns:
//   - algorithm: The algorithm used for signature computation
//   - credentials: The access key ID
//   - signedHeaders: The headers used in signature computation
//   - signature: The computed signature
//   - err: Error if parsing fails
func ParseSignatureHeader(signatureHeaderValue string) (algorithm, credentials, signedHeaders, signature string, err error) {
	// Split by comma to separate main components
	parts := splitKeyValue(signatureHeaderValue, ",", "=")

	// Extract required fields
	algorithm = parts[HeaderAlgorithm]
	credentials = parts[HeaderCredential]
	signedHeaders = parts[HeaderSignedHeaders]
	signature = parts[HeaderSignature]

	// Validate all required fields are present
	if algorithm == "" || credentials == "" || signedHeaders == "" || signature == "" {
		return "", "", "", "", ErrInvalidAuthHeader
	}

	if !strings.EqualFold(algorithm, AlgorithmValue) {
		return "", "", "", "", ErrInvalidAlgorithm
	}

	return algorithm, credentials, signedHeaders, signature, nil
}

// Add validation for time-based operations
func (a *APIAccessKey) IsValid() bool {
	now := time.Now()
	if now.Before(a.ActiveFrom) {
		return false
	}
	if a.ActiveUntil != nil && now.After(*a.ActiveUntil) {
		return false
	}
	return a.Enabled == "Y" // assuming "Y" means enabled
}
