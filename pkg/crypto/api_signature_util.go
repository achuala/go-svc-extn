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
func HmacSha256(data string, key []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return h.Sum(nil)
}

// Sha256 computes the SHA256 hash of the input string.
// It returns the resulting hash as a byte slice.
func Sha256(input string) []byte {
	h := sha256.New()
	h.Write([]byte(input))
	return h.Sum(nil)
}

// GetSignatureKey generates a signature key using the provided parameters.
// It combines the access secret key, timestamp, API name, and API version
// to create a unique signature key.
func GetSignatureKey(accessSecretKey, timeStamp, apiName, apiVersion string) []byte {
	const TERMINATOR = "@@"

	kSecret := []byte(accessSecretKey)
	kDate := HmacSha256(timeStamp, kSecret)
	kVersion := HmacSha256(apiVersion, kDate)
	kApi := HmacSha256(apiName, kVersion)
	return HmacSha256(TERMINATOR, kApi)
}

// ComputeSignature generates a signature for the given payload and headers.
// It uses the access secret key, timestamp, API name, and API version
// to compute a unique signature.
// The computed signature is then returned as a string.
func ComputeSignature(accessSecretKey, payload string, headers map[string]string) string {
	const ALGORITHM_KEY = "HMAC-SHA256"

	timestamp := headers["ts"]
	apiName := headers["api"]
	apiVersion := headers["ver"]
	channel := headers["chnl"]
	userId := headers["usrid"]

	signingKey := GetSignatureKey(accessSecretKey, timestamp, apiName, apiVersion)

	payloadHash := Sha256(payload)

	request := channel + userId + hex.EncodeToString(payloadHash)

	stringToSign := ALGORITHM_KEY + timestamp + hex.EncodeToString(Sha256(request))

	return hex.EncodeToString(HmacSha256(stringToSign, signingKey))
}

// VerifySignature verifies the signature of the given payload and headers.
// It uses the access secret key, timestamp, API name, and API version
// to compute a unique signature and compare it with the provided signature.
func VerifySignature(tokenHeader, securityHeader, payload string, accessSecretProvider AccessSecretProvider) error {
	tokens := splitKeyValue(tokenHeader, "/", "=")

	credentials := splitKeyValue(tokens["creds"], "\n", ":")
	accessKeyId := credentials["access-key"]
	accessSecret, err := accessSecretProvider.GetAccessSecret(accessKeyId)
	if err != nil {
		return err
	}

	headers := splitKeyValue(securityHeader, "/", "=")
	providedSignature := tokens["signature"]
	computedSignature := ComputeSignature(accessSecret, payload, headers)
	if computedSignature != providedSignature {
		return errors.New("SIGNATURE_MISMATCH")
	}
	return nil
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
