package presentation

import "github.com/nemdull/sf-meter-api/domain"

// TechDTO は技術情報のDTO（Data Transfer Object）
type TechDTO struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Score       int    `json:"score"`
}

// toDTOは domain.Tech を TechDTO に変換する
func toDTO(t domain.Tech) TechDTO {
	return TechDTO{
		ID:          t.ID,
		Name:        t.Name,
		Description: t.Description,
		Score:       t.Score,
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
