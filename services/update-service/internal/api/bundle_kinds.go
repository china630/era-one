package api

import (
	"fmt"
	"os"

	"era/services/update-service/internal/build"
	"era/services/update-service/internal/bundle"
)

func manifestForKind(kind string) (*bundle.Manifest, error) {
	switch kind {
	case bundle.KindSigmaCorpus:
		root := os.Getenv("ERA_SIGMA_CORPUS")
		if root == "" {
			root = "../../data/sigma-corpus/curated"
		}
		return build.ScanCorpus(root)
	case bundle.KindCVEFeed:
		root := os.Getenv("ERA_CVE_FEED_DIR")
		if root == "" {
			root = "../../data/cve-feed"
		}
		return build.ScanDir(root, ".json")
	case bundle.KindConnector:
		root := os.Getenv("ERA_CONNECTOR_DIR")
		if root == "" {
			root = "../../data/connectors"
		}
		return build.ScanDir(root, ".yaml", ".yml", ".json")
	case bundle.KindAIPack:
		root := os.Getenv("ERA_AI_PACK_DIR")
		if root == "" {
			root = "../../data/ai-packs"
		}
		return build.ScanDir(root, ".yaml", ".yml", ".json")
	default:
		return nil, fmt.Errorf("unknown bundle kind: %s", kind)
	}
}

func bundleKindFromEnv() string {
	kind := os.Getenv("ERA_BUNDLE_KIND")
	if kind == "" {
		return bundle.KindSigmaCorpus
	}
	if !bundle.ValidKind(kind) {
		return bundle.KindSigmaCorpus
	}
	return kind
}
