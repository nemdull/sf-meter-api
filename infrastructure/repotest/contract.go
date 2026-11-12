// Package repotest は、usecase.TechRepository のどの実装にも同じように当てるテストを持つ。
//
// 保存先が違っても、外から見た振る舞い（絞り込み・並び順・件数・見つからないときのエラー）は同じでなければならない。
// メモリ実装と PostgreSQL 実装の両方がこのテストを通ることで、registry で実装を差し替えても結果が変わらないことを保証する。
package repotest

import (
	"context"
	"errors"
	"testing"

	"github.com/nemdull/sf-meter-api/domain"
	"github.com/nemdull/sf-meter-api/usecase"
)

// Fixture は、どのテストも同じ状態から始めるための初期データ。ID は 1 から順に振られる。
var Fixture = []domain.Tech{
	{Name: "タイムマシン", Description: "時空間を操作する装置", Score: 1},
	{Name: "ナノマシン", Description: "体内を修復する医療ロボット群", Score: 55},
	{Name: "量子暗号通信", Description: "量子もつれを使った秘匿通信", Score: 88},
	{Name: "Warp Drive", Description: "100%_faster than light", Score: 15},
	{Name: "ホログラム通信", Description: "立体映像による遠距離の通信", Score: 55},
}

// Factory は、Fixture だけが入った新しいリポジトリを返す。テストごとに呼ばれる。
type Factory func(t *testing.T) usecase.TechRepository

