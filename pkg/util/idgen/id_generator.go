package idgen

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"net"
	"os"
	"strings"
	"sync"

	"github.com/btcsuite/btcutil/base58"
	"github.com/google/uuid"
	"github.com/lithammer/shortuuid/v4"
	"github.com/sony/sonyflake"
)

var (
	sf   *sonyflake.Sonyflake
	once sync.Once
)

func initSonyflake() {
	once.Do(func() {
		var st sonyflake.Settings
		st.MachineID = createMachineID
		sf = sonyflake.NewSonyflake(st)
	})
}

const maxNodeID = 1023 // 2^10 - 1, adjust this based on your needs

func createMachineID() (uint16, error) {
	// Try getting machine ID from hostname, this will be unique for each machine
	// and works when running in a containerized environment
	if hostname, err := os.Hostname(); err == nil {
		h := fnv.New32()
		h.Write([]byte(hostname))
		return uint16(h.Sum32() & maxNodeID), nil
	}
	// Try getting machine ID from MAC addresses
	if interfaces, err := net.Interfaces(); err == nil {
		var sb strings.Builder
		for _, iface := range interfaces {
			mac := iface.HardwareAddr
			if len(mac) > 0 {
				sb.WriteString(fmt.Sprintf("%02x", mac))
			}
		}

		if sb.Len() > 0 {
			// Create hash from MAC addresses
			h := fnv.New32()
			h.Write([]byte(sb.String()))
			return uint16(h.Sum32() & maxNodeID), nil
		}
	}

	// Fallback to random number if MAC address method fails
	buf := make([]byte, 2)
	if _, err := rand.Read(buf); err != nil {
		return 0, err
	}
	return uint16(binary.BigEndian.Uint16(buf) & maxNodeID), nil
}

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
	initSonyflake()
	id, _ := sf.NextID()
	return id
}

// Generates a base58 encoded new ID, based on snowflake implementation
func NewSnowflakeIdEnc() string {
	id := NewSnowflakeId()
	return EncodeUint64(id)
}

func GenerateId(ctx context.Context) uint64 {
	return NewSnowflakeId()
}
