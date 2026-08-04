package lotus

import (
	"era/services/comms/migration/internal/connectors/source/communigate"
	"era/services/comms/migration/internal/importers/imap"
)

// Discover probes Domino IMAP when enabled (Phase 3 go path).
func Discover(cfg imap.NetworkConfig) (communigate.DiscoveryResult, error) {
	return communigate.Discover(cfg)
}

// MapFolder reuses CG-style mapping for Domino IMAP namespaces.
func MapFolder(name string) string {
	return communigate.MapFolder(name)
}
