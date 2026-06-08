package migrations

import (
	"crypto/sha256"
	"fmt"
)

// Checksum returns a stable SHA-256 based checksum for a migration.
// It is derived from Version and Name, making it independent of file location.
func Checksum(m Migration) string {
	h := sha256.New()
	h.Write([]byte(m.Version()))
	h.Write([]byte{0})
	h.Write([]byte(m.Name()))
	return fmt.Sprintf("%x", h.Sum(nil))
}
