package kvs

import "sync"

type KeyValueStore interface {
	Get(key string) (string, error)
	Put(key, value string) error
	Delete(key string) error
}

var _ KeyValueStore = &store{}

type store struct {
	m map[string]string
	sync.RWMutex
}

func NewKeyValueStore() KeyValueStore {
	return &store{m: make(map[string]string)}
}

func (s *store) Get(key string) (string, error) {
	v, ok := s.m[key]
	if !ok {
		return "", ErrorNoSuchKey
	}
	return v, nil
}

func (s *store) Put(key string, value string) error {
	s.m[key] = value
	return nil
}

func (s *store) Delete(key string) error {
	delete(s.m, key)
	return nil
}
