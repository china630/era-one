// Command mtls-api — HTTP JSON API с client mTLS (ERA_TLS_*).
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"era/services/platform/tlsutil"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: mtls-api METHOD URL [json-body]")
		os.Exit(2)
	}
	method := os.Args[1]
	url := os.Args[2]
	var body io.Reader
	if len(os.Args) > 3 {
		body = bytes.NewReader([]byte(os.Args[3]))
	}
	client, err := tlsutil.ClientFromEnv().HTTPClient(15 * time.Second)
	if err != nil {
		fmt.Fprintf(os.Stderr, "tls: %v\n", err)
		os.Exit(1)
	}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "req: %v\n", err)
		os.Exit(1)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("X-ERA-Role", "admin")
	req.Header.Set("X-ERA-Actor", "pilot-local")
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "do: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		fmt.Fprintf(os.Stderr, "status %d: %s\n", resp.StatusCode, out)
		os.Exit(1)
	}
	var pretty json.RawMessage
	if json.Unmarshal(out, &pretty) == nil {
		enc, _ := json.MarshalIndent(pretty, "", "  ")
		fmt.Println(string(enc))
	} else {
		fmt.Println(string(out))
	}
}
