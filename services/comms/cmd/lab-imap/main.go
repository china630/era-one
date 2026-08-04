// Command lab-imap — RFC3501 lab IMAP for Migration/Connect (compose service dovecot-lab).
// Air-gap: no external image; seeds lab1@mail.gov.az INBOX for L-1 smoke.
package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"era/services/comms/internal/imapclient"
)

func main() {
	addr := os.Getenv("ERA_LAB_IMAP_ADDR")
	if addr == "" {
		addr = ":143"
	}
	seed := []byte("From: lab1@mail.gov.az\r\nTo: lab1@mail.gov.az\r\nSubject: lab-seed\r\n\r\nLab IMAP seed body\r\n")
	folders := map[string][]imapclient.SeedMessage{
		"INBOX": {{Raw: seed}},
		"Sent":  {},
	}
	bound, stop, err := imapclient.StartLabServer(addr, folders, map[string][]string{
		"INBOX": {`\HasNoChildren`},
		"Sent":  {`\Sent`, `\HasNoChildren`},
	})
	if err != nil {
		log.Fatal(err)
	}
	defer stop()
	log.Printf("dovecot-lab (era lab-imap) listening %s", bound)

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	fmt.Println("lab-imap shutdown")
}
