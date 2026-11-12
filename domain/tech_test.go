package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/nemdull/sf-meter-api/domain"
)

func TestTech_Validate(t *testing.T) {
	tests := []struct {
		name      string
		tech      domain.Tech
		wantField string // 空なら、エラーにならないこと
	}{
		{"下限の実現度", domain.Tech{Name: "タイムマシン", Score: 0}, ""},
		{"上限の実現度", domain.Tech{Name: "量子暗号通信", Score: 100}, ""},
		{"名前がちょうど上限", domain.Tech{Name: strings.Repeat("あ", domain.MaxNameLength), Score: 50}, ""},
		{"名前が空", domain.Tech{Name: "", Score: 50}, "name"},
		{"名前が空白だけ", domain.Tech{Name: " 　\t", Score: 50}, "name"},
		{"名前が長すぎる", domain.Tech{Name: strings.Repeat("あ", domain.MaxNameLength+1), Score: 50}, "name"},
		{"説明が長すぎる", domain.Tech{Name: "x", Description: strings.Repeat("あ", domain.MaxDescriptionLength+1), Score: 50}, "description"},
		{"実現度が下にはみ出す", domain.Tech{Name: "x", Score: -1}, "score"},
		{"実現度が上にはみ出す", domain.Tech{Name: "x", Score: 101}, "score"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.tech.Validate()
			if tc.wantField == "" {
				if err != nil {
					t.Fatalf("エラーにならないはずが: %v", err)
				}
				return
			}
			var invalid *domain.ValidationError
			if !errors.As(err, &invalid) {
				t.Fatalf("ValidationError になるはずが: %v", err)
			}
			if invalid.Field != tc.wantField {
				t.Errorf("項目 = %q、期待は %q", invalid.Field, tc.wantField)
			}
		})
	}
}

func TestTech_Level(t *testing.T) {
	tests := []struct {
		score int
		want  domain.Level
	}{
		{0, domain.LevelFiction},
		{29, domain.LevelFiction},
		{30, domain.LevelEmerging},
		{69, domain.LevelEmerging},
		{70, domain.LevelReality},
		{100, domain.LevelReality},
	}
	for _, tc := range tests {
		if got := (domain.Tech{Score: tc.score}).Level(); got != tc.want {
			t.Errorf("Score %d → %q、期待は %q", tc.score, got, tc.want)
		}
	}
}

func TestTech_Normalized(t *testing.T) {
	got := domain.Tech{Name: "  ナノマシン　", Description: "\t体内を修復する\n"}.Normalized()
	if got.Name != "ナノマシン" || got.Description != "体内を修復する" {
		t.Errorf("前後の空白が残っている: %+v", got)
	}
}
