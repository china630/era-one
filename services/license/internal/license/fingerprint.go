package license

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
)

// Signals — стабильные признаки инсталляции для отпечатка развёртывания (ADR-0010 §9).
// Сбор признаков платформозависим и выполняется control-plane.
type Signals struct {
	MachineID   string
	BoardSerial string
	DiskSerials []string
	MACs        []string
}

func normSlice(ss []string) []string {
	out := make([]string, 0, len(ss))
	for _, s := range ss {
		s = strings.ToLower(strings.TrimSpace(s))
		if s != "" {
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return out
}

// ComposeFingerprint строит строгий deployment ID для привязки лицензии.
// Порядок дисков/MAC не влияет на результат (нормализация + сортировка).
func ComposeFingerprint(s Signals, salt string) string {
	h := sha256.New()
	h.Write([]byte(salt))
	h.Write([]byte("|mid:" + strings.ToLower(strings.TrimSpace(s.MachineID))))
	h.Write([]byte("|board:" + strings.ToLower(strings.TrimSpace(s.BoardSerial))))
	for _, d := range normSlice(s.DiskSerials) {
		h.Write([]byte("|disk:" + d))
	}
	for _, m := range normSlice(s.MACs) {
		h.Write([]byte("|mac:" + m))
	}
	return "deploy-" + hex.EncodeToString(h.Sum(nil))[:24]
}

// componentHashes — хеши отдельных признаков для толерантного сопоставления.
func componentHashes(s Signals, salt string) []string {
	var comps []string
	add := func(prefix, v string) {
		v = strings.ToLower(strings.TrimSpace(v))
		if v == "" {
			return
		}
		sum := sha256.Sum256([]byte(salt + "|" + prefix + ":" + v))
		comps = append(comps, hex.EncodeToString(sum[:])[:16])
	}
	add("mid", s.MachineID)
	add("board", s.BoardSerial)
	for _, d := range s.DiskSerials {
		add("disk", d)
	}
	for _, m := range s.MACs {
		add("mac", m)
	}
	return comps
}

// MatchTolerant сравнивает наборы признаков; true, если совпало >= minMatch
// компонентов. Защищает лицензию от «слёта» при штатной замене диска/сетевой карты.
func MatchTolerant(baseline, current Signals, salt string, minMatch int) bool {
	set := map[string]bool{}
	for _, c := range componentHashes(baseline, salt) {
		set[c] = true
	}
	matched := 0
	for _, c := range componentHashes(current, salt) {
		if set[c] {
			matched++
		}
	}
	return matched >= minMatch
}
