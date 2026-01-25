package store

import (
	"os"
	"testing"

	"github.com/CrowGreen/dkv/internal/crypto"
	"github.com/CrowGreen/dkv/internal/wal"
)

func TestWALRecovery(t *testing.T) {
	dir := "/tmp/dkv_test_integration"
	os.MkdirAll(dir, 0755)
	defer os.RemoveAll(dir)

	w, err := wal.Open(dir + "/wal.log")
	if err != nil {
		t.Fatal(err)
	}

	// simulate writes
	w.Append(wal.Entry{Op: wal.OpPut, Key: "x", Value: []byte("1")})
	w.Append(wal.Entry{Op: wal.OpPut, Key: "y", Value: []byte("2")})
	w.Append(wal.Entry{Op: wal.OpPut, Key: "x", Value: []byte("3")})
	w.Append(wal.Entry{Op: wal.OpDelete, Key: "y"})
	w.Close()

	// "crash" and recover
	s := New()
	w2, _ := wal.Open(dir + "/wal.log")
	defer w2.Close()

	entries, _ := w2.Replay()
	for _, e := range entries {
		switch e.Op {
		case wal.OpPut:
			s.Put(e.Key, e.Value)
		case wal.OpDelete:
			s.Delete(e.Key)
		}
	}

	// verify state
	e, ok := s.Get("x")
	if !ok || string(e.Value) != "3" {
		t.Fatal("x should be 3 after replay")
	}
	_, ok = s.Get("y")
	if ok {
		t.Fatal("y should be deleted")
	}
}

func TestSnapshotEncryption(t *testing.T) {
	dir := "/tmp/dkv_test_snap"
	os.MkdirAll(dir, 0755)
	defer os.RemoveAll(dir)

	enc, _ := crypto.NewEncryptor("test-key")

	s1 := New()
	s1.Put("secret", []byte("classified"))
	s1.Put("data", []byte("important"))

	path := dir + "/snap.enc"
	if err := s1.SaveSnapshot(path, enc); err != nil {
		t.Fatal(err)
	}

	// verify file is encrypted (not readable as json)
	raw, _ := os.ReadFile(path)
	if string(raw[:1]) == "{" {
		t.Fatal("snapshot should be encrypted, not plaintext json")
	}

	// restore
	s2 := New()
	if err := s2.LoadSnapshot(path, enc); err != nil {
		t.Fatal(err)
	}

	e, ok := s2.Get("secret")
	if !ok || string(e.Value) != "classified" {
		t.Fatal("snapshot restore failed")
	}
}
