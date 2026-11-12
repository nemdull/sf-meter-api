package presentation

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/nemdull/sf-meter-api/domain"
	"github.com/nemdull/sf-meter-api/usecase"
)

// 一覧の件数。指定が無ければ defaultLimit、多くても maxLimit まで
const (
	defaultLimit = 50
	maxLimit     = 100
	// 追加・更新で受け取る本文の上限（バイト）
	maxBodyBytes = 16 << 10
)

// TechUseCase は TechHandler が依存するビジネスロジックを抽象化する。
// presentation層側で定義することで、usecase.TechServiceの具象型に縛られず、
// テストではモックを注入できる。
type TechUseCase interface {
	List(ctx context.Context, query usecase.TechQuery) (usecase.TechList, error)
	Get(ctx context.Context, id int) (domain.Tech, error)
	Random(ctx context.Context) (domain.Tech, error)
	Stats(ctx context.Context) (usecase.Stats, error)
	Create(ctx context.Context, input usecase.TechInput) (domain.Tech, error)
	Update(ctx context.Context, id int, input usecase.TechInput) (domain.Tech, error)
	Delete(ctx context.Context, id int) error
}

// TechHandler は技術情報に関するHTTPハンドラー
type TechHandler struct {
	uc TechUseCase
}

// NewTechHandlerはTechHandlerのインスタンスを生成して返す
func NewTechHandler(uc TechUseCase) *TechHandler {
	return &TechHandler{uc: uc}
}

// intParam は、クエリの数値を読む。無ければ fallback、数値でなければエラー
func intParam(r *http.Request, name string, fallback int) (int, error) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer", name)
	}
	return value, nil
}

// parseQuery は、クエリ文字列を一覧の条件に変換する
func parseQuery(r *http.Request) (usecase.TechQuery, error) {
	query := usecase.AllTechs()
	query.Keyword = strings.TrimSpace(r.URL.Query().Get("q"))

	var err error
	if query.MinScore, err = intParam(r, "min_score", domain.MinScore); err != nil {
		return query, err
	}
	if query.MaxScore, err = intParam(r, "max_score", domain.MaxScore); err != nil {
		return query, err
	}
	if query.MinScore > query.MaxScore {
		return query, fmt.Errorf("min_score must not be greater than max_score")
	}

	if sort := r.URL.Query().Get("sort"); sort != "" {
		query.Sort = usecase.SortKey(sort)
		if !usecase.ValidSortKey(query.Sort) {
			return query, fmt.Errorf("sort must be one of id, score, -score, name")
		}
	}

	if query.Limit, err = intParam(r, "limit", defaultLimit); err != nil {
		return query, err
	}
	if query.Limit < 1 || query.Limit > maxLimit {
		return query, fmt.Errorf("limit must be between 1 and %d", maxLimit)
	}
	if query.Offset, err = intParam(r, "offset", 0); err != nil {
		return query, err
	}
	if query.Offset < 0 {
		return query, fmt.Errorf("offset must not be negative")
	}
	return query, nil
}

// pathID は、パスの {id} を読む。正の整数でなければ 400 を返して false
func pathID(w http.ResponseWriter, r *http.Request) (int, bool) {
	// パスパラメータからIDを取得(Go 1.22+ ServeMuxのパスパターン)
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, codeInvalidParameter, "id must be a positive integer")
		return 0, false
	}
	return id, true
}

// decodeBody は、本文の JSON を読む。大きすぎる・形が違う・知らない項目がある、はどれも 400
func decodeBody(w http.ResponseWriter, r *http.Request) (usecase.TechInput, bool) {
	var body TechRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, codeInvalidBody, "request body must be a JSON object with name, description and score")
		return usecase.TechInput{}, false
	}
	input, invalid := body.toInput()
	if invalid != nil {
		writeUseCaseError(w, r, invalid)
		return usecase.TechInput{}, false
	}
	return input, true
}

// GetTechs は一覧を返す。GET /api/v1/techs?q=&min_score=&max_score=&sort=&limit=&offset=
// 本文は配列。絞り込み後の総件数は X-Total-Count ヘッダーに入る
func (h *TechHandler) GetTechs(w http.ResponseWriter, r *http.Request) {
	query, err := parseQuery(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, codeInvalidParameter, err.Error())
		return
	}

	// ビジネスロジックを呼び出して技術情報を取得
	list, err := h.uc.List(r.Context(), query)
	if err != nil {
		writeUseCaseError(w, r, err)
		return
	}

	w.Header().Set("X-Total-Count", strconv.Itoa(list.Total))
	writeJSON(w, http.StatusOK, toDTOs(list.Items))
}

// GetTechByID は 1 件を返す。GET /api/v1/techs/{id}
func (h *TechHandler) GetTechByID(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	tech, err := h.uc.Get(r.Context(), id)
	if err != nil {
		writeUseCaseError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toDTO(tech))
}

// GetRandomTech は、どれか 1 件を選んで返す。GET /api/v1/techs/random
func (h *TechHandler) GetRandomTech(w http.ResponseWriter, r *http.Request) {
	tech, err := h.uc.Random(r.Context())
	if err != nil {
		writeUseCaseError(w, r, err)
		return
	}
	// 毎回違う結果を返すので、途中でキャッシュされないようにする
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, toDTO(tech))
}

// GetStats は全体の集計を返す。GET /api/v1/stats
func (h *TechHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.uc.Stats(r.Context())
	if err != nil {
		writeUseCaseError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toStatsDTO(stats))
}

// CreateTech は 1 件を追加する。POST /api/v1/techs（管理用のトークンが要る）
func (h *TechHandler) CreateTech(w http.ResponseWriter, r *http.Request) {
	input, ok := decodeBody(w, r)
	if !ok {
		return
	}
	tech, err := h.uc.Create(r.Context(), input)
	if err != nil {
		writeUseCaseError(w, r, err)
		return
	}
	w.Header().Set("Location", fmt.Sprintf("%s/techs/%d", apiPrefix, tech.ID))
	writeJSON(w, http.StatusCreated, toDTO(tech))
}

// UpdateTech は 1 件を置き換える。PUT /api/v1/techs/{id}（管理用のトークンが要る）
func (h *TechHandler) UpdateTech(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	input, ok := decodeBody(w, r)
	if !ok {
		return
	}
	tech, err := h.uc.Update(r.Context(), id, input)
	if err != nil {
		writeUseCaseError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toDTO(tech))
}

// DeleteTech は 1 件を消す。DELETE /api/v1/techs/{id}（管理用のトークンが要る）
func (h *TechHandler) DeleteTech(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if err := h.uc.Delete(r.Context(), id); err != nil {
		writeUseCaseError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
