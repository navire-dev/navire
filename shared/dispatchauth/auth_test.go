package dispatchauth

import "testing"

func TestSignAndCompare(t *testing.T) {
	secret := []byte("a-secret-that-is-long-enough-for-hmac")
	signature := Sign(secret, "dspc-1", 123, "nonce")

	if !EqualSignature(signature, Sign(secret, "dspc-1", 123, "nonce")) {
		t.Fatal("signature was not accepted")
	}
	if EqualSignature(signature, Sign(secret, "dspc-2", 123, "nonce")) {
		t.Fatal("signature accepted a different dispatcher")
	}
	if EqualSignature(signature, Sign(secret, "dspc-1", 124, "nonce")) {
		t.Fatal("signature accepted a different timestamp")
	}
}

func TestHashIsDeterministic(t *testing.T) {
	value := Hash("token")
	if value != Hash("token") {
		t.Fatal("hash is not deterministic")
	}
	if Hash("token") == Hash("other-token") {
		t.Fatal("different values have the same hash")
	}
}
