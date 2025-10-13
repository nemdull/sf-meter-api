package presentation

import "net/http"

// NewRouter はルーターのインスタンスを生成して返す
func NewRouter(handler *TechHandler) http.Handler {
	// ルーティングの設定
	mux := http.NewServeMux()

	// GET /api/v1/techs -> 全技術情報の取得
	mux.HandleFunc("/api/v1/techs", handler.GetAllTechs)

	// GET /api/v1/tech?id={id} -> 指定IDの技術情報の取得
	mux.HandleFunc("/api/v1/tech", handler.GetTechByID)

	return mux
}
