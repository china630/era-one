package netflow

import (
	"context"
	"log"
	"net"
	"os"
	"sync/atomic"
)

// UDPHandler вызывается для каждого успешно разобранного v5 flow (первый record).
type UDPHandler func(rec Record)

// ListenUDP принимает NetFlow v5 на UDP (по умолчанию :2055). ERA_NETFLOW_UDP_ADDR переопределяет.
func ListenUDP(ctx context.Context, onFlow UDPHandler) error {
	addr := os.Getenv("ERA_NETFLOW_UDP_ADDR")
	if addr == "" {
		addr = ":2055"
	}
	pc, err := net.ListenPacket("udp", addr)
	if err != nil {
		return err
	}
	log.Printf("observe netflow UDP listening %s", addr)
	go func() {
		<-ctx.Done()
		_ = pc.Close()
	}()
	buf := make([]byte, 65535)
	var nOk atomic.Uint64
	for {
		n, _, err := pc.ReadFrom(buf)
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				return err
			}
		}
		if n < 24 {
			continue
		}
		_, recs, err := ParseV5(buf[:n])
		if err != nil || len(recs) == 0 {
			continue
		}
		nOk.Add(1)
		if onFlow != nil {
			onFlow(recs[0])
		}
	}
}
