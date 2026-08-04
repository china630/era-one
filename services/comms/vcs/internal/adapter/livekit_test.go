package adapter_test

import (
	"os"
	"testing"

	"era/services/comms/vcs/internal/adapter"
)

func TestFromEnv_StubDefault(t *testing.T) {
	os.Unsetenv("ERA_LIVEKIT_URL")
	a := adapter.FromEnv()
	id, err := a.CreateRoom("demo")
	if err != nil || id == "" {
		t.Fatalf("%v %s", err, id)
	}
	tok, err := a.IssueToken("demo", "alice")
	if err != nil || tok == "" {
		t.Fatalf("%v %s", err, tok)
	}
}

func TestLiveKitHS256Token(t *testing.T) {
	l := &adapter.LiveKitHTTP{BaseURL: "http://127.0.0.1:9", APIKey: "key", APISecret: "secret"}
	tok, err := l.IssueToken("room1", "bob")
	if err != nil || tok == "" {
		t.Fatal(err)
	}
	dots := 0
	for _, c := range tok {
		if c == '.' {
			dots++
		}
	}
	if dots != 2 {
		t.Fatalf("want JWT 3 parts, got %q", tok)
	}
}
