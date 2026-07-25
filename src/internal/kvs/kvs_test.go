package kvs_test

import (
	"kvs/src/internal/kvs"
	"sync"
	"testing"
)

func TestPutAndGet(t *testing.T) {
	s := kvs.NewKeyValueStore()

	if err := s.Put("foo", "bar"); err != nil {
		t.Fatal(err)
	}

	v, err := s.Get("foo")
	if err != nil {
		t.Fatal(err)
	}
	if v != "bar" {
		t.Errorf("expected 'bar', got %q", v)
	}
}

func TestGetMissingKey(t *testing.T) {
	s := kvs.NewKeyValueStore()

	_, err := s.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPutOverwrites(t *testing.T) {
	s := kvs.NewKeyValueStore()

	s.Put("k", "v1")
	s.Put("k", "v2")

	v, _ := s.Get("k")
	if v != "v2" {
		t.Errorf("expected 'v2', got %q", v)
	}
}

func TestDelete(t *testing.T) {
	s := kvs.NewKeyValueStore()

	s.Put("k", "v")
	s.Delete("k")

	_, err := s.Get("k")
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestDeleteNonExistent(t *testing.T) {
	s := kvs.NewKeyValueStore()

	if err := s.Delete("ghost"); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestConcurrentAccess(t *testing.T) {
	s := kvs.NewKeyValueStore()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			s.Put("key", "val")
			s.Get("key")
			s.Delete("key")
		}(i)
	}
	wg.Wait()
}
