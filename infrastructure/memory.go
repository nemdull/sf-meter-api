package infrastructure

import (
	"sync"

	"github.com/nemdull/sf-meter-api/domain"
)

// MemoryTechRepo is an in-memory implementation of domain.TechRepository.
// Useful for local development and integration tests without a real DB.
type MemoryTechRepo struct {
	mu    sync.RWMutex
	store map[int]domain.Tech
	next  int
}

// NewMemoryTechRepo seeds the repo with default SF technologies.
func NewMemoryTechRepo() *MemoryTechRepo {
	r := &MemoryTechRepo{store: make(map[int]domain.Tech), next: 1}
	seeds := []domain.Tech{
		{Name: "ドミネーター", Description: "支配射撃体系", Score: 35},
		{Name: "シビラシステム", Description: "脳スキャンで犯罪係数を算出する予防監視システム", Score: 91},
		{Name: "ニューラルリンク", Description: "脳とコンピュータを直接接続するBCIデバイス", Score: 82},
	}
	for _, t := range seeds {
		_ = r.create(t)
	}
	return r
}

func (r *MemoryTechRepo) create(t domain.Tech) domain.Tech {
	t.ID = r.next
	r.store[r.next] = t
	r.next++
	return t
}

// FindAll returns all stored techs.
func (r *MemoryTechRepo) FindAll() ([]domain.Tech, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	techs := make([]domain.Tech, 0, len(r.store))
	for _, t := range r.store {
		techs = append(techs, t)
	}
	return techs, nil
}

// FindByID returns the tech with the given ID, or nil if not found.
func (r *MemoryTechRepo) FindByID(id int) (*domain.Tech, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.store[id]
	if !ok {
		return nil, nil
	}
	return &t, nil
}
