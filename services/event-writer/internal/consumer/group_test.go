package consumer

import (
	"testing"

	"era/services/event-writer/internal/chwriter"
)

// C-05: два writer-инстанса используют один consumer group — rebalance без дублей на уровне конфигурации.
func TestSharedConsumerGroupID(t *testing.T) {
	w := &chwriter.Writer{}
	r1 := New([]string{"127.0.0.1:9092"}, "era-event-writer", w)
	r2 := New([]string{"127.0.0.1:9092"}, "era-event-writer", w)
	if r1.groupID != r2.groupID || r1.groupID != "era-event-writer" {
		t.Fatalf("group mismatch: %q vs %q", r1.groupID, r2.groupID)
	}
	if len(r1.readers) != len(defaultTopics) {
		t.Fatalf("expected %d topic readers, got %d", len(defaultTopics), len(r1.readers))
	}
	for i, rd := range r1.readers {
		if rd.Config().GroupID != "era-event-writer" {
			t.Fatalf("reader %d group=%s", i, rd.Config().GroupID)
		}
	}
}

func TestDistinctGroupsDoNotShare(t *testing.T) {
	w := &chwriter.Writer{}
	a := New([]string{"127.0.0.1:9092"}, "era-writer-a", w)
	b := New([]string{"127.0.0.1:9092"}, "era-writer-b", w)
	if a.groupID == b.groupID {
		t.Fatal("distinct groups required for duplicate consumption test isolation")
	}
}
