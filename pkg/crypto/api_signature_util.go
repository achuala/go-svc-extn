package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

const ALGORITHM_KEY = "HMAC-SHA256"
const TERMINATOR = "@@"
const CREDENTIAL_KEY = "creds"
const SIGNED_HEADERS_KEY = "x-kplex-si"
const SIGNATURE_KEY = "sign"

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
	accessKeys map[string]*APIAccessKey
}

func NewDbAccessSecretProvider(db *gorm.DB) *DbAccessSecretProvider {
	return &DbAccessSecretProvider{db: db}
}

func (r *DbAccessSecretProvider) GetAccessSecret(keyID string) (*APIAccessKey, error) {
	var accessKey APIAccessKey
	err := r.db.Where("key_id = ?", keyID).Find(&accessKey).Error
	if err != nil {
		return nil, err
	}
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
	return hmacSha256(TERMINATOR, kApi)
}

// ComputeSignature generates a cryptographic signature for API request validation.
// It uses HMAC-SHA256 algorithm to create a signature based on the provided secret key,
// payload, and headers.
//
// Parameters:
//   - accessSecretKey: The secret key used for signature generation
//   - payloadHash: The SHA256 hash of the request body or payload in hexadecimal format
//   - headers: A map containing required headers:
//   - "ts": Timestamp
//   - "api": API name
//   - "ver": API version
//   - "chnl": Channel identifier
//   - "usrid": User ID
//
// Returns:
//   - string: The computed signature as a hexadecimal string
//
// The signature is computed using the following steps:
//  1. Generate a signing key using the secret key and header information
//  2. Combine channel, userId, and payload hash
//  3. Create final signature using algorithm, timestamp, and request hash
func ComputeSignature(accessSecretKey, payloadHash string, headers map[string]string) string {
	timestamp := headers["ts"]
	apiName := headers["api"]
	apiVersion := headers["ver"]
	channel := headers["chnl"]
	userId := headers["usrid"]

	signingKey := getSignatureKey(accessSecretKey, timestamp, apiName, apiVersion)

	request := channel + userId + payloadHash

	stringToSign := ALGORITHM_KEY + timestamp + hex.EncodeToString(sha256Hash(request))

	return hex.EncodeToString(hmacSha256(stringToSign, signingKey))
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
		return false, errors.New("SIGNATURE_MISSING")
	}
	singedHeaders := splitKeyValue(signedHeadersValue, "/", "=")
	if len(singedHeaders) < 5 {
		return false, errors.New("INVALID_SIGNED_HEADERS")
	}
	computedSignature := ComputeSignature(accessSecret, payloadHash, singedHeaders)
	if !strings.EqualFold(computedSignature, providedSignature) {
		return false, errors.New("SIGNATURE_MISMATCH")
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

func buildAuthorizationHeader(credentialStr, signedHeadersStr, signingSignature string) string {
	const algorithmKey = "alg="
	const algorithm = "HMAC-SHA256"
	const credential = "creds="
	const signedHeaders = "signed-headers="
	const signature = "sign="
	const separator = ","

	var parts strings.Builder
	parts.Grow(len(algorithmKey) + len(algorithm) + len(separator) +
		len(credential) + len(credentialStr) + len(separator) +
		len(signedHeaders) + len(signedHeadersStr) + len(separator) +
		len(signature) + len(signingSignature),
	)
	parts.WriteString(algorithmKey)
	parts.WriteString(algorithm)
	parts.WriteString(separator)
	parts.WriteString(credential)
	parts.WriteString(credentialStr)
	parts.WriteString(separator)
	parts.WriteString(signedHeaders)
	parts.WriteString(signedHeadersStr)
	parts.WriteString(separator)
	parts.WriteString(signature)
	parts.WriteString(signingSignature)
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
//   - accessSecretProvider: Interface to retrieve access secrets
//
// Returns:
//   - signature: The computed signature for the request
//   - authHeader: Complete authorization header string
//   - signedHeader: String containing all signed headers
//   - err: Error if signature generation fails
//
// The function performs the following steps:
//  1. Generates current timestamp in RFC3339 format
//  2. Retrieves access secret using the provided accessKeyId
//  3. Validates required parameters
//  4. Computes payload hash and signature
//  5. Builds authorization header with all required components
//
// Possible errors:
//   - MISSING_REQUIRED_HEADERS: If any required header is empty
//   - INVALID_ACCESS_SECRET: If access secret cannot be retrieved
func SignPayload(apiName, apiVersion, channel, userId, payload, accessKeyId string, accessSecretProvider AccessSecretProvider[string]) (signature, authHeader, signedHeader string, err error) {
	timestamp := time.Now().Format(time.RFC3339)
	accessSecret, err := accessSecretProvider.GetAccessSecret(accessKeyId)
	if err != nil {
		return "", "", "", err
	}
	if apiName == "" || apiVersion == "" || channel == "" || userId == "" {
		return "", "", "", errors.New("MISSING_REQUIRED_HEADERS")
	}
	if accessSecret == "" {
		return "", "", "", errors.New("INVALID_ACCESS_SECRET")
	}
	headers := map[string]string{
		"ts":    timestamp,
		"api":   apiName,
		"ver":   apiVersion,
		"chnl":  channel,
		"usrid": userId,
	}
	signedHeaders := ""
	for key, value := range headers {
		signedHeaders += key + "=" + value + "/"
	}
	payloadHash := hex.EncodeToString(sha256Hash(payload))
	computedSignature := ComputeSignature(accessSecret, payloadHash, headers)
	authHeader = buildAuthorizationHeader(accessKeyId, signedHeaders, computedSignature)
	return computedSignature, authHeader, signedHeaders, nil
}

// ParseAuthorizationHeader parses the authorization header value and returns its components.
// Format: "alg=HMAC-SHA256,creds=access-key,signed-headers=header1=value1/header2=value2/,sign=signature"
// Sample header value
// alg=HMAC-SHA256,creds=test-key-id,signed-headers=chnl=web/usrid=test-user/ts=2025-01-05T21:00:02+05:30/api=test-api/ver=v1/,sign=5b15ecf0a5a6cc14c12651f628a9bbc8958dcd8edc9bbe8e9970481bb72668af
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
	algorithm = parts["alg"]
	credentials = parts["creds"]
	signedHeaders = parts["signed-headers"]
	signature = parts["sign"]

	// Validate all required fields are present
	if algorithm == "" || credentials == "" || signedHeaders == "" || signature == "" {
		return "", "", "", "", errors.New("INVALID_AUTHORIZATION_HEADER")
	}

	if !strings.EqualFold(algorithm, "HMAC-SHA256") {
		return "", "", "", "", errors.New("INVALID_ALGORITHM")
	}

	return algorithm, credentials, signedHeaders, signature, nil
}
