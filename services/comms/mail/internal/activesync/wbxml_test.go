package activesync

import (
	"encoding/hex"
	"os"
	"testing"
)

func TestGoldenHexDump(t *testing.T) {
	if os.Getenv("DUMP_GOLDEN") == "" {
		t.Skip("set DUMP_GOLDEN=1 to dump")
	}
	reqFolder := buildFolderSyncRequest("0")
	reqSync := buildSyncRequest("0", "1")
	dumps := map[string][]byte{
		"provision": encodeProvisionResponse("1"),
		"foldersync_req": reqFolder,
		"foldersync_resp": encodeFolderSyncResponse("1"),
		"sync_req": reqSync,
		"sync_resp": encodeSyncResponse("1", 1),
		"ping": encodePingResponse(),
	}
	for name, b := range dumps {
		t.Logf("%s: %s", name, hex.EncodeToString(b))
	}
}

func buildFolderSyncRequest(syncKey string) []byte {
	var b []byte
	b = append(b, wbxmlHeader()...)
	b = append(b, switchPage(pageAirSync)...)
	b = append(b, startTag(tagFolderSync)...)
	b = append(b, startTag(tagSyncKey)...)
	b = append(b, inlineString(syncKey)...)
	b = append(b, endTag()...)
	b = append(b, endTag()...)
	return b
}

func buildSyncRequest(syncKey, collectionID string) []byte {
	var b []byte
	b = append(b, wbxmlHeader()...)
	b = append(b, switchPage(pageAirSync)...)
	b = append(b, startTag(tagSync)...)
	b = append(b, startTag(tagCollection)...)
	b = append(b, startTag(tagSyncKey)...)
	b = append(b, inlineString(syncKey)...)
	b = append(b, endTag()...)
	b = append(b, startTag(tagCollectionID)...)
	b = append(b, inlineString(collectionID)...)
	b = append(b, endTag()...)
	b = append(b, endTag()...)
	b = append(b, endTag()...)
	return b
}
