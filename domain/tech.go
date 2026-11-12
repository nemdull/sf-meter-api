// Package domain は、SF メーターの中心になる型と、その不変条件を持つ。
// ほかの層（usecase / infrastructure / presentation）のことは何も知らない。
package domain

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

// 値の上限・下限。db/init.sql の制約（VARCHAR(100)、score の CHECK）と合わせてある。
const (
	MaxNameLength        = 100
	MaxDescriptionLength = 500
	MinScore             = 0
	MaxScore             = 100
)

// ErrNotFound は、指定した技術が存在しないことを表す。
// リポジトリはこのエラーを返し、ハンドラーが 404 に変換する。
var ErrNotFound = errors.New("tech: not found")

// ValidationError は、不変条件に違反した項目と理由を持つ。
// ハンドラーは errors.As で取り出して、400 の応答に変換する。
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("tech: %s %s", e.Field, e.Message)
}

// Level は、実現度（Score）を 3 段階に丸めたもの。
type Level string

const (
	LevelFiction  Level = "fiction"  // 0〜29: まだ SF
	LevelEmerging Level = "emerging" // 30〜69: 研究が進んでいる
	LevelReality  Level = "reality"  // 70〜100: ほぼ現実
)

// Tech は、SF 作品に登場するテクノロジーと、その実現度を表す。
type Tech struct {
	ID          int
	Name        string
	Description string
	Score       int // 0〜100。現実にどれくらい近いか
}

// Normalized は、前後の空白を取り除いた値を返す。保存する前に通す。
func (t Tech) Normalized() Tech {
	t.Name = strings.TrimSpace(t.Name)
	t.Description = strings.TrimSpace(t.Description)
	return t
}

// Validate は不変条件を確かめる。
//   - 名前は空でなく、MaxNameLength 文字以内
//   - 説明は MaxDescriptionLength 文字以内
//   - 実現度は MinScore 以上 MaxScore 以下
func (t Tech) Validate() error {
	if strings.TrimSpace(t.Name) == "" {
		return &ValidationError{Field: "name", Message: "must not be empty"}
	}
	if utf8.RuneCountInString(t.Name) > MaxNameLength {
		return &ValidationError{Field: "name", Message: fmt.Sprintf("must be at most %d characters", MaxNameLength)}
	}
	if utf8.RuneCountInString(t.Description) > MaxDescriptionLength {
		return &ValidationError{Field: "description", Message: fmt.Sprintf("must be at most %d characters", MaxDescriptionLength)}
	}
	if t.Score < MinScore || t.Score > MaxScore {
		return &ValidationError{Field: "score", Message: fmt.Sprintf("%d is out of range [%d, %d]", t.Score, MinScore, MaxScore)}
	}
	return nil
}

// Level は、実現度がどの段階にあるかを返す。
func (t Tech) Level() Level {
	switch {
	case t.Score >= 70:
		return LevelReality
	case t.Score >= 30:
		return LevelEmerging
	default:
		return LevelFiction
	}
}
