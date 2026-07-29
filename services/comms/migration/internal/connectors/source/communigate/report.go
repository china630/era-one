package communigate

import (
	"encoding/csv"
	"fmt"
	"strings"

	"era/services/comms/migration/internal/importers/imap"
)

// ReportCSV formats discovery as CSV for PS/RFQ.
func ReportCSV(cfg imap.NetworkConfig) (string, error) {
	res, err := Discover(cfg)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	w := csv.NewWriter(&b)
	_ = w.Write([]string{"folder", "messages", "mapped_to"})
	for _, f := range res.Folders {
		_ = w.Write([]string{f.Name, fmt.Sprintf("%d", f.Messages), f.MappedTo})
	}
	_ = w.Write([]string{"TOTAL", fmt.Sprintf("%d", res.TotalMessages), fmt.Sprintf("%d bytes", res.EstimateBytes)})
	w.Flush()
	return b.String(), w.Error()
}
