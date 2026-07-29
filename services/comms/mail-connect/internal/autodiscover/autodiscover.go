package autodiscover

import (
	"fmt"
	"strings"
)

func Render(email, endpoint string) (string, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" || !strings.Contains(email, "@") {
		return "", fmt.Errorf("invalid email")
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<Autodiscover>
  <Response>
    <Account>
      <Type>CONNECT</Type>
      <Email>%s</Email>
      <Endpoint>%s</Endpoint>
    </Account>
  </Response>
</Autodiscover>
`, email, endpoint), nil
}
