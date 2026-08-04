package mail

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// DocumentsClient builds workspace deep links to ERA Documents.
type DocumentsClient struct {
	WorkspaceBaseURL string
	IntentSecret     []byte
	LicenseOK        bool
}

// NewDocumentsClient creates a deep-link helper for Documents UI.
func NewDocumentsClient(workspaceBase string) *DocumentsClient {
	secret := os.Getenv("ERA_DOCS_INTENT_SECRET")
	if secret == "" {
		secret = os.Getenv("ERA_IDENTITY_JWT_SECRET")
	}
	if secret == "" {
		secret = "dev-only-change-in-prod"
	}
	lic := os.Getenv("ERA_LICENSE_OFFICE_DOCUMENTS") == "1" ||
		os.Getenv("ERA_LICENSE_OFFICE_DOCUMENTS") == "true" ||
		os.Getenv("ERA_OFFICE_DEV") == "1"
	return &DocumentsClient{
		WorkspaceBaseURL: strings.TrimRight(workspaceBase, "/"),
		IntentSecret:     []byte(secret),
		LicenseOK:        lic,
	}
}

// EditLink returns signed /docs/{driveObjectID}?intent=... URL for «Edit in Documents».
func (c *DocumentsClient) EditLink(driveObjectID string) (string, error) {
	if c == nil || c.WorkspaceBaseURL == "" {
		return "", fmt.Errorf("documents: client not configured")
	}
	if !c.LicenseOK {
		return "", fmt.Errorf("documents: office-documents license required")
	}
	if driveObjectID == "" {
		return "", fmt.Errorf("documents: object id required")
	}
	exp := time.Now().Add(15 * time.Minute).Unix()
	sig := signIntent(c.IntentSecret, driveObjectID, exp)
	u, err := url.Parse(c.WorkspaceBaseURL + "/docs/" + url.PathEscape(driveObjectID))
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("intent_exp", strconv.FormatInt(exp, 10))
	q.Set("intent_sig", sig)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// VerifyIntent checks HMAC intent query params (Scaffold AC-O8).
func VerifyIntent(secret []byte, objectID, expStr, sig string) bool {
	exp, err := strconv.ParseInt(expStr, 10, 64)
	if err != nil || time.Now().Unix() > exp {
		return false
	}
	want := signIntent(secret, objectID, exp)
	return hmac.Equal([]byte(want), []byte(sig))
}

func signIntent(secret []byte, objectID string, exp int64) string {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(objectID))
	_, _ = mac.Write([]byte("|"))
	_, _ = mac.Write([]byte(strconv.FormatInt(exp, 10)))
	sum := mac.Sum(nil)
	return hex.EncodeToString(sum)
}

// SignedIntentB64 is for tests (optional opaque blob).
func SignedIntentB64(secret []byte, objectID string, exp int64) string {
	return base64.RawURLEncoding.EncodeToString([]byte(signIntent(secret, objectID, exp)))
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
