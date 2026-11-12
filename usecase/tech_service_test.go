package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nemdull/sf-meter-api/domain"
	"github.com/nemdull/sf-meter-api/usecase"
)

// ─── Mock ───────────────────────────────────────────────────────────────────

// mockTechRepo は、渡された値を覚えておき、決めた値を返すだけのリポジトリ。
// 絞り込みや並べ替えそのものは infrastructure/repotest で確かめるので、ここでは実装しない
type mockTechRepo struct {
	techs    []domain.Tech
	forceErr error

	gotQuery usecase.TechQuery
	saved    *domain.Tech
	deleted  int
}

func (m *mockTechRepo) Find(_ context.Context, query usecase.TechQuery) ([]domain.Tech, int, error) {
	m.gotQuery = query
	if m.forceErr != nil {
		return nil, 0, m.forceErr
	}
	return m.techs, len(m.techs), nil
}

func (m *mockTechRepo) FindByID(_ context.Context, id int) (domain.Tech, error) {
	if m.forceErr != nil {
		return domain.Tech{}, m.forceErr
	}
	for _, t := range m.techs {
		if t.ID == id {
			return t, nil
		}
	}
	return domain.Tech{}, domain.ErrNotFound
}

func (m *mockTechRepo) Create(_ context.Context, tech domain.Tech) (domain.Tech, error) {
	if m.forceErr != nil {
		return domain.Tech{}, m.forceErr
	}
	tech.ID = len(m.techs) + 1
	m.saved = &tech
	return tech, nil
}

func (m *mockTechRepo) Update(_ context.Context, tech domain.Tech) (domain.Tech, error) {
	if m.forceErr != nil {
		return domain.Tech{}, m.forceErr
	}
	m.saved = &tech
	return tech, nil
}

func (m *mockTechRepo) Delete(_ context.Context, id int) error {
	if m.forceErr != nil {
		return m.forceErr
	}
	m.deleted = id
	return nil
}

var ctx = context.Background()

var sample = []domain.Tech{
	{ID: 1, Name: "ドミネーター", Description: "支配射撃体系", Score: 35},
	{ID: 2, Name: "ニューラルリンク", Description: "脳直結インターフェース", Score: 82},
	{ID: 3, Name: "タイムマシン", Description: "時空間を操作する装置", Score: 1},
	{ID: 4, Name: "量子暗号通信", Description: "量子もつれを使った秘匿通信", Score: 88},
}

// ─── List ────────────────────────────────────────────────────────────────────

func TestTechService_List_PassesQueryAndReturnsTotal(t *testing.T) {
	repo := &mockTechRepo{techs: sample}
	svc := usecase.NewTechService(repo)

	query := usecase.TechQuery{Keyword: "通信", MinScore: 10, MaxScore: 90, Sort: usecase.SortByScoreDesc, Limit: 5, Offset: 2}
	list, err := svc.List(ctx, query)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.gotQuery != query {
		t.Errorf("条件がそのまま渡っていない: %+v", repo.gotQuery)
	}
	if len(list.Items) != len(sample) || list.Total != len(sample) {
		t.Errorf("expected %d items, got %d (total %d)", len(sample), len(list.Items), list.Total)
	}
}

