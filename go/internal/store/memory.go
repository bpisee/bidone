package store

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/lee/BidOne/pkg/model"
)

var (
	errNotFound = errors.New("product not found")
)

type MemoryStore struct {
	mu    sync.RWMutex
	items map[string]model.Product
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{items: make(map[string]model.Product)}
}

func (s *MemoryStore) List(ctx context.Context) ([]model.Product, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]model.Product, 0, len(s.items))
	for _, p := range s.items {
		result = append(result, p)
	}
	return result, nil
}

func (s *MemoryStore) Get(ctx context.Context, id string) (model.Product, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.items[id]
	if !ok {
		return model.Product{}, errNotFound
	}
	return p, nil
}

func (s *MemoryStore) Create(ctx context.Context, p model.Product) (model.Product, error) {
	if err := p.ValidateForCreate(); err != nil {
		return model.Product{}, err
	}
	now := time.Now().UTC()
	p.ID = uuid.NewString()
	p.CreatedAt = now
	p.UpdatedAt = now

	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[p.ID] = p
	return p, nil
}

func (s *MemoryStore) Update(ctx context.Context, id string, p model.Product) (model.Product, error) {
	if err := p.ValidateForUpdate(); err != nil {
		return model.Product{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	old, ok := s.items[id]
	if !ok {
		return model.Product{}, errNotFound
	}
	p.ID = id
	p.CreatedAt = old.CreatedAt
	p.UpdatedAt = time.Now().UTC()
	s.items[id] = p
	return p, nil
}

func (s *MemoryStore) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[id]; !ok {
		return errNotFound
	}
	delete(s.items, id)
	return nil
}
