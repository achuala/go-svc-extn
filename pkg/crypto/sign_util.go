package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

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
	TERMINATOR := "TERMINATOR"

	kSecret := []byte(accessSecretKey)
	kDate := HmacSha256(timeStamp, kSecret)
	kVersion := HmacSha256(apiVersion, kDate)
	kApi := HmacSha256(apiName, kVersion)
	return HmacSha256(TERMINATOR, kApi)
}

func ComputeSignature(accessSecretKey, payload string, headers map[string]string) string {
	ALGORITHM_KEY := "HMAC-SHA256"

	timestamp := headers["timestamp"]
	apiName := headers["api-name"]
	apiVersion := headers["api-version"]
	signingKey := GetSignatureKey(accessSecretKey, timestamp, apiName, apiVersion)
	payloadHash := Sha256(payload)
	channel := headers["channel"]
	userId := headers["user-id"]

	request := channel + userId + hex.EncodeToString(payloadHash)
	stringToSign := ALGORITHM_KEY + timestamp + hex.EncodeToString(Sha256(request))
	return hex.EncodeToString(HmacSha256(stringToSign, signingKey))
}
