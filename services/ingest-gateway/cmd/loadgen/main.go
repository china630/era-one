// Command loadgen — нагрузочный генератор для AC2/S1-9 (gRPC PushEvents).
package main

import (
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	erav1 "era/contracts/gen/era/v1"
	"era/services/platform/tlsutil"
	"github.com/oklog/ulid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func main() {
	addr := flag.String("addr", "localhost:50051", "ingest-gateway gRPC")
	rate := flag.Int("rate", 10_000, "events per second target")
	duration := flag.Duration("duration", 5*time.Second, "test duration")
	workers := flag.Int("workers", 8, "parallel gRPC workers")
	agents := flag.Int("agents", 1, "simulated distinct agent identities (S7-17)")
	flag.Parse()

	var creds credentials.TransportCredentials = insecure.NewCredentials()
	if tc, err := tlsutil.ClientFromEnv().Load(); err != nil {
		log.Fatalf("tls: %v", err)
	} else if tc != nil {
		creds = tc
	}
	conn, err := grpc.NewClient(*addr, grpc.WithTransportCredentials(creds))
	if err != nil {
		log.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	client := erav1.NewIngestServiceClient(conn)

	deadline := time.Now().Add(*duration)
	var sent, ok, fail atomic.Uint64
	var wg sync.WaitGroup
	var once sync.Once
	var firstErr error

	worker := func() {
		defer wg.Done()
		agentIdx := 0
		for time.Now().Before(deadline) {
			batch := makeBatch(64, *agents, agentIdx)
			agentIdx = (agentIdx + 1) % max(1, *agents)
			sent.Add(uint64(len(batch.Events)))
			rpcCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			ack, err := client.PushEvents(rpcCtx, batch)
			cancel()
			if err != nil {
				fail.Add(uint64(len(batch.Events)))
				once.Do(func() { firstErr = err })
				time.Sleep(50 * time.Millisecond)
				continue
			}
			if ack.GetStatus() == erav1.Status_STATUS_RETRY || ack.GetStatus() == erav1.Status_STATUS_REJECTED {
				fail.Add(uint64(len(batch.Events)))
				once.Do(func() { firstErr = fmt.Errorf("status=%v msg=%s", ack.GetStatus(), ack.GetMessage()) })
				continue
			}
			ok.Add(uint64(len(batch.Events)))
			// throttle: ~rate/workers events per second
			sleep := time.Second * time.Duration(len(batch.Events)) * time.Duration(*workers) / time.Duration(*rate)
			if sleep > 0 {
				time.Sleep(sleep)
			}
		}
	}

	for w := 0; w < *workers; w++ {
		wg.Add(1)
		go worker()
	}
	wg.Wait()

	elapsed := duration.Seconds()
	evps := float64(ok.Load()) / elapsed
	fmt.Printf("loadgen: sent=%d ok=%d fail=%d duration=%s ev/s=%.0f\n",
		sent.Load(), ok.Load(), fail.Load(), duration, evps)
	if firstErr != nil {
		fmt.Printf("first error: %v\n", firstErr)
	}
	if evps < float64(*rate)*0.5 {
		log.Fatalf("FAIL: throughput below 50%% of target %d ev/s (got %.0f)", *rate, evps)
	}
}

func makeBatch(n int, agents int, seq int) *erav1.EventBatch {
	if agents < 1 {
		agents = 1
	}
	agentSlot := seq % agents
	tenantID := fmt.Sprintf("tenant-load-%03d", agentSlot%1000)
	nodeID := fmt.Sprintf("node-load-%04d", agentSlot)
	agentID := fmt.Sprintf("agent-load-%04d", agentSlot)
	events := make([]*erav1.Envelope, 0, n)
	now := timestamppb.Now()
	for i := 0; i < n; i++ {
		id := newULID()
		events = append(events, &erav1.Envelope{
			SchemaVersion: "1.0.0",
			EventId:       id,
			ObservedAt:    now,
			Category:      erav1.EventCategory_EVENT_CATEGORY_PROCESS,
			Severity:      erav1.Severity_SEVERITY_MEDIUM,
			PiiSanitized:  true,
			Source: &erav1.Source{
				TenantId: tenantID,
				NodeId:   nodeID,
				Hostname: "loadgen",
				AgentId:  agentID,
				Platform: erav1.Platform_PLATFORM_WINDOWS,
			},
			Ocsf: &erav1.OcsfMeta{ClassUid: 1007, CategoryUid: 1, ActivityId: 1},
			Payload: &erav1.Envelope_Process{
				Process: &erav1.ProcessEvent{
					Action:      "create",
					Pid:         1000,
					Ppid:        4,
					ImagePath:   "C:/Windows/System32/cmd.exe",
					CommandLine: "cmd.exe /c echo load",
					User:        "pseudo:loadtest",
				},
			},
		})
	}
	bid := newULID()
	return &erav1.EventBatch{
		BatchId:         bid,
		AgentId:         agentID,
		TenantId:        tenantID,
		Events:          events,
		ProducerVersion: "loadgen",
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func newULID() []byte {
	id, err := ulid.New(ulid.Now(), rand.Reader)
	if err != nil {
		panic(err)
	}
	return id[:]
}
