package presentation

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"log/slog"
	"net/http"
	"runtime/debug"
	"slices"
	"strings"
	"time"
)

type contextKey string

const requestIDKey contextKey = "request_id"

// requestIDFrom は、リクエストに振った番号を取り出す。振られていなければ空
func requestIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}

func newRequestID() string {
	buffer := make([]byte, 8)
	if _, err := rand.Read(buffer); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(buffer)
}

// withRequestID は、リクエストごとに番号を振って X-Request-ID で返す。
// 記録と突き合わせるためのもの。外から来た値は信用せず、必ずこちらで振り直す
func withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := newRequestID()
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey, id)))
	})
}

// statusRecorder は、あとで記録に残すために、返したステータスを覚えておく
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

// withLogging は、1 リクエストにつき 1 行の記録を残す
func withLogging(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(recorder, r)
			logger.InfoContext(r.Context(), "request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", recorder.status,
				"duration_ms", time.Since(started).Milliseconds(),
				"request_id", requestIDFrom(r.Context()),
			)
		})
	}
}

// withRecover は、ハンドラーの中で panic が起きてもサーバーごと落ちないようにする
func withRecover(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					logger.ErrorContext(r.Context(), "panic",
						"error", recovered,
						"stack", string(debug.Stack()),
						"request_id", requestIDFrom(r.Context()),
					)
					writeError(w, http.StatusInternalServerError, codeInternal, "internal server error")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// withCORS は、別のオリジンにある画面から API を呼べるようにする。
// 許すオリジンは一覧で渡す。"*" が入っていれば、どこからでも読める（書き込みはトークンが要るので、読み取りだけが開く）
func withCORS(allowedOrigins []string) func(http.Handler) http.Handler {
	anyOrigin := slices.Contains(allowedOrigins, "*")
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" && (anyOrigin || slices.Contains(allowedOrigins, origin)) {
				header := w.Header()
				header.Add("Vary", "Origin")
				if anyOrigin {
					header.Set("Access-Control-Allow-Origin", "*")
				} else {
					header.Set("Access-Control-Allow-Origin", origin)
				}
				header.Set("Access-Control-Expose-Headers", "X-Total-Count, X-Request-ID")
				// 事前確認（preflight）には、ここで答えて終わる
				if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
					header.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
					header.Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
					header.Set("Access-Control-Max-Age", "600")
					w.WriteHeader(http.StatusNoContent)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// requireAdmin は、書き込みの操作を管理用のトークンで守る。
//   - トークンが設定されていなければ、書き込みそのものを閉じる（403）
//   - Authorization: Bearer <トークン> が合わなければ 401
//
// 比較は、かかった時間から中身を推測されないように ConstantTimeCompare を使う
func requireAdmin(token string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if token == "" {
			writeError(w, http.StatusForbidden, codeWritesDisabled, "write operations are disabled on this server")
			return
		}
		given, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || subtle.ConstantTimeCompare([]byte(given), []byte(token)) != 1 {
			w.Header().Set("WWW-Authenticate", `Bearer realm="sf-meter"`)
			writeError(w, http.StatusUnauthorized, codeUnauthorized, "a valid admin token is required")
			return
		}
		next(w, r)
	}
}
