package icewarp

import (
	"era/services/comms/internal/imapclient"
	"era/services/comms/migration/internal/connectors/source/communigate"
	"era/services/comms/migration/internal/importers/imap"
)

// Discover lists IceWarp mailboxes for upsell planning (IceWarp → ERA).
func Discover(cfg imap.NetworkConfig) (communigate.DiscoveryResult, error) {
	return communigate.Discover(cfg)
}

// ImportFolder fetches folder from IceWarp source for ERA target migration.
func ImportFolder(cfg imap.NetworkConfig, folder string) ([]imapclient.Message, error) {
	return imap.ImportNetwork(cfg, folder)
}
