package custody

import "testing"

func TestHashChainSealVerify(t *testing.T) {
	c := NewChain()
	e1 := c.Seal([]byte("event-a"))
	if e1.PrevHash != genesis {
		t.Fatalf("prev=%s", e1.PrevHash)
	}
	if !Verify(e1.PrevHash, e1.Hash, []byte("event-a")) {
		t.Fatal("verify e1")
	}
	e2 := c.Seal([]byte("event-b"))
	if e2.PrevHash != e1.Hash {
		t.Fatalf("chain break prev=%s want=%s", e2.PrevHash, e1.Hash)
	}
	if c.Head() != e2.Hash {
		t.Fatal("head mismatch")
	}
}
