package crud

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"cnbattle.com/ai/pkg/cache"
)

// mockCache 用于测试的内存缓存实现。
type mockCache struct {
	mu    sync.RWMutex
	store map[string]string
}

func newMockCache() *mockCache {
	return &mockCache{store: make(map[string]string)}
}

func (m *mockCache) Set(key, value string, exp time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.store[key] = value
	return nil
}

func (m *mockCache) Get(key string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.store[key]
	if !ok {
		return "", errors.New("not found")
	}
	return v, nil
}

func (m *mockCache) Del(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.store, key)
	return nil
}

func (m *mockCache) GetOrLoad(ctx context.Context, key string, exp time.Duration, loader cache.Loader) (string, error) {
	m.mu.RLock()
	v, ok := m.store[key]
	m.mu.RUnlock()
	if ok {
		return v, nil
	}

	v, err := loader(ctx)
	if err != nil {
		return "", err
	}

	m.mu.Lock()
	m.store[key] = v
	m.mu.Unlock()
	return v, nil
}

func (m *mockCache) DeleteEmpty(key string) error {
	return m.Del(key)
}

func (m *mockCache) DelMulti(keys ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, key := range keys {
		delete(m.store, key)
	}
	return nil
}

// mockModel 测试用 model。
type mockModel struct {
	id   int64
	name string
}

func (mockModel) TableName() string         { return "mocks" }
func (m mockModel) CachePrefix() string     { return "test" }
func (m mockModel) PrimaryKeyColumn() string { return "id" }
func (m mockModel) PrimaryKey() any         { return m.id }

func TestCachedModel_PrimaryKey(t *testing.T) {
	m := NewCachedModel[int64](42)
	if m.PrimaryKey() != int64(42) {
		t.Errorf("PrimaryKey() = %v, want 42", m.PrimaryKey())
	}
}

func TestCachedModel_PrimaryKeyColumn(t *testing.T) {
	m := NewCachedModel[int64](1)
	if m.PrimaryKeyColumn() != "id" {
		t.Errorf("PrimaryKeyColumn() = %q, want %q", m.PrimaryKeyColumn(), "id")
	}
}

func TestCachedModel_CachePrefix(t *testing.T) {
	m := NewCachedModel[int64](1)
	if m.CachePrefix() != CachePrefix {
		t.Errorf("CachePrefix() = %q, want %q", m.CachePrefix(), CachePrefix)
	}
}
