package infrastructure

import (
	"cmp"
	"context"
	"slices"
	"strings"
	"sync"

	"github.com/nemdull/sf-meter-api/domain"
	"github.com/nemdull/sf-meter-api/usecase"
)

// MemoryTechRepo は、メモリ上に持つ usecase.TechRepository の実装。
// DB を用意せずに動かしたいときと、テストで使う。プロセスを止めると中身は消える。
type MemoryTechRepo struct {
	mu    sync.RWMutex
	store map[int]domain.Tech
	next  int
}

// NewMemoryTechRepo は、渡した技術を ID 1 から順に入れた状態で作る。
// 何も渡さなければ空になる。初期データ入りで作るなら NewMemoryTechRepo(SeedTechs()...)。
func NewMemoryTechRepo(seeds ...domain.Tech) *MemoryTechRepo {
	r := &MemoryTechRepo{store: make(map[int]domain.Tech), next: 1}
	for _, tech := range seeds {
		tech.ID = r.next
		r.store[tech.ID] = tech
		r.next++
	}
	return r
}

// Find は条件に合う技術と、絞り込み後の総件数を返す。
func (r *MemoryTechRepo) Find(_ context.Context, query usecase.TechQuery) ([]domain.Tech, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	keyword := strings.ToLower(query.Keyword)
	matched := make([]domain.Tech, 0, len(r.store))
	for _, tech := range r.store {
		if tech.Score < query.MinScore || tech.Score > query.MaxScore {
			continue
		}
		if keyword != "" &&
			!strings.Contains(strings.ToLower(tech.Name), keyword) &&
			!strings.Contains(strings.ToLower(tech.Description), keyword) {
			continue
		}
		matched = append(matched, tech)
	}

	// map の取り出し順は毎回変わるので、必ず並べ替える。同じ値どうしは ID 順にして、結果がぶれないようにする
	slices.SortFunc(matched, func(a, b domain.Tech) int {
		var order int
		switch query.Sort {
		case usecase.SortByScoreAsc:
			order = cmp.Compare(a.Score, b.Score)
		case usecase.SortByScoreDesc:
			order = cmp.Compare(b.Score, a.Score)
		case usecase.SortByName:
			order = strings.Compare(a.Name, b.Name)
		}
		return cmp.Or(order, cmp.Compare(a.ID, b.ID))
	})

	total := len(matched)
	start := min(max(query.Offset, 0), total)
	end := total
	if query.Limit > 0 {
		end = min(start+query.Limit, total)
	}
	return matched[start:end], total, nil
}

// FindByID は指定 ID の技術を返す。無ければ domain.ErrNotFound。
func (r *MemoryTechRepo) FindByID(_ context.Context, id int) (domain.Tech, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tech, ok := r.store[id]
	if !ok {
		return domain.Tech{}, domain.ErrNotFound
	}
	return tech, nil
}

// Create は新しい ID を割り当てて保存する。
func (r *MemoryTechRepo) Create(_ context.Context, tech domain.Tech) (domain.Tech, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	tech.ID = r.next
	r.store[tech.ID] = tech
	r.next++
	return tech, nil
}

// Update は tech.ID の技術を置き換える。無ければ domain.ErrNotFound。
func (r *MemoryTechRepo) Update(_ context.Context, tech domain.Tech) (domain.Tech, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.store[tech.ID]; !ok {
		return domain.Tech{}, domain.ErrNotFound
	}
	r.store[tech.ID] = tech
	return tech, nil
}

// Delete は指定 ID の技術を消す。無ければ domain.ErrNotFound。
func (r *MemoryTechRepo) Delete(_ context.Context, id int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.store[id]; !ok {
		return domain.ErrNotFound
	}
	delete(r.store, id)
	return nil
}
