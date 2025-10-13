package registry

import (
	"net/http"

	"github.com/nemdull/sf-meter-api/domain"
)

func NewHTTPServer(addr string) *http.Server {
	// モックデータ
	seed := []domain.Tech{
		{ID: 1, Name: "ドミネーター", Description: "犯罪係数で裁く未来銃", Score: 35},
		{ID: 2, Name: "タイムマシン", Description: "時間を超える装置", Score: 5},
		{ID: 3, Name: "量子通信", Description: "瞬時に情報伝送（量子もつれ）", Score: 60},
	}

	// リポジトリ実装を選択
	repo := domain.NewInMemoryTechRepository(seed)

	// Usecaseにリポジトリを注入
	service := domain.NewTechService(repo)

	// ハンドラを生成
	handler := NewTechHandler(service)

	//ルーター生成
	router := presentation.NewRouter(handler)

	return &http.Server{
		Addr:    addr,
		Handler: router,
	}
}
