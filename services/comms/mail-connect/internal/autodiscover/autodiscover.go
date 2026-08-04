package autodiscover

import (
	"fmt"
	"os"
	"strings"
)

func Render(email, endpoint string) (string, error) {
	return RenderWithIMAP(email, endpoint, strings.TrimSpace(os.Getenv("ERA_CONNECT_IMAP_HOST")))
}

// RenderWithIMAP includes external IMAP host for Connect lab Autodiscover (B-CONN / RT-10).
func RenderWithIMAP(email, endpoint, imapHost string) (string, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" || !strings.Contains(email, "@") {
		return "", fmt.Errorf("invalid email")
	}
	imapBlock := ""
	if imapHost != "" {
		imapBlock = fmt.Sprintf(`
      <Protocol>
        <Type>IMAP</Type>
        <Server>%s</Server>
        <Port>143</Port>
        <SSL>off</SSL>
      </Protocol>`, imapHost)
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<Autodiscover>
  <Response>
    <Account>
      <Type>CONNECT</Type>
      <Email>%s</Email>
      <Endpoint>%s</Endpoint>%s
    </Account>
  </Response>
</Autodiscover>
`, email, endpoint, imapBlock), nil
}
