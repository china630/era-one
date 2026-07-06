package netflow

import (
	"encoding/binary"
	"errors"
	"net"
)

const v5HeaderSize = 24
const v5RecordSize = 48

// V5Header — NetFlow v5 export header.
type V5Header struct {
	Version          uint16
	Count            uint16
	SysUptime        uint32
	UnixSecs         uint32
	UnixNsecs        uint32
	FlowSequence     uint32
	EngineType       uint8
	EngineID         uint8
	SamplingInterval uint16
}

// ParseV5 разбирает NetFlow v5 UDP payload (header + records).
func ParseV5(pkt []byte) (V5Header, []Record, error) {
	if len(pkt) < v5HeaderSize {
		return V5Header{}, nil, errors.New("packet too short for v5 header")
	}
	h := V5Header{
		Version:          binary.BigEndian.Uint16(pkt[0:2]),
		Count:            binary.BigEndian.Uint16(pkt[2:4]),
		SysUptime:        binary.BigEndian.Uint32(pkt[4:8]),
		UnixSecs:         binary.BigEndian.Uint32(pkt[8:12]),
		UnixNsecs:        binary.BigEndian.Uint32(pkt[12:16]),
		FlowSequence:     binary.BigEndian.Uint32(pkt[16:20]),
		EngineType:       pkt[20],
		EngineID:         pkt[21],
		SamplingInterval: binary.BigEndian.Uint16(pkt[22:24]),
	}
	if h.Version != 5 {
		return h, nil, errors.New("not netflow v5")
	}
	need := v5HeaderSize + int(h.Count)*v5RecordSize
	if len(pkt) < need {
		return h, nil, errors.New("truncated v5 records")
	}
	recs := make([]Record, 0, h.Count)
	off := v5HeaderSize
	for i := uint16(0); i < h.Count; i++ {
		rec, err := parseV5Record(pkt[off : off+v5RecordSize])
		if err != nil {
			return h, recs, err
		}
		recs = append(recs, rec)
		off += v5RecordSize
	}
	return h, recs, nil
}

func parseV5Record(b []byte) (Record, error) {
	if len(b) < v5RecordSize {
		return Record{}, errors.New("record too short")
	}
	src := net.IP(b[0:4]).String()
	dst := net.IP(b[4:8]).String()
	srcPort := binary.BigEndian.Uint16(b[32:34])
	dstPort := binary.BigEndian.Uint16(b[34:36])
	protoNum := b[38]
	bytes := uint64(binary.BigEndian.Uint32(b[16:20]))
	return Record{
		SrcIP: src, DstIP: dst,
		SrcPort: uint32(srcPort), DstPort: uint32(dstPort),
		Proto: protoName(protoNum),
		Bytes: bytes,
	}, nil
}

func protoName(n uint8) string {
	switch n {
	case 6:
		return "tcp"
	case 17:
		return "udp"
	case 1:
		return "icmp"
	default:
		return "ip"
	}
}
