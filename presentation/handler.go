package presentation

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/nemdull/sf-meter-api/domain"
)

// TechUseCase は TechHandler が依存するビジネスロジックを抽象化する。
// presentation層側で定義することで、usecase.TechServiceの具象型に縛られず、
// テストではモックを注入できる。
type TechUseCase interface {
	GetAll() ([]domain.Tech, error)
	GetByID(id int) (*domain.Tech, error)
}

// TechHandler は技術情報に関するHTTPハンドラー
type TechHandler struct {
	uc TechUseCase
}

// NewTechHandlerはTechHandlerのインスタンスを生成して返す
func NewTechHandler(uc TechUseCase) *TechHandler {
	return &TechHandler{uc: uc}
}

// GET /techs
func (h *TechHandler) GetTechs(w http.ResponseWriter, r *http.Request) {
	// ビジネスロジックを呼び出して全技術情報を取得
	techs, err := h.uc.GetAll()
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

// GET /techs/{id}
func (h *TechHandler) GetTechByID(w http.ResponseWriter, r *http.Request) {
	// パスパラメータからIDを取得(Go 1.22+ ServeMuxのパスパターン)
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		http.Error(w, "Invalid id parameter", http.StatusBadRequest)
		return
	}

	// ビジネスロジックを呼び出して指定IDの技術情報を取得
	tech, err := h.uc.GetByID(id)
	if err != nil {
		http.Error(w, "Failed to get tech", http.StatusInternalServerError)
		return
	}
	if tech == nil {
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
