package oidc

import "testing"

func TestHashAndVerifyPassword(t *testing.T) {
	hash, err := hashPassword("1234")
	if err != nil {
		t.Fatal(err)
	}
	if !verifyPassword(hash, "1234") {
		t.Fatal("expected password to verify")
	}
	if verifyPassword(hash, "wrong") {
		t.Fatal("expected wrong password to fail")
	}
}

func TestPgTextArray(t *testing.T) {
	got := pgTextArray([]string{"http://a/cb", "http://b/cb"})
	want := `{"http://a/cb","http://b/cb"}`
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
