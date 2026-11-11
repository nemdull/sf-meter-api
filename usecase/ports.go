package usecase

import "github.com/nemdull/sf-meter-api/domain"

// TechRepository は技術情報のデータ取得を抽象化するインターフェース
type TechRepository interface {
	FindAll() ([]domain.Tech, error)
	FindByID(id int) (*domain.Tech, error)
}
