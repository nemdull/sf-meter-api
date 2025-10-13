package infrastructure

import (
	"errors"

	"github.com/nemdull/sf-meter-api/domain"
)

// TechRepoMemoryはusecaseが使うTechRepositoryのメモリ実装
// ここでは簡単のためにメモリ実装を使うが、実際はDBや外部APIなどの実装になる
type TechRepoMemory struct {
	data map[int]domain.Tech
}

// 初期データseedを受け取り、TechRepoMemoryのインスタンスを生成して返す
func NewTechRepoMemory(seed []domain.Tech) *TechRepoMemory {
	// 初期データをマップに格納
	m := &TechRepoMemory{data: make(map[int]domain.Tech)}
	for _, tech := range seed {
		m.data[tech.ID] = tech
	}
	return m
}

// GetAllは全てのTechデータを取得する
func (m *TechRepoMemory) GetAll() ([]domain.Tech, error) {
	techs := make([]domain.Tech, 0, len(m.data))
	for _, tech := range m.data {
		techs = append(techs, tech)
	}
	return techs, nil
}

// GetByIDはIDでTechデータを取得する
func (m *TechRepoMemory) GetByID(id int) (domain.Tech, error) {
	if tech, ok := m.data[id]; ok {
		return tech, nil
	}
	return domain.Tech{}, errors.New("ID not found")
}
