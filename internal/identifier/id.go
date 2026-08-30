// Package identifier creates sortable UUIDv7 identifiers shared by in-memory
// and PostgreSQL repositories.
package identifier

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/google/uuid"
)

func New() string {
	var value [16]byte
	millis := uint64(time.Now().UTC().UnixMilli())
	value[0], value[1], value[2] = byte(millis>>40), byte(millis>>32), byte(millis>>24)
	value[3], value[4], value[5] = byte(millis>>16), byte(millis>>8), byte(millis)
	if _, err := rand.Read(value[6:]); err != nil {
		binary.BigEndian.PutUint64(value[8:], uint64(time.Now().UTC().UnixNano()))
	}
	value[6] = (value[6] & 0x0f) | 0x70
	value[8] = (value[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", value[0:4], value[4:6], value[6:8], value[8:10], value[10:])
}

func FromKey(scope, value string) string {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(scope+"\x00"+value)).String()
}
