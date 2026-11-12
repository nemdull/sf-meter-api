package presentation

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

// apiPrefix は、いまの API のパスの先頭
const apiPrefix = "/api/v1"

// Pinger は、保存先につながるかを確かめられるもの（PostgreSQL 実装が満たす）
type Pinger interface {
	Ping(ctx context.Context) error
}

// RouterOptions は、ルーターを組み立てるときの設定
type RouterOptions struct {
	// AdminToken は、追加・更新・削除に要るトークン。空なら書き込みを閉じる
	AdminToken string
	// AllowedOrigins は、ブラウザーから API を呼んでよいオリジン
	AllowedOrigins []string
	// Store は、いまの保存先の名前（memory / postgres）。/health に出す
	Store string
	// Pinger があれば、/health で保存先につながるかも確かめる
	Pinger Pinger
	Logger *slog.Logger
}

// Routes は、公開しているパスの一覧（メソッドとパス）を返す。
// docs/openapi.yaml と食い違っていないかを、テストが突き合わせる
func Routes() []string {
	return []string{
		"GET " + apiPrefix + "/techs",
		"GET " + apiPrefix + "/techs/random",
		"GET " + apiPrefix + "/techs/{id}",
		"POST " + apiPrefix + "/techs",
		"PUT " + apiPrefix + "/techs/{id}",
		"DELETE " + apiPrefix + "/techs/{id}",
		"GET " + apiPrefix + "/stats",
		"GET /health",
	}
}

// NewRouter は、パスとハンドラーを結び付け、共通の処理（記録・CORS など）をかぶせて返す
func NewRouter(h *TechHandler, opts RouterOptions) http.Handler {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	// Go 1.22 以降の ServeMux は、メソッドと {id} のようなパスの変数を扱える
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+apiPrefix+"/techs", h.GetTechs)
	mux.HandleFunc("GET "+apiPrefix+"/techs/random", h.GetRandomTech)
	mux.HandleFunc("GET "+apiPrefix+"/techs/{id}", h.GetTechByID)
	mux.HandleFunc("POST "+apiPrefix+"/techs", requireAdmin(opts.AdminToken, h.CreateTech))
	mux.HandleFunc("PUT "+apiPrefix+"/techs/{id}", requireAdmin(opts.AdminToken, h.UpdateTech))
	mux.HandleFunc("DELETE "+apiPrefix+"/techs/{id}", requireAdmin(opts.AdminToken, h.DeleteTech))
	mux.HandleFunc("GET "+apiPrefix+"/stats", h.GetStats)
	mux.HandleFunc("GET /health", health(opts))

	// /api/v1 を付ける前のパス。前から使っている人のために、読み取りだけ残してある
	mux.HandleFunc("GET /techs", h.GetTechs)
	mux.HandleFunc("GET /techs/{id}", h.GetTechByID)

	// 外側から順に: 番号を振る → 記録する → panic を受け止める → CORS → 各ハンドラー
	handler := withJSONFallback(mux)
	handler = withCORS(opts.AllowedOrigins)(handler)
	handler = withRecover(logger)(handler)
	handler = withLogging(logger)(handler)
	handler = withRequestID(handler)
	return handler
}

// probe は、応答を実際には送らずに、ステータスとヘッダーだけを受け取る
type probe struct {
	header http.Header
	status int
}

func (p *probe) Header() http.Header         { return p.header }
func (p *probe) Write(b []byte) (int, error) { return len(b), nil }
func (p *probe) WriteHeader(status int)      { p.status = status }

// withJSONFallback は、どのパスにも当たらない（404）・メソッドが違う（405）ときの応答を、
// ほかのエラーと同じ形の JSON にする。標準の ServeMux は、この 2 つを文字だけで返すため
func withJSONFallback(mux *http.ServeMux) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fallback, pattern := mux.Handler(r)
		if pattern != "" {
			mux.ServeHTTP(w, r)
			return
		}
		// 標準の応答を空振りさせて、404 と 405 のどちらなのか、使えるメソッドは何かを聞き出す
		asked := &probe{header: http.Header{}, status: http.StatusNotFound}
		fallback.ServeHTTP(asked, r)
		if asked.status == http.StatusMethodNotAllowed {
			w.Header().Set("Allow", asked.header.Get("Allow"))
			writeError(w, http.StatusMethodNotAllowed, codeMethodNotAllowed, "method not allowed")
			return
		}
		writeError(w, http.StatusNotFound, codeNotFound, "no such endpoint")
	})
}

// health は、サーバーが動いているかを返す。保存先につながらなければ 503
func health(opts RouterOptions) http.HandlerFunc {
	type response struct {
		Status string `json:"status"`
		Store  string `json:"store"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if opts.Pinger != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()
			if err := opts.Pinger.Ping(ctx); err != nil {
				writeJSON(w, http.StatusServiceUnavailable, response{Status: "unavailable", Store: opts.Store})
				return
			}
		}
		writeJSON(w, http.StatusOK, response{Status: "ok", Store: opts.Store})
	}
}
