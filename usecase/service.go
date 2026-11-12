package usecase

import (
	"context"
	"math/rand/v2"

	"github.com/nemdull/sf-meter-api/domain"
)

// TechInput は、追加・更新のときに外から受け取る値。
type TechInput struct {
	Name        string
	Description string
	Score       int
}

// TechList は、一覧の 1 ページ分。
type TechList struct {
	Items []domain.Tech
	Total int // 絞り込み後の総件数
}

// Stats は、登録されている技術全体の集計。
type Stats struct {
	Count   int
	Average float64 // 実現度の平均。0 件のときは 0
	Min     int
	Max     int
	Levels  map[domain.Level]int // 段階ごとの件数。0 件の段階も入る
}

// TechService は技術情報に関するビジネスロジックを提供する
type TechService struct {
	repo TechRepository
	// pick は 0 以上 n 未満の数を返す。Random のテストで結果を固定できるように、差し替えられるようにしてある
	pick func(n int) int
}

// NewTechService はTechServiceのインスタンスを生成して返す
func NewTechService(repo TechRepository) *TechService {
	return &TechService{repo: repo, pick: rand.IntN}
}

// WithPicker は、Random が使う乱数を差し替えた TechService を返す（テスト用）
func (s *TechService) WithPicker(pick func(n int) int) *TechService {
	return &TechService{repo: s.repo, pick: pick}
}

// List は条件に合う技術情報を取得する
func (s *TechService) List(ctx context.Context, query TechQuery) (TechList, error) {
	items, total, err := s.repo.Find(ctx, query)
	if err != nil {
		return TechList{}, err
	}
	return TechList{Items: items, Total: total}, nil
}

// Get はIDに基づいて技術情報を取得する。無ければ domain.ErrNotFound
func (s *TechService) Get(ctx context.Context, id int) (domain.Tech, error) {
	return s.repo.FindByID(ctx, id)
}

// Random は登録されている技術から 1 件を選んで返す。1 件も無ければ domain.ErrNotFound
func (s *TechService) Random(ctx context.Context) (domain.Tech, error) {
	items, _, err := s.repo.Find(ctx, AllTechs())
	if err != nil {
		return domain.Tech{}, err
	}
	if len(items) == 0 {
		return domain.Tech{}, domain.ErrNotFound
	}
	return items[s.pick(len(items))], nil
}

// Stats は全体の件数・平均・段階ごとの内訳を返す
func (s *TechService) Stats(ctx context.Context) (Stats, error) {
	items, _, err := s.repo.Find(ctx, AllTechs())
	if err != nil {
		return Stats{}, err
	}
	stats := Stats{
		Count:  len(items),
		Levels: map[domain.Level]int{domain.LevelFiction: 0, domain.LevelEmerging: 0, domain.LevelReality: 0},
	}
	if len(items) == 0 {
		return stats, nil
	}
	sum := 0
	stats.Min, stats.Max = items[0].Score, items[0].Score
	for _, tech := range items {
		sum += tech.Score
		stats.Min = min(stats.Min, tech.Score)
		stats.Max = max(stats.Max, tech.Score)
		stats.Levels[tech.Level()]++
	}
	stats.Average = float64(sum) / float64(len(items))
	return stats, nil
}

// Create は入力を確かめてから、新しい技術情報を保存する
func (s *TechService) Create(ctx context.Context, input TechInput) (domain.Tech, error) {
	tech := domain.Tech{Name: input.Name, Description: input.Description, Score: input.Score}.Normalized()
	if err := tech.Validate(); err != nil {
		return domain.Tech{}, err
	}
	return s.repo.Create(ctx, tech)
}

// Update は入力を確かめてから、指定IDの技術情報を置き換える。無ければ domain.ErrNotFound
func (s *TechService) Update(ctx context.Context, id int, input TechInput) (domain.Tech, error) {
	tech := domain.Tech{ID: id, Name: input.Name, Description: input.Description, Score: input.Score}.Normalized()
	if err := tech.Validate(); err != nil {
		return domain.Tech{}, err
	}
	return s.repo.Update(ctx, tech)
}

// Delete は指定IDの技術情報を削除する。無ければ domain.ErrNotFound
func (s *TechService) Delete(ctx context.Context, id int) error {
	return s.repo.Delete(ctx, id)
}
