package dnsx

import (
	"testing"
	"time"

	"era/services/resolve/internal/atlas"
	"era/services/resolve/internal/guard"
	"era/services/resolve/internal/policy"
	"era/services/resolve/internal/trace"
)

func TestDNSUDPRelay(t *testing.T) {
	pol := policy.NewStore()
	eng := guard.New(pol, atlas.New())
	tr := trace.New(32, nil)
	srv := &Server{Guard: eng, Trace: tr, Addr: "127.0.0.1:0"}
	if err := srv.Listen(); err != nil {
		t.Fatal(err)
	}
	defer srv.Close()
	go func() { _ = srv.serveLoop() }()
	time.Sleep(30 * time.Millisecond)
	ua := srv.LocalAddr()

	resp, err := QueryUDP(ua, "lab.malware.test", 1)
	if err != nil {
		t.Fatal(err)
	}
	if Rcode(resp) != 3 {
		t.Fatalf("want NXDOMAIN rcode=3 got %d", Rcode(resp))
	}

	resp, err = QueryUDP(ua, "x.phish.test", 1)
	if err != nil {
		t.Fatal(err)
	}
	if Rcode(resp) != 0 || !HasAnswerA(resp) {
		t.Fatalf("sinkhole rcode=%d ans=%v", Rcode(resp), HasAnswerA(resp))
	}
}
