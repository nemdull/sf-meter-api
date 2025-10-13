package presentation

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/nemdull/sf-meter-api/usecase"
)

// TechHandler は技術情報に関するHTTPハンドラー
type TechHandler struct {
	service *usecase.TechService
}

// NewTechHandlerはTechHandlerのインスタンスを生成して返す
func NewTechHandler(service *usecase.TechService) *TechHandler {
	return &TechHandler{service: service}
}

// GET /api/v1/techs
func (h *TechHandler) GetAllTechs(w http.ResponseWriter, r *http.Request) {
	// ビジネスロジックを呼び出して全技術情報を取得
	techs, err := h.service.GetAllTechs()
	if err != nil {
		http.Error(w, "Failed to get techs", http.StatusInternalServerError)
		return
	}

	// JSON形式でレスポンスを返す
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(toDTOs(techs)); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// GET /api/v1/tech?id={id}
func (h *TechHandler) GetTechByID(w http.ResponseWriter, r *http.Request) {
	// クエリパラメータからIDを取得
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		http.Error(w, "Missing id parameter", http.StatusBadRequest)
		return
	}

	// stringをintに変換
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		http.Error(w, "Invalid id parameter", http.StatusBadRequest)
		return
	}

	// ビジネスロジックを呼び出して指定IDの技術情報を取得
	tech, err := h.service.GetTechByID(id)
	if err != nil {
		http.Error(w, "Tech not found", http.StatusNotFound)
		return
	}
	// JSON形式でレスポンスを返す
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(toDTO(*tech)); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}

}
