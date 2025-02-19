package crypto

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

const TIMESTAMP_KEY = "timestamp"
const API_KEY = "api-name"
const VERSION_KEY = "api-version"
const CHANNEL_KEY = "channel"
const USER_ID_KEY = "user-id"
const ALGORITHM_KEY = "alg"
const CREDENTIAL_KEY = "access-key"
const SIGNED_HEADERS_KEY = "signed-headers"
const SIGNATURE_KEY = "signature"

const TERMINATOR_VALUE = "@@"
const ALGORITHM_VALUE = "HMAC-SHA256"

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

type DbAccessSecretProvider struct {
	db         *gorm.DB
	accessKeys sync.Map
}

func NewDbAccessSecretProvider(db *gorm.DB) *DbAccessSecretProvider {
	return &DbAccessSecretProvider{db: db}
}

func (r *DbAccessSecretProvider) GetAccessSecret(ctx context.Context, keyID string) (*APIAccessKey, error) {
	// Check cache first
	if cached, ok := r.accessKeys.Load(keyID); ok {
		return cached.(*APIAccessKey), nil
	}

	var accessKey APIAccessKey
	if keyID == "" {
		return nil, errors.New("EMPTY_KEY_ID")
	}

	result := r.db.Where("key_id = ?", keyID).Find(&accessKey)
	if result.Error != nil {
		return nil, fmt.Errorf("database error: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, errors.New("ACCESS_KEY_NOT_FOUND")
	}

	// Store in cache
	r.accessKeys.Store(keyID, &accessKey)
	return &accessKey, nil
}

func (r *DbAccessSecretProvider) CreateAccessKey(accessKey *APIAccessKey) error {
	return r.db.Create(accessKey).Error
}

func (r *DbAccessSecretProvider) UpdateAccessKey(accessKey *APIAccessKey) error {
	return r.db.Save(accessKey).Error
}

func (r *DbAccessSecretProvider) DeleteAccessKey(keyID string) error {
	return r.db.Delete(&APIAccessKey{}, "key_id = ?", keyID).Error
}

// HmacSha256 computes the HMAC-SHA256 of the given data using the provided key.
// It returns the resulting hash as a byte slice.
func hmacSha256(data string, key []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return h.Sum(nil)
}

// Sha256 computes the SHA256 hash of the input string.
// It returns the resulting hash as a byte slice.
func sha256Hash(input string) []byte {
	h := sha256.New()
	h.Write([]byte(input))
	return h.Sum(nil)
}

// GetSignatureKey generates a signature key using the provided parameters.
// It combines the access secret key, timestamp, API name, and API version
// to create a unique signature key.
func getSignatureKey(accessSecretKey, timeStamp, apiName, apiVersion string) []byte {

	kSecret := []byte(accessSecretKey)
	kDate := hmacSha256(timeStamp, kSecret)
	kVersion := hmacSha256(apiVersion, kDate)
	kApi := hmacSha256(apiName, kVersion)
	return hmacSha256(TERMINATOR_VALUE, kApi)
}

// ComputeSignature generates a cryptographic signature for API request validation.
// It uses HMAC-SHA256 algorithm to create a signature based on the provided secret key,
// payload, and headers.
//
// Parameters:
//   - accessSecretKey: The secret key used for signature generation
//   - payloadHash: The SHA256 hash of the request body or payload in hexadecimal format
//   - headers: A map containing required headers:
//   - "timestamp": Timestamp
//   - "api-name": API name
//   - "api-version": API version
//   - "channel": Channel identifier
//   - "user-id": User ID
//
// Returns:
//   - string: The computed signature as a hexadecimal string
//
// The signature is computed using the following steps:
//  1. Generate a signing key using the secret key and header information
//  2. Combine channel, userId, and payload hash
//  3. Create final signature using algorithm, timestamp, and request hash
func ComputeSignature(accessSecretKey, payloadHash string, headers map[string]string) (string, error) {
	timestamp := headers[TIMESTAMP_KEY]
	apiName := headers[API_KEY]
	apiVersion := headers[VERSION_KEY]
	channel := headers[CHANNEL_KEY]
	userId := headers[USER_ID_KEY]

	if timestamp == "" || apiName == "" || apiVersion == "" || channel == "" || userId == "" {
		return "", ErrMissingRequiredHeaders
	}

	signingKey := getSignatureKey(accessSecretKey, timestamp, apiName, apiVersion)

	request := channel + userId + payloadHash

	stringToSign := ALGORITHM_VALUE + timestamp + hex.EncodeToString(sha256Hash(request))

	return hex.EncodeToString(hmacSha256(stringToSign, signingKey)), nil
}

// VerifySignature validates the authenticity of a request by comparing the provided signature
// with a computed signature using the request payload and headers.
//
// Parameters:
//   - signedHeadersValue: The signed headers value in the format "header1=value1/header2=value2/"
//   - payloadHash: The SHA256 hash of the request body or payload in hexadecimal format
//   - signedHeadersValue: The signed headers value in the format "header1=value1/header2=value2/"
//   - providedSignature: The provided signature to be verified, in hexadecimal format
//   - accessSecret: The access secret key for signature computation and validation
//
// Use ParseAuthorizationHeader to extract the values and pass it here.
// Returns:
//   - bool: true if signature is valid, false otherwise
//   - error: Error if validation fails or if required parameters are missing/invalid
//
// Possible errors:
//   - SIGNATURE_MISSING: If signature is not provided
//   - INVALID_SIGNED_HEADERS: If required headers are missing
//   - SIGNATURE_MISMATCH: If computed signature doesn't match provided signature
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
	parts.Grow(len(ALGORITHM_KEY) + len(ALGORITHM_VALUE) + len(separator) +
		len(CREDENTIAL_KEY) + len(credentialStr) + len(separator) +
		len(SIGNED_HEADERS_KEY) + len(signedHeadersStr) + len(separator) +
		len(SIGNATURE_KEY) + len(computedSignature))
	parts.WriteString(ALGORITHM_KEY)
	parts.WriteString(ALGORITHM_VALUE)
	parts.WriteString(separator)
	parts.WriteString(CREDENTIAL_KEY)
	parts.WriteString(credentialStr)
	parts.WriteString(separator)
	parts.WriteString(SIGNED_HEADERS_KEY)
	parts.WriteString(signedHeadersStr)
	parts.WriteString(separator)
	parts.WriteString(SIGNATURE_KEY)
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
		TIMESTAMP_KEY: timestamp,
		API_KEY:       apiName,
		VERSION_KEY:   apiVersion,
		CHANNEL_KEY:   channel,
		USER_ID_KEY:   userId,
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

// ParseAuthorizationHeader parses the authorization header value and returns its components.
// Format: "alg=HMAC-SHA256,access-key=access-key-id,signed-headers=header1=value1/header2=value2/,signature=signature"
// Sample header value
// alg=HMAC-SHA256,access-key=test-key-id,signed-headers=channel=web/user-id=test-user/timestamp=2025-01-05T21:00:02+05:30/api-name=test-api/api-version=v1/,signature=5b15ecf0a5a6cc14c12651f628a9bbc8958dcd8edc9bbe8e9970481bb72668af
// Returns:
//   - algorithm: The algorithm used for signature computation
//   - credentials: The access key ID
//   - signedHeaders: The headers used in signature computation
//   - signature: The computed signature
//   - err: Error if parsing fails
func ParseAuthorizationHeader(authorizationHeaderValue string) (algorithm, credentials, signedHeaders, signature string, err error) {
	// Split by comma to separate main components
	parts := splitKeyValue(authorizationHeaderValue, ",", "=")

	// Extract required fields
	algorithm = parts[ALGORITHM_KEY]
	credentials = parts[CREDENTIAL_KEY]
	signedHeaders = parts[SIGNED_HEADERS_KEY]
	signature = parts[SIGNATURE_KEY]

	// Validate all required fields are present
	if algorithm == "" || credentials == "" || signedHeaders == "" || signature == "" {
		return "", "", "", "", ErrInvalidAuthHeader
	}

	if !strings.EqualFold(algorithm, ALGORITHM_VALUE) {
		return "", "", "", "", ErrInvalidAlgorithm
	}

	return algorithm, credentials, signedHeaders, signature, nil
}

// Add constants for error messages
const (
	ErrEmptyKeyID        = "EMPTY_KEY_ID"
	ErrAccessKeyNotFound = "ACCESS_KEY_NOT_FOUND"
	ErrInvalidSignature  = "SIGNATURE_MISMATCH"
	ErrMissingHeaders    = "MISSING_REQUIRED_HEADERS"
	// ... other error constants
)

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
