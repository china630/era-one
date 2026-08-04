package netflow

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestListenUDPParsesV5(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := pc.LocalAddr().String()
	_ = pc.Close()

	t.Setenv("ERA_NETFLOW_UDP_ADDR", addr)
	got := make(chan Record, 1)
	done := make(chan struct{})
	go func() {
		_ = ListenUDP(ctx, func(rec Record) {
			select {
			case got <- rec:
			default:
			}
		})
		close(done)
	}()
	time.Sleep(80 * time.Millisecond)

	c, err := net.Dial("udp", addr)
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.Write(buildV5GoldenPacket())
	_ = c.Close()
	if err != nil {
		t.Fatal(err)
	}

	select {
	case rec := <-got:
		if rec.SrcIP == "" && rec.DstIP == "" {
			t.Fatalf("empty record: %+v", rec)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for netflow UDP record")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
	}
}
