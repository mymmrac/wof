package id

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/sony/sonyflake"
)

var flake *sonyflake.Sonyflake //nolint:gochecknoglobals

func init() { //nolint:gochecknoinits
	var flakeErr error
	flake, flakeErr = sonyflake.New(sonyflake.Settings{
		StartTime: time.Date(2026, 4, 28, 0, 0, 0, 0, time.UTC),
		MachineID: func() (uint16, error) {
			hostname, err := os.Hostname()
			if err != nil {
				return 0, fmt.Errorf("get hostname: %w", err)
			}

			var mac string
			interfaces, err := net.Interfaces()
			if err == nil {
				for _, i := range interfaces {
					if len(i.HardwareAddr) > 0 {
						mac = i.HardwareAddr.String()
						break
					}
				}
			}
			if mac == "" {
				mac = rand.Text()
			}

			pidBuf := make([]byte, binary.MaxVarintLen64)
			binary.PutVarint(pidBuf, int64(os.Getppid()))

			hash := sha256.New()
			_, _ = hash.Write([]byte(mac))
			_, _ = hash.Write([]byte(hostname))
			_, _ = hash.Write(pidBuf)
			sum := hash.Sum(nil)

			return binary.BigEndian.Uint16(sum[:2]), nil
		},
	})
	if flakeErr != nil {
		panic(fmt.Errorf("create sonyflake: %w", flakeErr))
	}
}

// ID is a unique identifier.
type ID int64 //nolint:recvcheck

// New returns a unique ID.
func New() ID {
	id, err := flake.NextID()
	if err != nil {
		panic(fmt.Errorf("unreacheble: %w", err))
	}
	return ID(id & math.MaxInt64) // Ensure the ID is not negative
}

// Parse returns a unique ID from a string representation.
func Parse(value string) (ID, error) {
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse id: %w", err)
	}
	if id <= 0 {
		return 0, fmt.Errorf("invalid id value: %d", id)
	}
	return ID(id), nil
}

// String returns a string representation of a unique ID.
func (i ID) String() string {
	return strconv.FormatInt(int64(i), 10)
}

// MarshalText implements encoding.TextMarshaler.
func (i ID) MarshalText() ([]byte, error) {
	return []byte(i.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (i *ID) UnmarshalText(data []byte) error {
	id, err := Parse(string(data))
	if err != nil {
		return err
	}
	*i = id
	return nil
}
