package kvs

import "sync"

type KeyValueStore interface {
	Get(key string) (string, error)
	Put(key, value string) error
	Delete(key string) error
	Ping() error
}

type Store struct {
	m map[string]string
	sync.RWMutex
}

func NewKeyValueStore() KeyValueStore {
	return &Store{m: make(map[string]string)}
}

func (s *Store) Get(key string) (string, error) {
	s.RLock()
	defer s.RUnlock()
	v, ok := s.m[key]
	if !ok {
		return "", ErrorNoSuchKey
	}
	return v, nil
}

func (s *Store) Put(key string, value string) error {
	s.Lock()
	defer s.Unlock()
	s.m[key] = value
	return nil
}

func (s *Store) Delete(key string) error {
	s.Lock()
	defer s.Unlock()
	delete(s.m, key)
	return nil
}

func (s *Store) Ping() error {
	return nil
}
