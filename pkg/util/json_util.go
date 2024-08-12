package util

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
)

func ConvertJsonToMap(jsonData string) (map[string]string, error) {
	result := make(map[string]string)
	err := json.Unmarshal([]byte(jsonData), &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func ConvertMapToJson(data map[string]string) ([]byte, error) {
	jb, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return jb, nil
}

// Generates unique SHA256 based unique key for the values in the map for the
// given keys
func GenerateUniqueKeyForValues(data map[string]string, keyNames []string) (string, error) {
	// get the values from the map, convert the data to hash value and send it
	var values []string
	for _, k := range keyNames {
		if v := data[k]; v == "" {
			return "", errors.New("key not found in data")
		} else {
			values = append(values, v)
		}
	}
	return hashStrings(values...), nil

}

// HashStrings creates a SHA-256 hash of the concatenated input strings
func hashStrings(data ...string) string {
	// Concatenate the strings
	combined := strings.Join(data, ",")

	// Create a new SHA-256 hash
	hasher := sha256.New()

	// Write the combined string to the hasher
	hasher.Write([]byte(combined))

	// Get the resulting hash as a byte slice
	hashBytes := hasher.Sum(nil)

	// Encode the byte slice to a hexadecimal string
	hashString := hex.EncodeToString(hashBytes)

	return hashString
}
