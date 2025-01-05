package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"

	"gorm.io/gorm"
)

// AccessSecretProvider is an interface for retrieving access secrets.
// Implementations of this interface should provide a method to get an access secret
// given an access key ID.
type AccessSecretProvider interface {
	GetAccessSecret(accessKeyId string) (string, error)
}

type DbAccessSecretProvider struct {
	db         *gorm.DB
	accessKeys map[string]string
}

func NewDbAccessSecretProvider(db *gorm.DB) *DbAccessSecretProvider {
	return &DbAccessSecretProvider{db: db, accessKeys: make(map[string]string)}
}

// GetAccessSecret retrieves the access secret for a given access key ID.
// It first checks the in-memory cache, and if not found, queries the database.
// The retrieved secret is then cached for future use.
func (p *DbAccessSecretProvider) GetAccessSecret(accessKeyId string) (string, error) {
	if secret, ok := p.accessKeys[accessKeyId]; ok {
		return secret, nil
	}

	var accessSecret string
	err := p.db.Table("api_access_keys").Where("key_id = ?", accessKeyId).Pluck("secret", &accessSecret).Error
	if err != nil {
		return "", err
	}

	if accessSecret != "" {
		p.accessKeys[accessKeyId] = accessSecret
	}

	return accessSecret, nil
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
	const TERMINATOR = "@@"

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
//   - payload: The request body or payload to be signed
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
//  2. Calculate SHA256 hash of the payload
//  3. Combine channel, userId, and payload hash
//  4. Create final signature using algorithm, timestamp, and request hash
func ComputeSignature(accessSecretKey, payload string, headers map[string]string) string {
	const ALGORITHM_KEY = "HMAC-SHA256"

	timestamp := headers["ts"]
	apiName := headers["api"]
	apiVersion := headers["ver"]
	channel := headers["chnl"]
	userId := headers["usrid"]

	signingKey := getSignatureKey(accessSecretKey, timestamp, apiName, apiVersion)

	payloadHash := sha256Hash(payload)

	request := channel + userId + hex.EncodeToString(payloadHash)

	stringToSign := ALGORITHM_KEY + timestamp + hex.EncodeToString(sha256Hash(request))

	return hex.EncodeToString(hmacSha256(stringToSign, signingKey))
}

// VerifySignature validates the authenticity of a request by comparing the provided signature
// with a computed signature using the request payload and headers.
//
// Parameters:
//   - authorizationHeader: The authorization header containing algorithm, credentials, and signature
//     Format: "alg=HMAC-SHA256/creds=access-key:value/sign=signature"
//   - signedHeader: Headers used in signature computation
//     Format: "ts=timestamp/api=apiName/ver=version/chnl=channel/usrid=userId"
//   - payload: The request body or payload to verify
//   - accessSecretProvider: Interface to retrieve access secrets for signature computation
//
// Returns:
//   - bool: true if signature is valid, false otherwise
//   - error: Error if validation fails or if required parameters are missing/invalid
//
// Possible errors:
//   - INVALID_AUTHORIZATION_HEADER: If authorization header format is incorrect
//   - INVALID_ALGORITHM: If algorithm is not HMAC-SHA256
//   - INVALID_ACCESS_KEY_ID: If access key is missing
//   - SIGNATURE_MISSING: If signature is not provided
//   - INVALID_SIGNED_HEADERS: If required headers are missing
//   - SIGNATURE_MISMATCH: If computed signature doesn't match provided signature
func VerifySignature(authorizationHeader, signedHeader, payload string, accessSecretProvider AccessSecretProvider) (bool, error) {
	tokens := splitKeyValue(authorizationHeader, "/", "=")
	if len(tokens) < 3 {
		return false, errors.New("INVALID_AUTHORIZATION_HEADER")
	}
	algorithm := tokens["alg"]
	if !strings.EqualFold(algorithm, "HMAC-SHA256") {
		return false, errors.New("INVALID_ALGORITHM")
	}
	credentials := splitKeyValue(tokens["creds"], "\n", ":")
	accessKeyId := credentials["access-key"]
	if accessKeyId == "" {
		return false, errors.New("INVALID_ACCESS_KEY_ID")
	}
	accessSecret, err := accessSecretProvider.GetAccessSecret(accessKeyId)
	if err != nil {
		return false, err
	}

	providedSignature := tokens["sign"]
	if providedSignature == "" {
		return false, errors.New("SIGNATURE_MISSING")
	}
	singedHeaders := splitKeyValue(signedHeader, "/", "=")
	if len(singedHeaders) < 5 {
		return false, errors.New("INVALID_SIGNED_HEADERS")
	}
	computedSignature := ComputeSignature(accessSecret, payload, singedHeaders)
	if strings.EqualFold(computedSignature, providedSignature) {
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
