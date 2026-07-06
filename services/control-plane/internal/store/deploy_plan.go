package store

import (
	"strings"
)

func matchProduct(installedName, catalogProduct string) bool {
	return strings.Contains(strings.ToLower(installedName), strings.ToLower(catalogProduct))
}

func planPatchesFromSoftware(software map[string][]*AssetSoftware, catalog []PatchCatalogEntry) []PatchPlanRow {
	if len(catalog) == 0 {
		catalog = DefaultPatchCatalog()
	}
	var plan []PatchPlanRow
	for nodeID, swList := range software {
		for _, sw := range swList {
			if sw == nil {
				continue
			}
			for _, cat := range catalog {
				if matchProduct(sw.Name, cat.Product) {
					plan = append(plan, PatchPlanRow{
						NodeID: nodeID, Product: sw.Name, Version: sw.Version,
						CVEID: cat.CVEID, PackageRef: cat.PackageRef,
					})
				}
			}
		}
	}
	return plan
}
