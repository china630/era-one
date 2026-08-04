package repo

import "testing"

func TestArgon2HashVerify(t *testing.T) {
	hash, err := HashPassword("secret")
	if err != nil {
		t.Fatal(err)
	}
	if !VerifyPassword(hash, "secret") {
		t.Fatal("argon verify failed")
	}
	if VerifyPassword(hash, "wrong") {
		t.Fatal("wrong password accepted")
	}
}
