package license

import "testing"

func sampleSignals() Signals {
	return Signals{
		MachineID:   "MID-123",
		BoardSerial: "BOARD-XYZ",
		DiskSerials: []string{"DISK-A", "DISK-B"},
		MACs:        []string{"00:11:22:33:44:55", "AA:BB:CC:DD:EE:FF"},
	}
}

func TestFingerprintDeterministicAndOrderIndependent(t *testing.T) {
	s1 := sampleSignals()
	s2 := s1
	s2.DiskSerials = []string{"DISK-B", "DISK-A"} // другой порядок
	s2.MACs = []string{"aa:bb:cc:dd:ee:ff", "00:11:22:33:44:55"}

	if ComposeFingerprint(s1, "salt") != ComposeFingerprint(s2, "salt") {
		t.Fatal("отпечаток должен быть независим от порядка и регистра")
	}
}

func TestFingerprintChangesOnDifferentHardware(t *testing.T) {
	s := sampleSignals()
	other := s
	other.BoardSerial = "BOARD-OTHER"
	if ComposeFingerprint(s, "salt") == ComposeFingerprint(other, "salt") {
		t.Fatal("отпечаток должен меняться при смене железа")
	}
}

func TestFingerprintSaltMatters(t *testing.T) {
	s := sampleSignals()
	if ComposeFingerprint(s, "salt-a") == ComposeFingerprint(s, "salt-b") {
		t.Fatal("разный salt -> разный отпечаток")
	}
}

func TestMatchTolerantSurvivesOneChange(t *testing.T) {
	baseline := sampleSignals()
	current := baseline
	current.DiskSerials = []string{"DISK-NEW", "DISK-B"} // заменили один диск

	// 5 компонентов всего; совпасть должно >= 4 (mid, board, disk-b, 2 mac).
	if !MatchTolerant(baseline, current, "salt", 4) {
		t.Fatal("толерантное сопоставление должно пережить замену одного диска")
	}
	// При строгом требовании всех 6 — не совпадёт.
	if MatchTolerant(baseline, current, "salt", 6) {
		t.Fatal("строгое требование не должно проходить при изменении")
	}
}
