package wal

import (
	"os"
	"testing"
)

func TestAppendAndReplay(t *testing.T) {
	path := "/tmp/test_wal.log"
	defer os.Remove(path)

	w, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}

	entries := []Entry{
		{Op: OpPut, Key: "name", Value: []byte("alice")},
		{Op: OpPut, Key: "age", Value: []byte("30")},
		{Op: OpDelete, Key: "name", Value: nil},
	}

	for _, e := range entries {
		if err := w.Append(e); err != nil {
			t.Fatal(err)
		}
	}
	w.Close()

	w2, _ := Open(path)
	defer w2.Close()

	replayed, err := w2.Replay()
	if err != nil {
		t.Fatal(err)
	}
	if len(replayed) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(replayed))
	}
	if replayed[0].Key != "name" || string(replayed[0].Value) != "alice" {
		t.Fatal("first entry mismatch")
	}
	if replayed[2].Op != OpDelete {
		t.Fatal("third entry should be delete")
	}
}
