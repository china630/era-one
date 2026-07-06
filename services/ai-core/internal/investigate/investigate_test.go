package investigate

import "testing"

func TestIsSuspicious(t *testing.T) {
	if !isSuspicious(`{"command_line":"powershell -enc ABC"}`) {
		t.Fatal("expected suspicious")
	}
	if isSuspicious(`{"image_path":"notepad.exe"}`) {
		t.Fatal("expected benign")
	}
}

func TestSummarize(t *testing.T) {
	s := summarize("process", `{"image_path":"cmd.exe"}`)
	if s == "" {
		t.Fatal("empty summary")
	}
}
