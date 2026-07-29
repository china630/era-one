package communigate

var cgFolderTree = []struct {
	CG     string
	Target string
}{
	{"INBOX", "INBOX"},
	{"Sent", "Sent"},
	{"#Users#demo#Sent", "Sent"},
	{"#Users#demo#INBOX", "INBOX"},
	{"Drafts", "Drafts"},
	{"Deleted Items", "Trash"},
	{"#Users#demo#Projects#Archive", "Users/demo/Projects/Archive"},
}

// GoldenMappings returns expected CG→target mapping for tests.
func GoldenMappings() map[string]string {
	out := make(map[string]string, len(cgFolderTree))
	for _, row := range cgFolderTree {
		out[row.CG] = row.Target
	}
	return out
}