func names(techs []domain.Tech) []string {
	out := make([]string, len(techs))
	for i, tech := range techs {
		out[i] = tech.Name
	}
	return out
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Run は、リポジトリの実装が満たすべき振る舞いをまとめて確かめる。
func Run(t *testing.T, newRepo Factory) {
	t.Helper()
	ctx := context.Background()

	t.Run("Find: 並び順", func(t *testing.T) {
		tests := []struct {
			sort usecase.SortKey
			want []string
		}{
			{usecase.SortByID, []string{"タイムマシン", "ナノマシン", "量子暗号通信", "Warp Drive", "ホログラム通信"}},
			// 同じ実現度（55）どうしは ID 順になる
			{usecase.SortByScoreAsc, []string{"タイムマシン", "Warp Drive", "ナノマシン", "ホログラム通信", "量子暗号通信"}},
			{usecase.SortByScoreDesc, []string{"量子暗号通信", "ナノマシン", "ホログラム通信", "Warp Drive", "タイムマシン"}},
			// 名前順は文字コード順（英字 → カタカナ → 漢字）
			{usecase.SortByName, []string{"Warp Drive", "タイムマシン", "ナノマシン", "ホログラム通信", "量子暗号通信"}},
		}
		for _, tc := range tests {
			query := usecase.AllTechs()
			query.Sort = tc.sort
			items, total, err := newRepo(t).Find(ctx, query)
			if err != nil {
				t.Fatalf("sort=%q: %v", tc.sort, err)
			}
			if total != len(Fixture) {
				t.Errorf("sort=%q: total = %d、期待は %d", tc.sort, total, len(Fixture))
			}
			if got := names(items); !equal(got, tc.want) {
				t.Errorf("sort=%q:\n got  %v\n want %v", tc.sort, got, tc.want)
			}
		}
	})

	t.Run("Find: 絞り込み", func(t *testing.T) {
		tests := []struct {
			name   string
			change func(q *usecase.TechQuery)
			want   []string
		}{
			{"名前に含まれる", func(q *usecase.TechQuery) { q.Keyword = "マシン" }, []string{"タイムマシン", "ナノマシン"}},
			{"説明に含まれる", func(q *usecase.TechQuery) { q.Keyword = "量子もつれ" }, []string{"量子暗号通信"}},
			{"大文字と小文字を区別しない", func(q *usecase.TechQuery) { q.Keyword = "warp" }, []string{"Warp Drive"}},
			{"% は文字として探す", func(q *usecase.TechQuery) { q.Keyword = "100%" }, []string{"Warp Drive"}},
			{"_ は文字として探す", func(q *usecase.TechQuery) { q.Keyword = "%_f" }, []string{"Warp Drive"}},
			{"どれにも当たらない", func(q *usecase.TechQuery) { q.Keyword = "ワープ" }, []string{}},
			{"実現度の下限（含む）", func(q *usecase.TechQuery) { q.MinScore = 55 }, []string{"ナノマシン", "量子暗号通信", "ホログラム通信"}},
			{"実現度の上限（含む）", func(q *usecase.TechQuery) { q.MaxScore = 15 }, []string{"タイムマシン", "Warp Drive"}},
			{"実現度の範囲と文字の両方", func(q *usecase.TechQuery) { q.MinScore, q.MaxScore, q.Keyword = 50, 60, "通信" }, []string{"ホログラム通信"}},
		}
		for _, tc := range tests {
			query := usecase.AllTechs()
			tc.change(&query)
			items, total, err := newRepo(t).Find(ctx, query)
			if err != nil {
				t.Fatalf("%s: %v", tc.name, err)
			}
			if got := names(items); !equal(got, tc.want) {
				t.Errorf("%s:\n got  %v\n want %v", tc.name, got, tc.want)
			}
			if total != len(tc.want) {
				t.Errorf("%s: total = %d、期待は %d", tc.name, total, len(tc.want))
			}
			if items == nil {
				t.Errorf("%s: 0 件のときも nil ではなく空のスライスを返す", tc.name)
			}
		}
	})

	t.Run("Find: ページ分けしても total は絞り込み後の全件数", func(t *testing.T) {
		tests := []struct {
			limit, offset int
			want          []string
		}{
			{2, 0, []string{"タイムマシン", "ナノマシン"}},
			{2, 2, []string{"量子暗号通信", "Warp Drive"}},
			{2, 4, []string{"ホログラム通信"}},
			{2, 9, []string{}},
			{0, 3, []string{"Warp Drive", "ホログラム通信"}},
		}
		for _, tc := range tests {
			query := usecase.AllTechs()
			query.Limit, query.Offset = tc.limit, tc.offset
			items, total, err := newRepo(t).Find(ctx, query)
			if err != nil {
				t.Fatalf("limit=%d offset=%d: %v", tc.limit, tc.offset, err)
			}
			if got := names(items); !equal(got, tc.want) {
				t.Errorf("limit=%d offset=%d:\n got  %v\n want %v", tc.limit, tc.offset, got, tc.want)
			}
			if total != len(Fixture) {
				t.Errorf("limit=%d offset=%d: total = %d、期待は %d", tc.limit, tc.offset, total, len(Fixture))
			}
		}
	})

	t.Run("FindByID", func(t *testing.T) {
		repo := newRepo(t)
		tech, err := repo.FindByID(ctx, 3)
		if err != nil {
			t.Fatal(err)
		}
		want := domain.Tech{ID: 3, Name: "量子暗号通信", Description: "量子もつれを使った秘匿通信", Score: 88}
		if tech != want {
			t.Errorf("got %+v、want %+v", tech, want)
		}
		if _, err := repo.FindByID(ctx, 999); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("無い ID は ErrNotFound になるはずが: %v", err)
		}
	})

	t.Run("Create: 続きの ID が振られ、あとから取り出せる", func(t *testing.T) {
		repo := newRepo(t)
		created, err := repo.Create(ctx, domain.Tech{Name: "軌道エレベーター", Description: "宇宙まで届く昇降機", Score: 20})
		if err != nil {
			t.Fatal(err)
		}
		if created.ID != len(Fixture)+1 {
			t.Errorf("ID = %d、期待は %d", created.ID, len(Fixture)+1)
		}
		found, err := repo.FindByID(ctx, created.ID)
		if err != nil {
			t.Fatal(err)
		}
		if found != created {
			t.Errorf("保存した値と取り出した値が違う: %+v / %+v", created, found)
		}
	})

	t.Run("Update: 置き換わる。無い ID は ErrNotFound", func(t *testing.T) {
		repo := newRepo(t)
		changed := domain.Tech{ID: 2, Name: "ナノマシン改", Description: "書き換えた説明", Score: 60}
		if _, err := repo.Update(ctx, changed); err != nil {
			t.Fatal(err)
		}
		found, err := repo.FindByID(ctx, 2)
		if err != nil {
			t.Fatal(err)
		}
		if found != changed {
			t.Errorf("got %+v、want %+v", found, changed)
		}
		if _, err := repo.Update(ctx, domain.Tech{ID: 999, Name: "x"}); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("無い ID は ErrNotFound になるはずが: %v", err)
		}
	})

	t.Run("Delete: 消える。2 回目と無い ID は ErrNotFound", func(t *testing.T) {
		repo := newRepo(t)
		if err := repo.Delete(ctx, 1); err != nil {
			t.Fatal(err)
		}
		if _, err := repo.FindByID(ctx, 1); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("消したあとは ErrNotFound になるはずが: %v", err)
		}
		if err := repo.Delete(ctx, 1); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("2 回目の削除は ErrNotFound になるはずが: %v", err)
		}
		_, total, err := repo.Find(ctx, usecase.AllTechs())
		if err != nil {
			t.Fatal(err)
		}
		if total != len(Fixture)-1 {
			t.Errorf("total = %d、期待は %d", total, len(Fixture)-1)
		}
	})

	t.Run("消した ID は使い回さない", func(t *testing.T) {
		repo := newRepo(t)
		if err := repo.Delete(ctx, len(Fixture)); err != nil {
			t.Fatal(err)
		}
		created, err := repo.Create(ctx, domain.Tech{Name: "新入り", Score: 10})
		if err != nil {
			t.Fatal(err)
		}
		if created.ID != len(Fixture)+1 {
			t.Errorf("ID = %d、期待は %d", created.ID, len(Fixture)+1)
		}
	})
}
