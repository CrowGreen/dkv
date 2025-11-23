package crypto

import (
	"bytes"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	enc, err := NewEncryptor("test-passphrase-123")
	if err != nil {
		t.Fatal(err)
	}

	original := []byte("hello world this is secret data")
	ct, err := enc.Encrypt(original)
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Equal(ct, original) {
		t.Fatal("ciphertext should differ from plaintext")
	}

	pt, err := enc.Decrypt(ct)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(pt, original) {
		t.Fatalf("expected %s, got %s", original, pt)
	}
}

func TestWrongKey(t *testing.T) {
	enc1, _ := NewEncryptor("key1")
	enc2, _ := NewEncryptor("key2")

	ct, _ := enc1.Encrypt([]byte("secret"))
	_, err := enc2.Decrypt(ct)
	if err == nil {
		t.Fatal("should fail with wrong key")
	}
}
