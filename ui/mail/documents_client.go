package mail

import (
	"fmt"
	"net/url"
	"strings"
)

// DocumentsClient builds workspace deep links to ERA Documents.
type DocumentsClient struct {
	WorkspaceBaseURL string
}

// NewDocumentsClient creates a deep-link helper for Documents UI.
func NewDocumentsClient(workspaceBase string) *DocumentsClient {
	return &DocumentsClient{WorkspaceBaseURL: strings.TrimRight(workspaceBase, "/")}
}

// EditLink returns /docs/{driveObjectID} URL for «Edit in Documents».
func (c *DocumentsClient) EditLink(driveObjectID string) (string, error) {
	if c == nil || c.WorkspaceBaseURL == "" {
		return "", fmt.Errorf("documents: client not configured")
	}
	if driveObjectID == "" {
		return "", fmt.Errorf("documents: object id required")
	}
	u, err := url.Parse(c.WorkspaceBaseURL + "/docs/" + driveObjectID)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

// IsEradOrDocx reports whether attachment can open in Documents.
func IsEradOrDocx(filename, contentType string) bool {
	lower := strings.ToLower(filename)
	if strings.HasSuffix(lower, ".erad") || strings.HasSuffix(lower, ".docx") {
		return true
	}
	ct := strings.ToLower(contentType)
	return ct == "application/vnd.era.erad" ||
		ct == "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
}
