package hybrid

import (
	"fmt"
	"strings"

	"era/services/control-plane/internal/store"
)

func applyBundleToPolicy(st store.Repository, bundle *bundleClaims) {
	pol := st.Policy()
	if pol.Rules == nil {
		pol.Rules = make(map[string]string)
	}
	ref := fmt.Sprintf("bundle:%s", bundle.BundleID)
	switch bundle.Kind {
	case "sigma-corpus":
		pol.Rules["sigma"] = ref
	case "cve-feed":
		pol.Rules["cve-feed"] = ref
	case "connector":
		pol.Rules["connector"] = ref
	case "ai-pack":
		pol.Rules["ai-pack"] = ref
	default:
		pol.Rules[bundle.Kind] = ref
	}
	pol.Version = bumpPolicyVersion(pol.Version, bundle.BundleID)
	st.SetPolicy(pol)
}

func bumpPolicyVersion(cur, bundleID string) string {
	if cur == "" {
		return "1.0.0+" + bundleID
	}
	parts := strings.SplitN(cur, "+", 2)
	base := parts[0]
	segs := strings.Split(base, ".")
	if len(segs) == 3 {
		var patch int
		_, _ = fmt.Sscanf(segs[2], "%d", &patch)
		return fmt.Sprintf("%s.%s.%d+%s", segs[0], segs[1], patch+1, bundleID)
	}
	return base + "+" + bundleID
}
