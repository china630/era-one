// Package taxii — исходящий псевдонимизированный экспорт TI (L-02).
package taxii

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"era/services/national-hub/internal/hub"
	"era/services/national-hub/internal/sanitize"
	"era/services/national-hub/internal/stix"
	"era/services/platform/licensegate"
)

// OutboundExporter — псевдонимизированный экспорт IoC для hybrid-пилота.
type OutboundExporter struct {
	Store hub.ObjectStore
	Gate  *licensegate.Gate
	Salt  string
}

func NewOutbound(st hub.ObjectStore, gate *licensegate.Gate, salt string) *OutboundExporter {
	if salt == "" {
		salt = "era-national-export"
	}
	return &OutboundExporter{Store: st, Gate: gate, Salt: salt}
}

func (o *OutboundExporter) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/taxii2/api1/outbound/export", o.handleExport)
	return mux
}

func (o *OutboundExporter) handleExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !o.Gate.Allow(licensegate.ModuleNational) {
		http.Error(w, "module national not licensed", http.StatusForbidden)
		return
	}
	objs := o.Store.Poll(hub.DefaultCollection)
	bundle, err := o.pseudonymizeBundle(objs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	raw, err := bundle.JSON()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := sanitize.AuditBundle(raw); err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"bundle":      json.RawMessage(raw),
		"object_count": len(bundle.Objects),
		"exported_at": time.Now().UTC(),
	})
}

func (o *OutboundExporter) pseudonymizeBundle(objs []hub.Object) (*stix.Bundle, error) {
	now := time.Now().UTC()
	out := &stix.Bundle{
		Type: stix.BundleType, ID: "bundle--export", SpecVersion: stix.SpecVersion,
	}
	for _, obj := range objs {
		var src stix.Bundle
		if err := json.Unmarshal([]byte(obj.RawJSON), &src); err != nil {
			continue
		}
		for _, ind := range src.Objects {
			pseudoOrg := pseudoID(o.Salt, obj.OrgID)
			ind.Name = strings.TrimSpace(ind.Name)
			if ind.Name == "" {
				ind.Name = "indicator"
			}
			ind.Name = pseudoOrg + ":" + ind.Name
			ind.Labels = append(ind.Labels, "pseudonymized", pseudoOrg)
			ind.Created = now
			ind.Modified = now
			ind.ValidFrom = now
			out.Objects = append(out.Objects, ind)
		}
	}
	return out, nil
}

func pseudoID(salt, orgID string) string {
	h := sha256.Sum256([]byte(salt + ":" + orgID))
	return "org-" + hex.EncodeToString(h[:8])
}
