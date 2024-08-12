package idgen

import (
	"context"
	"encoding/binary"

	"github.com/btcsuite/btcutil/base58"
	"github.com/godruoyi/go-snowflake"
	"github.com/google/uuid"
	"github.com/lithammer/shortuuid/v4"
)

// Encodes the given UUID to base58
func Encode(u uuid.UUID) string {
	return base58.Encode(u[:])
}

// Encodes the given uint64 using base58
func EncodeUint64(v uint64) string {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v)
	return base58.Encode(b)
}

// Decodes the given base58 encoded string to UUID
func DecodeToUuid(s string) (uuid.UUID, error) {
	return uuid.FromBytes(base58.Decode(s))
}

// Decodes the given base58 encoded data to Uint64
func DecodeToUint64(s string) uint64 {
	return binary.BigEndian.Uint64(base58.Decode(s))
}

// Generates a new ID, based on short UUID.
func NewId() string {
	return shortuuid.New()
}

// Generates a new ID, based on snowflake implementation.
func NewSnowflakeId() uint64 {
	return snowflake.ID()
}

// Generates a base58 encoded new ID, based on snowflake implementation
func NewSnowflakeIdEnc() string {
	id := NewSnowflakeId()
	return EncodeUint64(id)
}

func GenerateId(ctx context.Context) uint64 {
	return NewSnowflakeId()
}
