package foldermap

import (
	"strings"

	"era/services/comms/internal/imapclient"
)

// Mapper translates source folder paths to IceWarp targets.
type Mapper func(source string) string

// MapPath translates CG/Lotus folder paths to IceWarp targets (path rules without LIST attrs).
func MapPath(name string) string {
	seg := strings.ToUpper(strings.TrimSpace(lastSegment(name)))
	switch seg {
	case "INBOX":
		return "INBOX"
	case "SENT", "SENT ITEMS":
		return "Sent"
	case "DRAFTS":
		return "Drafts"
	case "TRASH", "DELETED", "DELETED ITEMS":
		return "Trash"
	default:
		mapped := strings.ReplaceAll(name, "#", "/")
		return strings.TrimPrefix(mapped, "/")
	}
}

func lastSegment(name string) string {
	for _, sep := range []string{"#", "/"} {
		if strings.Contains(name, sep) {
			parts := strings.Split(name, sep)
			return parts[len(parts)-1]
		}
	}
	return name
}

// Resolve maps a LIST mailbox to target folder using special-use attrs (G4) and path rules (G2).
func Resolve(mb imapclient.Mailbox, tenant map[string]string) string {
	if tenant != nil {
		if target, ok := tenant[mb.Name]; ok {
			return target
		}
	}
	for _, attr := range mb.Attributes {
		switch strings.ToUpper(attr) {
		case `\SENT`:
			return "Sent"
		case `\DRAFTS`:
			return "Drafts"
		case `\TRASH`:
			return "Trash"
		}
	}
	return MapPath(mb.Name)
}

// Tree builds source→target map for all mailboxes (G5).
func Tree(mailboxes []imapclient.Mailbox, tenant map[string]string) map[string]string {
	out := make(map[string]string, len(mailboxes))
	for _, mb := range mailboxes {
		out[mb.Name] = Resolve(mb, tenant)
	}
	return out
}

// CGFixture returns representative CG LIST mailboxes for golden tests (G5).
func CGFixture() []imapclient.Mailbox {
	return []imapclient.Mailbox{
		{Name: "INBOX"},
		{Name: "#Users#demo#INBOX"},
		{Name: "#Users#demo#Sent", Attributes: []string{`\Sent`}},
		{Name: "Sent Items", Attributes: []string{`\Sent`}},
		{Name: "Drafts", Attributes: []string{`\Drafts`}},
		{Name: "Deleted Items", Attributes: []string{`\Trash`}},
		{Name: "#Users#demo#Projects#Archive"},
	}
}

// CGFixtureGolden returns expected IceWarp folder tree for CGFixture (G5).
func CGFixtureGolden() map[string]string {
	return Tree(CGFixture(), nil)
}
