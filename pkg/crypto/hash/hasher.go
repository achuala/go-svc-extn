package hash

import (
	"context"
)

// Hasher provides methods for generating and comparing secret hashes.
type Hasher interface {
	// Generate returns a hash derived from the secret or an error if the hash method failed.
	Generate(ctx context.Context, secret []byte) ([]byte, error)

	// Understands returns whether the given hash can be understood by this hasher.
	Understands(hash []byte) bool
}

type HashProvider interface {
	Hasher(ctx context.Context) Hasher
}
