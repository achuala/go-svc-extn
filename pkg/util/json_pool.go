package util

import (
	"bytes"
	"encoding/json"
	"slices"
	"sync"
)

const (
	initialBufferSize     = 4096  // 4KB
	maxPoolBufferSize     = 65536 // 64KB
	smallPayloadThreshold = 2048  // 2KB - use standard marshaling below this
)

// JsonPool provides a thread-safe pool of bytes.Buffer for efficient JSON operations
// to reduce heap allocations in high-traffic paths.
type JsonPool struct {
	pool sync.Pool
}

// NewJsonPool creates a new JsonPool with buffers initialized to 4KB capacity.
// Buffers will grow automatically as needed, but are reset before returning to pool.
func NewJsonPool() *JsonPool {
	return &JsonPool{
		pool: sync.Pool{
			New: func() any {
				return bytes.NewBuffer(make([]byte, 0, initialBufferSize))
			},
		},
	}
}

// Marshal marshals the given value to JSON using a pooled buffer.
// The returned byte slice is a copy and safe to use after the call.
// If the JSON is larger than the buffer capacity, the buffer will grow automatically.
func (jp *JsonPool) Marshal(v any) ([]byte, error) {
	buf := jp.pool.Get().(*bytes.Buffer)
	defer func() {
		// Only return buffers under 64KB to pool to prevent memory bloat
		// Larger buffers are discarded and will be garbage collected
		if buf.Cap() <= maxPoolBufferSize { // 64KB threshold
			buf.Reset()
			jp.pool.Put(buf)
		}
	}()

	encoder := json.NewEncoder(buf)
	if err := encoder.Encode(v); err != nil {
		return nil, err
	}

	// Remove trailing newline and clone the buffer data
	result := buf.Bytes()
	if len(result) > 0 && result[len(result)-1] == '\n' {
		result = result[:len(result)-1]
	}
	return slices.Clone(result), nil
}

// Unmarshal unmarshals JSON data into the given value.
// This method delegates directly to json.Unmarshal for optimal performance.
func (jp *JsonPool) Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

// UnmarshalString unmarshals JSON string data into the given value.
// This method delegates directly to json.Unmarshal for optimal performance.
func (jp *JsonPool) UnmarshalString(data string, v any) error {
	return json.Unmarshal([]byte(data), v)
}

// MarshalToString marshals the given value to a JSON string using a pooled buffer.
// If the JSON is larger than the buffer capacity, the buffer will grow automatically.
func (jp *JsonPool) MarshalToString(v any) (string, error) {
	buf := jp.pool.Get().(*bytes.Buffer)
	defer func() {
		// Only return buffers under 64KB to pool to prevent memory bloat
		// Larger buffers are discarded and will be garbage collected
		if buf.Cap() <= maxPoolBufferSize { // 64KB threshold
			buf.Reset()
			jp.pool.Put(buf)
		}
	}()

	encoder := json.NewEncoder(buf)
	if err := encoder.Encode(v); err != nil {
		return "", err
	}

	// Convert buffer to string and remove trailing newline added by Encode
	result := buf.String()
	if len(result) > 0 && result[len(result)-1] == '\n' {
		result = result[:len(result)-1]
	}
	return result, nil
}

// MarshalIndent marshals the given value to indented JSON using a pooled buffer.
// The returned byte slice is a copy and safe to use after the call.
// If the JSON is larger than the buffer capacity, the buffer will grow automatically.
func (jp *JsonPool) MarshalIndent(v any, prefix, indent string) ([]byte, error) {
	buf := jp.pool.Get().(*bytes.Buffer)
	defer func() {
		// Only return buffers under 64KB to pool to prevent memory bloat
		// Larger buffers are discarded and will be garbage collected
		if buf.Cap() <= maxPoolBufferSize { // 64KB threshold
			buf.Reset()
			jp.pool.Put(buf)
		}
	}()

	encoder := json.NewEncoder(buf)
	encoder.SetIndent(prefix, indent)
	if err := encoder.Encode(v); err != nil {
		return nil, err
	}

	// Remove trailing newline and clone the buffer data
	result := buf.Bytes()
	if len(result) > 0 && result[len(result)-1] == '\n' {
		result = result[:len(result)-1]
	}
	return slices.Clone(result), nil
}
