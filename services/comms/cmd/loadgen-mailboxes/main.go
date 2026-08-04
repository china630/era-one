// Command loadgen-mailboxes — synthetic mailbox scale proof for Comms C-5 (F-C33).
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	addr := flag.String("addr", "http://127.0.0.1:8096", "comms-ai base URL")
	mailboxes := flag.Int("mailboxes", 60000, "synthetic mailbox count")
	workers := flag.Int("workers", 32, "parallel workers")
	quick := flag.Bool("quick", false, "CI smoke: cap mailboxes at 1000")
	logPath := flag.String("log", "", "optional report log path")
	flag.Parse()

	if *quick && *mailboxes > 1000 {
		*mailboxes = 1000
	}

	report, err := RunScaleProof(context.Background(), Config{
		Addr:      *addr,
		Mailboxes: *mailboxes,
		Workers:   *workers,
	})
	if err != nil {
		log.Fatal(err)
	}
	line := report.String()
	fmt.Print(line)
	if *logPath != "" {
		if err := os.WriteFile(*logPath, []byte(line), 0o644); err != nil {
			log.Fatal(err)
		}
	}
	if report.Errors > 0 {
		os.Exit(1)
	}
}

type Config struct {
	Addr      string
	Mailboxes int
	Workers   int
}

type Report struct {
	Mailboxes   int
	Workers     int
	DurationMs  int64
	Throughput  float64
	P50Ms       int64
	P95Ms       int64
	Errors      uint64
	HealthOK    bool
	SummaryOK   uint64
}

func (r Report) String() string {
	return fmt.Sprintf(
		"comms-scale mailboxes=%d workers=%d duration_ms=%d throughput_mb_per_s=%.2f p50_ms=%d p95_ms=%d errors=%d health_ok=%v summary_ok=%d\n",
		r.Mailboxes, r.Workers, r.DurationMs, r.Throughput, r.P50Ms, r.P95Ms, r.Errors, r.HealthOK, r.SummaryOK,
	)
}

func RunScaleProof(ctx context.Context, cfg Config) (Report, error) {
	if cfg.Workers < 1 {
		cfg.Workers = 1
	}
	if cfg.Mailboxes < 1 {
		return Report{}, fmt.Errorf("mailboxes must be >= 1")
	}

	client := &http.Client{Timeout: 10 * time.Second}
	healthOK := false
	if resp, err := client.Get(cfg.Addr + "/healthz"); err == nil {
		healthOK = resp.StatusCode == http.StatusOK
		resp.Body.Close()
	}

	start := time.Now()
	jobs := make(chan int, cfg.Workers*2)
	var wg sync.WaitGroup
	var errors, summaryOK atomic.Uint64
	latencies := make([]int64, 0, cfg.Mailboxes)
	var latMu sync.Mutex

	for w := 0; w < cfg.Workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for mb := range jobs {
				t0 := time.Now()
				if err := postSummary(ctx, client, cfg.Addr, mb); err != nil {
					errors.Add(1)
					continue
				}
				summaryOK.Add(1)
				ms := time.Since(t0).Milliseconds()
				latMu.Lock()
				latencies = append(latencies, ms)
				latMu.Unlock()
			}
		}()
	}

	for i := 0; i < cfg.Mailboxes; i++ {
		jobs <- i
	}
	close(jobs)
	wg.Wait()

	duration := time.Since(start)
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })

	p50, p95 := int64(0), int64(0)
	if n := len(latencies); n > 0 {
		p50 = latencies[n*50/100]
		p95 = latencies[n*95/100]
	}

	throughput := float64(cfg.Mailboxes) / duration.Seconds()

	return Report{
		Mailboxes:  cfg.Mailboxes,
		Workers:    cfg.Workers,
		DurationMs: duration.Milliseconds(),
		Throughput: throughput,
		P50Ms:      p50,
		P95Ms:      p95,
		Errors:     errors.Load(),
		HealthOK:   healthOK,
		SummaryOK:  summaryOK.Load(),
	}, nil
}

func postSummary(ctx context.Context, client *http.Client, base string, mailbox int) error {
	body := map[string]any{
		"tenant_id":  "t-scale",
		"mailbox_id": fmt.Sprintf("mb-%d", mailbox),
		"thread": []map[string]string{
			{"from": "user@scale.local", "subject": "scale", "body": "synthetic mailbox load"},
		},
	}
	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/api/v1/ai/summary", bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-ERA-Tenant", "t-scale")
	req.Header.Set("X-ERA-Role", "mail.user")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	return nil
}
