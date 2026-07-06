package netflow

import (
	"encoding/binary"
	"encoding/json"
	"flag"
	"net"
	"os"
	"testing"
)

var updateGolden = flag.Bool("update", false, "update golden files")

func buildV5GoldenPacket() []byte {
	pkt := make([]byte, v5HeaderSize+v5RecordSize)
	binary.BigEndian.PutUint16(pkt[0:2], 5)
	binary.BigEndian.PutUint16(pkt[2:4], 1)
	binary.BigEndian.PutUint32(pkt[4:8], 1000)
	binary.BigEndian.PutUint32(pkt[8:12], 1700000000)
	binary.BigEndian.PutUint32(pkt[16:20], 42)
	copy(pkt[v5HeaderSize:], buildV5Record(
		net.ParseIP("10.0.0.10").To4(),
		net.ParseIP("10.0.0.20").To4(),
		443, 52000, 6, 9000,
	))
	return pkt
}

func buildV5Record(src, dst net.IP, sport, dport uint16, proto uint8, octets uint32) []byte {
	b := make([]byte, v5RecordSize)
	copy(b[0:4], src)
	copy(b[4:8], dst)
	binary.BigEndian.PutUint32(b[16:20], octets)
	binary.BigEndian.PutUint16(b[32:34], sport)
	binary.BigEndian.PutUint16(b[34:36], dport)
	b[38] = proto
	return b
}

func TestParseV5Golden(t *testing.T) {
	pktPath := "testdata/v5_packet.golden.bin"
	wantPath := "testdata/v5_records.golden.json"
	if *updateGolden {
		pkt := buildV5GoldenPacket()
		if err := os.WriteFile(pktPath, pkt, 0o644); err != nil {
			t.Fatal(err)
		}
		_, recs, err := ParseV5(pkt)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := json.MarshalIndent(recs, "", "  ")
		if err := os.WriteFile(wantPath, b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	raw, err := os.ReadFile(pktPath)
	if err != nil {
		t.Fatalf("read golden packet: %v (run with -update)", err)
	}
	_, got, err := ParseV5(raw)
	if err != nil {
		t.Fatal(err)
	}
	wantRaw, err := os.ReadFile(wantPath)
	if err != nil {
		t.Fatalf("read golden want: %v", err)
	}
	var want []Record
	if err := json.Unmarshal(wantRaw, &want); err != nil {
		t.Fatal(err)
	}
	gb, _ := json.Marshal(got)
	wb, _ := json.Marshal(want)
	if string(gb) != string(wb) {
		t.Fatalf("golden mismatch; run with -update\n got %s\nwant %s", gb, wb)
	}
}
