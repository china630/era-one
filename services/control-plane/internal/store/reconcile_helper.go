package store

import "strings"

func buildReconcileRows(installed, entitled map[string]int) []ReconcileRow {
	seen := map[string]struct{}{}
	var out []ReconcileRow
	for k, n := range installed {
		seen[k] = struct{}{}
		out = append(out, reconcileRow(k, n, entitled[k]))
	}
	for k, e := range entitled {
		if _, ok := seen[k]; ok {
			continue
		}
		out = append(out, reconcileRow(k, 0, e))
	}
	return out
}

func reconcileInstalledEntitled(sw []*AssetSoftware, lic []*SoftwareLicense) []ReconcileRow {
	installed := map[string]int{}
	for _, s := range sw {
		installed[strings.ToLower(s.Name)]++
	}
	entitled := map[string]int{}
	for _, l := range lic {
		entitled[strings.ToLower(l.Product)] += l.EntitledSeats
	}
	return buildReconcileRows(installed, entitled)
}
