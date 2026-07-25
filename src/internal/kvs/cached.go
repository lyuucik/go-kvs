package kvs

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

type CachedStore struct {
	store KeyValueStore
	rdb   *redis.Client
	ttl   time.Duration
}

func NewCachedStore(store KeyValueStore, redisURL string, ttl time.Duration) (*CachedStore, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}

	rdb := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		rdb.Close()
		return nil, err
	}

	return &CachedStore{store: store, rdb: rdb, ttl: ttl}, nil
}

func (c *CachedStore) Get(key string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	val, err := c.rdb.Get(ctx, key).Result()
	if err == nil {
		return val, nil
	}
	if err != redis.Nil {
		log.Printf("redis get error (cache miss fallback): %v", err)
	}

	v, err := c.store.Get(key)
	if err != nil {
		return "", err
	}

	setCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := c.rdb.Set(setCtx, key, v, c.ttl).Err(); err != nil {
		log.Printf("redis set error: %v", err)
	}

	return v, nil
}

func (c *CachedStore) Put(key, value string) error {
	if err := c.store.Put(key, value); err != nil {
		return err
	}
	c.invalidate(key)
	return nil
}

func (c *CachedStore) Delete(key string) error {
	if err := c.store.Delete(key); err != nil {
		return err
	}
	c.invalidate(key)
	return nil
}

func (c *CachedStore) Close() {
	c.rdb.Close()
}

func (c *CachedStore) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := c.rdb.Ping(ctx).Err(); err != nil {
		return c.store.Ping()
	}
	return nil
}

func (c *CachedStore) invalidate(key string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := c.rdb.Del(ctx, key).Err(); err != nil {
		log.Printf("redis del error: %v", err)
	}
}