func TestTechService_List_RepoError(t *testing.T) {
	svc := usecase.NewTechService(&mockTechRepo{forceErr: errors.New("db error")})

	if _, err := svc.List(ctx, usecase.AllTechs()); err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ─── Get ─────────────────────────────────────────────────────────────────────

func TestTechService_Get_Found(t *testing.T) {
	svc := usecase.NewTechService(&mockTechRepo{techs: sample})

	tech, err := svc.Get(ctx, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tech.Name != "ドミネーター" {
		t.Errorf("unexpected name: %s", tech.Name)
	}
}

func TestTechService_Get_NotFound(t *testing.T) {
	svc := usecase.NewTechService(&mockTechRepo{techs: sample})

	if _, err := svc.Get(ctx, 999); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ─── Random ──────────────────────────────────────────────────────────────────

func TestTechService_Random_UsesPicker(t *testing.T) {
	var asked int
	svc := usecase.NewTechService(&mockTechRepo{techs: sample}).WithPicker(func(n int) int {
		asked = n
		return 2
	})

	tech, err := svc.Random(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if asked != len(sample) {
		t.Errorf("件数と同じ範囲で選ぶはずが: %d", asked)
	}
	if tech.Name != "タイムマシン" {
		t.Errorf("3 件目が選ばれるはずが: %s", tech.Name)
	}
}

func TestTechService_Random_Empty(t *testing.T) {
	svc := usecase.NewTechService(&mockTechRepo{})

	if _, err := svc.Random(ctx); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("1 件も無いときは ErrNotFound になるはずが: %v", err)
	}
}

// ─── Stats ───────────────────────────────────────────────────────────────────

func TestTechService_Stats(t *testing.T) {
	svc := usecase.NewTechService(&mockTechRepo{techs: sample})

	stats, err := svc.Stats(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.Count != 4 || stats.Min != 1 || stats.Max != 88 {
		t.Errorf("unexpected stats: %+v", stats)
	}
	if stats.Average != 51.5 { // (35 + 82 + 1 + 88) / 4
		t.Errorf("average = %v、期待は 51.5", stats.Average)
	}
	want := map[domain.Level]int{domain.LevelFiction: 1, domain.LevelEmerging: 1, domain.LevelReality: 2}
	for level, count := range want {
		if stats.Levels[level] != count {
			t.Errorf("%s = %d、期待は %d", level, stats.Levels[level], count)
		}
	}
}

func TestTechService_Stats_Empty(t *testing.T) {
	svc := usecase.NewTechService(&mockTechRepo{})

	stats, err := svc.Stats(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.Count != 0 || stats.Average != 0 {
		t.Errorf("0 件のときは 0 になるはずが: %+v", stats)
	}
	if len(stats.Levels) != 3 {
		t.Errorf("0 件でも 3 つの段階がそろっているはずが: %v", stats.Levels)
	}
}

// ─── Create / Update / Delete ────────────────────────────────────────────────

func TestTechService_Create_TrimsAndSaves(t *testing.T) {
	repo := &mockTechRepo{techs: sample}
	svc := usecase.NewTechService(repo)

	tech, err := svc.Create(ctx, usecase.TechInput{Name: "  軌道エレベーター ", Description: " 宇宙まで届く昇降機\n", Score: 20})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tech.ID != 5 || tech.Name != "軌道エレベーター" || tech.Description != "宇宙まで届く昇降機" {
		t.Errorf("unexpected tech: %+v", tech)
	}
	if repo.saved == nil || *repo.saved != tech {
		t.Errorf("保存した値が違う: %+v", repo.saved)
	}
}

func TestTechService_Create_InvalidIsNotSaved(t *testing.T) {
	tests := []struct {
		name  string
		input usecase.TechInput
		field string
	}{
		{"名前が空白だけ", usecase.TechInput{Name: "   ", Score: 10}, "name"},
		{"実現度がはみ出す", usecase.TechInput{Name: "x", Score: 101}, "score"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockTechRepo{}
			_, err := usecase.NewTechService(repo).Create(ctx, tc.input)

			var invalid *domain.ValidationError
			if !errors.As(err, &invalid) || invalid.Field != tc.field {
				t.Fatalf("%s の ValidationError になるはずが: %v", tc.field, err)
			}
			if repo.saved != nil {
				t.Error("確かめに落ちた値は、保存しないはず")
			}
		})
	}
}

func TestTechService_Update_KeepsID(t *testing.T) {
	repo := &mockTechRepo{techs: sample}
	svc := usecase.NewTechService(repo)

	tech, err := svc.Update(ctx, 2, usecase.TechInput{Name: "ニューラルリンク改", Score: 90})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tech.ID != 2 || tech.Score != 90 {
		t.Errorf("unexpected tech: %+v", tech)
	}
}

func TestTechService_Update_InvalidIsNotSaved(t *testing.T) {
	repo := &mockTechRepo{techs: sample}

	if _, err := usecase.NewTechService(repo).Update(ctx, 2, usecase.TechInput{Name: "", Score: 10}); err == nil {
		t.Fatal("expected error, got nil")
	}
	if repo.saved != nil {
		t.Error("確かめに落ちた値は、保存しないはず")
	}
}

func TestTechService_Delete(t *testing.T) {
	repo := &mockTechRepo{techs: sample}

	if err := usecase.NewTechService(repo).Delete(ctx, 3); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.deleted != 3 {
		t.Errorf("deleted = %d、期待は 3", repo.deleted)
	}
}
