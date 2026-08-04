package activesync

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

// AirSync WBXML code pages (MS-ASWBXML subset).
const (
	pageAirSync   = 0x00
	pageProvision = 0x01
	pagePing      = 0x02

	tagSyncKey     = 0x0B
	tagStatus      = 0x0E
	tagChanges     = 0x0F
	tagAdd         = 0x10
	tagServerID    = 0x11
	tagParentID    = 0x12
	tagDisplayName = 0x13
	tagType        = 0x14
	tagFolderSync  = 0x16
	tagSync        = 0x05
	tagCollection  = 0x1F
	tagCollectionID = 0x12
	tagResponses   = 0x06
	tagProvision   = 0x0E
	tagPolicyKey   = 0x0C
	tagPing        = 0x0C
)

func wbxmlHeader() []byte {
	return []byte{0x03, 0x01, 0x6A, 0x00}
}

func switchPage(page byte) []byte {
	return []byte{0x00, page}
}

func startTag(tag byte) []byte {
	return []byte{0x40 | tag}
}

func endTag() []byte {
	return []byte{0x01}
}

func inlineString(s string) []byte {
	b := []byte{0x03}
	b = append(b, []byte(s)...)
	b = append(b, 0x00)
	return b
}

func inlineInt(n int) []byte {
	return inlineString(strconv.Itoa(n))
}

// decodeStrings extracts inline STR_I values from WBXML payload.
func decodeStrings(body []byte) []string {
	var out []string
	for i := 0; i < len(body); i++ {
		if body[i] != 0x03 {
			continue
		}
		j := i + 1
		for j < len(body) && body[j] != 0x00 {
			j++
		}
		if j >= len(body) {
			break
		}
		out = append(out, string(body[i+1:j]))
		i = j
	}
	return out
}

func parseBody(data []byte) ([]byte, error) {
	if len(data) < 4 {
		return nil, fmt.Errorf("wbxml: short header")
	}
	if data[0] != 0x03 {
		return nil, fmt.Errorf("wbxml: bad version")
	}
	return data[4:], nil
}

func firstString(data []byte) string {
	strs := decodeStrings(data)
	if len(strs) == 0 {
		return ""
	}
	return strs[0]
}

func encodeProvisionResponse(policyKey string) []byte {
	if policyKey == "" {
		policyKey = "1"
	}
	var b bytes.Buffer
	b.Write(wbxmlHeader())
	b.Write(switchPage(pageProvision))
	b.Write(startTag(tagProvision))
	b.Write(startTag(tagStatus))
	b.Write(inlineInt(1))
	b.Write(endTag())
	b.Write(startTag(tagPolicyKey))
	b.Write(inlineString(policyKey))
	b.Write(endTag())
	b.Write(endTag())
	return b.Bytes()
}

func encodeFolderSyncResponse(newSyncKey string) []byte {
	var b bytes.Buffer
	b.Write(wbxmlHeader())
	b.Write(switchPage(pageAirSync))
	b.Write(startTag(tagFolderSync))
	b.Write(startTag(tagStatus))
	b.Write(inlineInt(1))
	b.Write(endTag())
	b.Write(startTag(tagSyncKey))
	b.Write(inlineString(newSyncKey))
	b.Write(endTag())
	b.Write(startTag(tagChanges))
	writeFolderAdd(&b, "1", "Inbox", 2)
	writeFolderAdd(&b, "2", "Calendar", 8)
	writeFolderAdd(&b, "3", "Contacts", 9)
	b.Write(endTag())
	b.Write(endTag())
	return b.Bytes()
}

func writeFolderAdd(b *bytes.Buffer, id, name string, typ int) {
	b.Write(startTag(tagAdd))
	b.Write(startTag(tagServerID))
	b.Write(inlineString(id))
	b.Write(endTag())
	b.Write(startTag(tagParentID))
	b.Write(inlineString("0"))
	b.Write(endTag())
	b.Write(startTag(tagDisplayName))
	b.Write(inlineString(name))
	b.Write(endTag())
	b.Write(startTag(tagType))
	b.Write(inlineInt(typ))
	b.Write(endTag())
	b.Write(endTag())
}

func encodeSyncResponse(newSyncKey string, changeCount int) []byte {
	var b bytes.Buffer
	b.Write(wbxmlHeader())
	b.Write(switchPage(pageAirSync))
	b.Write(startTag(tagSync))
	b.Write(startTag(tagStatus))
	b.Write(inlineInt(1))
	b.Write(endTag())
	b.Write(startTag(tagResponses))
	b.Write(startTag(tagCollection))
	b.Write(startTag(tagStatus))
	b.Write(inlineInt(1))
	b.Write(endTag())
	b.Write(startTag(tagSyncKey))
	b.Write(inlineString(newSyncKey))
	b.Write(endTag())
	if changeCount > 0 {
		b.Write(startTag(tagChanges))
		b.Write(startTag(tagAdd))
		b.Write(startTag(tagServerID))
		b.Write(inlineString("1"))
		b.Write(endTag())
		b.Write(endTag())
		b.Write(endTag())
	}
	b.Write(endTag())
	b.Write(endTag())
	b.Write(endTag())
	return b.Bytes()
}

func encodePingResponse() []byte {
	var b bytes.Buffer
	b.Write(wbxmlHeader())
	b.Write(switchPage(pagePing))
	b.Write(startTag(tagPing))
	b.Write(startTag(tagStatus))
	b.Write(inlineInt(1))
	b.Write(endTag())
	b.Write(endTag())
	return b.Bytes()
}

func nextSyncKey(prev string) string {
	if prev == "" || prev == "0" {
		return "1"
	}
	if n, err := strconv.Atoi(prev); err == nil {
		return strconv.Itoa(n + 1)
	}
	return prev + "-next"
}

func folderIDFromRequest(body []byte) string {
	strs := decodeStrings(body)
	if len(strs) >= 2 {
		return strs[1]
	}
	if len(strs) == 1 {
		return "1"
	}
	return "1"
}

func syncKeyFromRequest(body []byte) string {
	return firstString(body)
}

func normalizeDeviceID(deviceID string) string {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return "era-device-1"
	}
	return deviceID
}
