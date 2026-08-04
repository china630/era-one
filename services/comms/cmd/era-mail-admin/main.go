// era-mail-admin — mailbox provisioning CLI (R-1 / R2-A).
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
)

func main() {
	api := env("ERA_MAIL_API_URL", "http://127.0.0.1:8150")
	create := flag.Bool("create-user", false, "create mailbox")
	setPass := flag.Bool("set-password", false, "update mailbox password")
	list := flag.Bool("list", false, "list mailboxes for tenant")
	email := flag.String("email", "", "mailbox email")
	password := flag.String("password", "", "password")
	tenant := flag.String("tenant", "t-demo", "tenant id")
	flag.Parse()

	switch {
	case *create:
		if *email == "" || *password == "" {
			flag.Usage()
			os.Exit(2)
		}
		createMailbox(api, *tenant, *email, *password)
	case *setPass:
		if *email == "" || *password == "" {
			flag.Usage()
			os.Exit(2)
		}
		patchMailbox(api, *email, map[string]string{"password": *password})
	case *list:
		listMailboxes(api, *tenant)
	default:
		flag.Usage()
		os.Exit(2)
	}
}

func createMailbox(api, tenant, email, password string) {
	body, _ := json.Marshal(map[string]any{
		"tenant_id":   tenant,
		"email":       email,
		"password":    password,
		"quota_bytes": 512 << 20,
	})
	resp, err := http.Post(api+"/api/v1/mailboxes", "application/json", bytes.NewReader(body))
	if err != nil {
		fail(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		failMsg("create status %d", resp.StatusCode)
	}
	fmt.Printf("mailbox %s created\n", email)
}

func patchMailbox(api, email string, fields map[string]string) {
	body, _ := json.Marshal(fields)
	req, _ := http.NewRequest(http.MethodPatch, api+"/api/v1/mailboxes/"+email, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fail(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		failMsg("patch status %d", resp.StatusCode)
	}
	fmt.Printf("mailbox %s updated\n", email)
}

func listMailboxes(api, tenant string) {
	fmt.Printf("tenant %s mailboxes (use API GET per email)\n", tenant)
	_ = api
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}

func failMsg(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

var _ = io.Discard
