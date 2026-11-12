package usecase

import (
	"context"

	"github.com/nemdull/sf-meter-api/domain"
)

// SortKey は一覧の並び順。
type SortKey string

const (
	SortByID        SortKey = "id"     // 登録した順
	SortByScoreAsc  SortKey = "score"  // 実現度の低い順
	SortByScoreDesc SortKey = "-score" // 実現度の高い順
	SortByName      SortKey = "name"   // 名前順（文字コード順）
)

// ValidSortKey は、受け付ける並び順かどうかを返す。
func ValidSortKey(key SortKey) bool {
	switch key {
	case SortByID, SortByScoreAsc, SortByScoreDesc, SortByName:
		return true
	}
	return false
}

// TechQuery は、一覧を取るときの絞り込みと並べ替えの条件。
// ゼロ値のままだと何も返らない（MaxScore が 0 のため）ので、AllTechs から作るか Normalize を通す。
type TechQuery struct {
	Keyword  string  // 名前か説明に含まれる文字。大文字・小文字は区別しない。空なら絞り込まない
	MinScore int     // 実現度の下限（含む）
	MaxScore int     // 実現度の上限（含む）
	Sort     SortKey // 並び順
	Limit    int     // 返す件数。0 なら全件
	Offset   int     // 先頭から飛ばす件数
}

// AllTechs は、絞り込みなし・登録順・全件の条件を返す。
func AllTechs() TechQuery {
	return TechQuery{MinScore: domain.MinScore, MaxScore: domain.MaxScore, Sort: SortByID}
}

// TechRepository は、技術情報の保存先を抽象化する。
// 実装は infrastructure 層にあり、どの実装も infrastructure/repotest の同じテストを通す。
//
// 見つからないときは、FindByID / Update / Delete が domain.ErrNotFound を返す。
type TechRepository interface {
	// Find は条件に合う技術と、絞り込み後の総件数（Limit / Offset を当てる前の数）を返す。
	Find(ctx context.Context, query TechQuery) (items []domain.Tech, total int, err error)
	FindByID(ctx context.Context, id int) (domain.Tech, error)
	// Create は新しい ID を割り当てて保存し、保存した値を返す。
	Create(ctx context.Context, tech domain.Tech) (domain.Tech, error)
	// Update は tech.ID の技術を置き換え、保存した値を返す。
	Update(ctx context.Context, tech domain.Tech) (domain.Tech, error)
	Delete(ctx context.Context, id int) error
}
