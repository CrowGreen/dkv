package store

import (
	"fmt"
	"testing"
)

func BenchmarkPut(b *testing.B) {
	s := New()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Put(fmt.Sprintf("key-%d", i), []byte("value"))
	}
}

func BenchmarkGet(b *testing.B) {
	s := New()
	for i := 0; i < 10000; i++ {
		s.Put(fmt.Sprintf("key-%d", i), []byte("value"))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Get(fmt.Sprintf("key-%d", i%10000))
	}
}

func BenchmarkConcurrentPut(b *testing.B) {
	s := New()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			s.Put(fmt.Sprintf("key-%d", i), []byte("value"))
			i++
		}
	})
}
