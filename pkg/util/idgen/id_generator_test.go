package idgen_test

import (
	"testing"

	"github.com/achuala/go-svc-extn/pkg/util/idgen"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestEncode(t *testing.T) {
	u := uuid.New()
	encoded := idgen.Encode(u)
	decoded, err := idgen.DecodeToUuid(encoded)

	assert.NoError(t, err)
	assert.Equal(t, u, decoded)
}

func TestEncodeUint64(t *testing.T) {
	var value uint64 = 123456789
	encoded := idgen.EncodeUint64(value)
	decoded := idgen.DecodeToUint64(encoded)

	assert.Equal(t, value, decoded)
}

func TestDecodeToUuid(t *testing.T) {
	u := uuid.New()
	encoded := idgen.Encode(u)
	decoded, err := idgen.DecodeToUuid(encoded)

	assert.NoError(t, err)
	assert.Equal(t, u, decoded)
}

func TestDecodeToUint64(t *testing.T) {
	var value uint64 = 123456789
	encoded := idgen.EncodeUint64(value)
	decoded := idgen.DecodeToUint64(encoded)

	assert.Equal(t, value, decoded)
}

func TestNewId(t *testing.T) {
	id := idgen.NewId()
	assert.NotZero(t, id)
}

func TestNewSnowflakeId(t *testing.T) {
	id := idgen.NewSnowflakeId()

	assert.NotZero(t, id)
}
