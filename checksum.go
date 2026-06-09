package migrations

import (
	"crypto/sha256"
	"fmt"
)

// Checksum returns a stable checksum for a migration.
// If m implements ChecksumMigration and its Checksum() returns a non-empty string,
// that value is used directly. Otherwise sha256(version + "\x00" + name) is used.
func Checksum(m Migration) string {
	if cm, ok := m.(ChecksumMigration); ok {
		if cs := cm.Checksum(); cs != "" {
			return cs
		}
	}
	h := sha256.New()
	h.Write([]byte(m.Version()))
	h.Write([]byte{0})
	h.Write([]byte(m.Name()))
	return fmt.Sprintf("%x", h.Sum(nil))
}
