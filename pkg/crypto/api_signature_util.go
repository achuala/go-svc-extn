package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"

	"gorm.io/gorm"
)

type AccessSecretProvider interface {
	GetAccessSecret(accessKeyId string) (string, error)
}

type DbAccessSecretProvider struct {
	db         *gorm.DB
	accessKeys map[string]string
}

func NewDbAccessSecretProvider(db *gorm.DB) *DbAccessSecretProvider {
	return &DbAccessSecretProvider{db, make(map[string]string)}
}

func (p *DbAccessSecretProvider) GetAccessSecret(accessKeyId string) (string, error) {
	if found, ok := p.accessKeys[accessKeyId]; !ok {
		var accessSecret string
		tx := p.db.Table("api_access_keys").Where("key_id = ?", accessKeyId).Scan(&accessSecret)
		if accessSecret != "" {
			p.accessKeys[accessKeyId] = accessSecret
		}
		return accessSecret, tx.Error
	} else {
		return found, nil
	}
}

func HmacSha256(data string, key []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return h.Sum(nil)
}

func Sha256(input string) []byte {
	h := sha256.New()
	h.Write([]byte(input))
	return h.Sum(nil)
}

func GetSignatureKey(accessSecretKey, timeStamp, apiName, apiVersion string) []byte {
	TERMINATOR := "@@"

	kSecret := []byte(accessSecretKey)
	kDate := HmacSha256(timeStamp, kSecret)
	kVersion := HmacSha256(apiVersion, kDate)
	kApi := HmacSha256(apiName, kVersion)
	return HmacSha256(TERMINATOR, kApi)
}

func ComputeSignature(accessSecretKey, payload string, headers map[string]string) string {
	ALGORITHM_KEY := "HMAC-SHA256"

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

func VerifySignature(tokenHeader, securityHeader, payload string, accesSecretProvider AccessSecretProvider) error {
	// Split the token by "/"
	tokens := splitKeyValue(tokenHeader, "/", "=")

	// Split the credentials (assuming tokens["key1"] == "value1:value2")
	credentials := splitKeyValue(tokens["creds"], "\n", ":")
	accessKeyId := credentials["access-key"]
	accessSecret, err := accesSecretProvider.GetAccessSecret(accessKeyId)
	if err != nil {
		return err
	}
	// TODO: Use the access key id to get the access secret
	// Split the security header by "/"
	headers := splitKeyValue(securityHeader, "/", "=")
	providedSignature := tokens["signature"]
	computedSignature := ComputeSignature(accessSecret, payload, headers)
	if computedSignature != providedSignature {
		return errors.New("SIGNATURE_MISMATCH")
	}
	return nil
}

// splitKeyValue using built-in strings.Split and strings.SplitN
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
