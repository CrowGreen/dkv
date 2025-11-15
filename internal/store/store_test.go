package store

import (
	"testing"
	"sync"
)

func TestPutGet(t *testing.T) {
	s := New()
	v := s.Put("foo", []byte("bar"))
	if v != 1 {
		t.Fatalf("expected version 1, got %d", v)
	}
	e, ok := s.Get("foo")
	if !ok {
		t.Fatal("key not found")
	}
	if string(e.Value) != "bar" {
		t.Fatalf("expected bar, got %s", e.Value)
	}
}

func TestDelete(t *testing.T) {
	s := New()
	s.Put("a", []byte("1"))
	ok := s.Delete("a")
	if !ok {
		t.Fatal("delete returned false")
	}
	_, found := s.Get("a")
	if found {
		t.Fatal("key should be deleted")
	}
}

func TestConcurrentAccess(t *testing.T) {
	s := New()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := "key"
			s.Put(key, []byte("val"))
			s.Get(key)
		}(i)
	}
	wg.Wait()
}

func TestWatch(t *testing.T) {
	s := New()
	ch := s.Subscribe("user:")

	go func() {
		s.Put("user:1", []byte("alice"))
	}()

	event := <-ch
	if event.Key != "user:1" {
		t.Fatalf("expected user:1, got %s", event.Key)
	}
}
