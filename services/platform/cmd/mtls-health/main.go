// Command mtls-health — GET с client mTLS (ERA_TLS_*), для pilot/loadgen на Windows.
package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"era/services/platform/tlsutil"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: mtls-health <url>")
		os.Exit(2)
	}
	client, err := tlsutil.ClientFromEnv().HTTPClient(5 * time.Second)
	if err != nil {
		fmt.Fprintf(os.Stderr, "tls: %v\n", err)
		os.Exit(1)
	}
	resp, err := client.Get(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "get: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "status %d: %s\n", resp.StatusCode, body)
		os.Exit(1)
	}
	fmt.Print(string(body))
}
