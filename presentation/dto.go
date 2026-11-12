package presentation

import (
	"math"

	"github.com/nemdull/sf-meter-api/domain"
	"github.com/nemdull/sf-meter-api/usecase"
)

// TechDTO は技術情報のDTO（Data Transfer Object）
type TechDTO struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Score       int    `json:"score"`
	Level       string `json:"level"` // fiction / emerging / reality
}

// toDTOは domain.Tech を TechDTO に変換する
func toDTO(t domain.Tech) TechDTO {
	return TechDTO{
		ID:          t.ID,
		Name:        t.Name,
		Description: t.Description,
		Score:       t.Score,
		Level:       string(t.Level()),
	}
}

// toDTOsは domain.Tech のスライスを TechDTO のスライスに変換する（複数形）
func toDTOs(techs []domain.Tech) []TechDTO {
	dtos := make([]TechDTO, len(techs))
	for i, t := range techs {
		dtos[i] = toDTO(t)
	}
	return dtos
}

// TechRequest は、追加（POST）と更新（PUT）で受け取る本文。
// 省略と 0 を区別したいので、score はポインターで受ける。
type TechRequest struct {
	Name        *string `json:"name"`
	Description string  `json:"description"`
	Score       *int    `json:"score"`
}

// toInput は必須の項目がそろっているかを確かめて、usecase の入力に変換する
func (r TechRequest) toInput() (usecase.TechInput, *domain.ValidationError) {
	if r.Name == nil {
		return usecase.TechInput{}, &domain.ValidationError{Field: "name", Message: "is required"}
	}
	if r.Score == nil {
		return usecase.TechInput{}, &domain.ValidationError{Field: "score", Message: "is required"}
	}
	return usecase.TechInput{Name: *r.Name, Description: r.Description, Score: *r.Score}, nil
}

// StatsDTO は /stats の応答
type StatsDTO struct {
	Count   int            `json:"count"`
	Average float64        `json:"average"` // 小数第 1 位まで
	Min     int            `json:"min"`
	Max     int            `json:"max"`
	Levels  map[string]int `json:"levels"`
}

func toStatsDTO(s usecase.Stats) StatsDTO {
	levels := make(map[string]int, len(s.Levels))
	for level, count := range s.Levels {
		levels[string(level)] = count
	}
	return StatsDTO{
		Count:   s.Count,
		Average: math.Round(s.Average*10) / 10,
		Min:     s.Min,
		Max:     s.Max,
		Levels:  levels,
	}
}

// ErrorDTO は、エラーのときの応答。どのエラーもこの形で返す
type ErrorDTO struct {
	Error ErrorBody `json:"error"`
}

// ErrorBody は、プログラムが見分けるための code と、人が読むための message を持つ
type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"` // 入力の誤りのとき、どの項目か
}
