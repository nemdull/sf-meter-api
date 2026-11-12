package presentation

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/nemdull/sf-meter-api/domain"
)

// エラーの code。画面や利用側のプログラムは、文言ではなくこれを見て分岐する
const (
	codeInvalidParameter = "invalid_parameter"
	codeInvalidBody      = "invalid_body"
	codeValidationFailed = "validation_failed"
	codeNotFound         = "not_found"
	codeMethodNotAllowed = "method_not_allowed"
	codeUnauthorized     = "unauthorized"
	codeWritesDisabled   = "writes_disabled"
	codeInternal         = "internal_error"
)

// writeJSON は、値を JSON にして返す
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		// ここまで来るとヘッダーは送ったあとなので、記録だけ残す
		slog.Error("応答を書き出せなかった", "error", err)
	}
}

// writeError は、エラーを決まった形の JSON で返す
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, ErrorDTO{Error: ErrorBody{Code: code, Message: message}})
}

// writeUseCaseError は、usecase から返ってきたエラーを HTTP の応答に変換する。
// 中で何が起きたかは外に出さず、記録にだけ残す
func writeUseCaseError(w http.ResponseWriter, r *http.Request, err error) {
	var invalid *domain.ValidationError
	switch {
	case errors.As(err, &invalid):
		writeJSON(w, http.StatusBadRequest, ErrorDTO{Error: ErrorBody{
			Code:    codeValidationFailed,
			Message: invalid.Field + " " + invalid.Message,
			Field:   invalid.Field,
		}})
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, http.StatusNotFound, codeNotFound, "tech not found")
	default:
		slog.ErrorContext(r.Context(), "処理に失敗した", "error", err, "path", r.URL.Path, "request_id", requestIDFrom(r.Context()))
		writeError(w, http.StatusInternalServerError, codeInternal, "internal server error")
	}
}
