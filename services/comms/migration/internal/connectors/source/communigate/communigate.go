package communigate

import (
	"era/services/comms/internal/imapclient"
	"era/services/comms/migration/internal/foldermap"
	"era/services/comms/migration/internal/importers/imap"
)

// DiscoveryResult — folder/message counts for pilot planning.
type DiscoveryResult struct {
	Folders       []FolderStat `json:"folders"`
	TotalMessages int          `json:"total_messages"`
	EstimateBytes int64        `json:"estimate_bytes"`
}

type FolderStat struct {
	Name     string `json:"name"`
	Messages int    `json:"messages"`
	MappedTo string `json:"mapped_to"`
}

// Discover lists CG mailboxes and estimates volume.
func Discover(cfg imap.NetworkConfig) (DiscoveryResult, error) {
	pass, err := cfg.ResolvePassword()
	if err != nil {
		return DiscoveryResult{}, err
	}
	port := cfg.Port
	if port == 0 {
		port = 143
	}
	cl, err := imapclient.Dial(imapclient.Config{
		Host: cfg.Host, Port: port, Username: cfg.Username, Password: pass, TLS: cfg.TLS || port == 993,
	})
	if err != nil {
		return DiscoveryResult{}, err
	}
	defer cl.Close()
	mboxes, err := cl.ListMailboxesDetailed()
	if err != nil {
		return DiscoveryResult{}, err
	}
	var out DiscoveryResult
	for _, mb := range mboxes {
		if !mb.Selectable() {
			continue
		}
		msgs, err := cl.FetchFolder(mb.Name)
		if err != nil {
			continue
		}
		var bytes int64
		for _, m := range msgs {
			bytes += int64(len(m.Raw))
		}
		out.Folders = append(out.Folders, FolderStat{
			Name:     mb.Name,
			Messages: len(msgs),
			MappedTo: foldermap.Resolve(mb, nil),
		})
		out.TotalMessages += len(msgs)
		out.EstimateBytes += bytes
	}
	return out, nil
}

// ImportFolder fetches one mapped folder from CG source.
func ImportFolder(cfg imap.NetworkConfig, folder string) ([]imapclient.Message, error) {
	return imap.ImportNetwork(cfg, folder)
}

// MapFolder translates CG folder names to IceWarp targets.
func MapFolder(name string) string {
	return foldermap.MapPath(name)
}
