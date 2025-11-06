package hash

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSha256(t *testing.T) {
	key, err := base64.RawStdEncoding.DecodeString("voVZgo5gWk2DS+3hwm8dzk1jmPpDNa+NALaQPd4zlas")
	require.NoError(t, err)
	conf := Sha256Configuration{Key: key}
	hasher := NewHasherSha256(&conf)
	hash1, err := hasher.Generate(context.Background(), []byte("test"))
	require.NoError(t, err)
	assert.NotEqual(t, []byte("test"), hash1)
	fmt.Println(string(hash1))

	hash2, err := hasher.Generate(context.Background(), []byte("test"))
	h32 := hex.EncodeToString(hash2)
	fmt.Println(h32)
	require.NoError(t, err)
	fmt.Println(string(hash2))
	assert.NotEqual(t, []byte("test"), hash2)
}
