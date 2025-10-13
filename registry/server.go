package registry

import (
	"net/http"

	"github.com/nemdull/sf-meter-api/domain"
	"github.com/nemdull/sf-meter-api/infrastructure"
	"github.com/nemdull/sf-meter-api/usecase"
)

// NewHTTPServer はHTTPサーバーのインスタンスを生成して返す
// 各層の依存関係をまとめる場所
// domain -> usecase -> presentation -> registryでDIを行う
func NewHTTPServer(addr string) *http.Server {
	// モックデータ
	seed := []domain.Tech{
		{ID: 1, Name: "ドミネーター", Description: "犯罪係数で裁く未来銃", Score: 35},
		{ID: 2, Name: "タイムマシン", Description: "時間を超える装置", Score: 5},
		{ID: 3, Name: "量子通信", Description: "瞬時に情報伝送（量子もつれ）", Score: 60},
	}

	// infrastructure層：データ取得の実装
	repo := infrastructure.NewTechRepoMemory(seed)

	// usecase層：ビジネスロジック（TechService）の実装
	service := usecase.NewTechService(repo)

	// presentation層：HTTPハンドラーの実装
	handler := NewTechHandler(service)

	// presentation層：ルーターの実装
	router := presentation.NewRouter(handler)

	// HTTPサーバーのインスタンスを生成して返す
	return &http.Server{
		Addr:    addr,
		Handler: router,
	}
}
