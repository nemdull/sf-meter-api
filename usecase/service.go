package usecase

import "github.com/nemdull/sf-meter-api/domain"

// TechService は技術情報に関するビジネスロジックを提供する
type TechService struct {
	repo TechRepository
}

// NewTechService はTechServiceのインスタンスを生成して返す
func NewTechService(repo TechRepository) *TechService {
	return &TechService{repo: repo}
}

// GetAllTechs はすべての技術情報を取得する
func (s *TechService) GetAllTechs() ([]domain.Tech, error) {
	return s.repo.GetAll()
}

// GetTechByID はIDに基づいて技術情報を取得する
func (s *TechService) GetTechByID(id int) (*domain.Tech, error) {
	return s.repo.GetByID(id)
}
