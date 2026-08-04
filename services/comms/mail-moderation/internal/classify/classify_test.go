package classify_test

import (
	"testing"

	"era/services/comms/mail-moderation/internal/classify"
)

func TestSuspicious(t *testing.T) {
	if !classify.Suspicious("Please buy gift card urgently") {
		t.Fatal("expected hit")
	}
	if classify.Suspicious("Weekly status update") {
		t.Fatal("expected miss")
	}
}
