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

// GetAll はすべての技術情報を取得する
func (s *TechService) GetAll() ([]domain.Tech, error) {
	return s.repo.FindAll()
}

// GetByID はIDに基づいて技術情報を取得する
func (s *TechService) GetByID(id int) (*domain.Tech, error) {
	return s.repo.FindByID(id)
}
