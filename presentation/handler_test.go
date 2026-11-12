package presentation_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nemdull/sf-meter-api/infrastructure"
	"github.com/nemdull/sf-meter-api/presentation"
	"github.com/nemdull/sf-meter-api/usecase"
)

// このファイルのテストは、ルーター → ハンドラー → usecase → メモリ実装まで、本物をつないで通す。
// 外から見える振る舞い（パス・ステータス・JSON の形）を、まとめて確かめるため

const adminToken = "test-admin-token"

var quiet = slog.New(slog.NewTextHandler(io.Discard, nil))

// newServer は、初期データ 15 件が入ったサーバーを作る
func newServer(t *testing.T, opts presentation.RouterOptions) http.Handler {
	t.Helper()
	repo := infrastructure.NewMemoryTechRepo(infrastructure.SeedTechs()...)
	handler := presentation.NewTechHandler(usecase.NewTechService(repo))
	if opts.Logger == nil {
		opts.Logger = quiet
	}
	if opts.Store == "" {
		opts.Store = "memory"
	}
	return presentation.NewRouter(handler, opts)
}

type request struct {
	method, path, body, token string
	header                    map[string]string
}

func do(t *testing.T, server http.Handler, req request) *httptest.ResponseRecorder {
	t.Helper()
	httpReq := httptest.NewRequest(req.method, req.path, strings.NewReader(req.body))
	if req.body != "" {
		httpReq.Header.Set("Content-Type", "application/json")
	}
	if req.token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+req.token)
	}
	for name, value := range req.header {
		httpReq.Header.Set(name, value)
	}
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, httpReq)
	return rec
}

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var value T
	if err := json.NewDecoder(bytes.NewReader(rec.Body.Bytes())).Decode(&value); err != nil {
		t.Fatalf("JSON として読めない: %v\n%s", err, rec.Body.String())
	}
	return value
}

func names(techs []presentation.TechDTO) string {
	out := make([]string, len(techs))
	for i, tech := range techs {
		out[i] = tech.Name
	}
	return strings.Join(out, ",")
}

// ─── GET /api/v1/techs ───────────────────────────────────────────────────────

func TestGetTechs_OK(t *testing.T) {
	rec := do(t, newServer(t, presentation.RouterOptions{}), request{method: "GET", path: "/api/v1/techs"})

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q", got)
	}
	if got := rec.Header().Get("X-Total-Count"); got != "15" {
		t.Errorf("X-Total-Count = %q、期待は 15", got)
	}
	techs := decode[[]presentation.TechDTO](t, rec)
	if len(techs) != 15 {
		t.Fatalf("expected 15 techs, got %d", len(techs))
	}
	first := techs[0]
	if first.ID != 1 || first.Name != "ドミネーター" || first.Score != 35 || first.Level != "emerging" {
		t.Errorf("unexpected first tech: %+v", first)
	}
}

func TestGetTechs_FilterSortPaginate(t *testing.T) {
	server := newServer(t, presentation.RouterOptions{})
	tests := []struct {
		name      string
		query     string
		wantNames string
		wantTotal string
	}{
		{"実現度の高い順に 3 件", "sort=-score&limit=3", "シビラシステム,量子暗号通信,ニューラルリンク", "15"},
		{"実現度の低い順の 2 ページ目", "sort=score&limit=2&offset=2", "反重力エンジン,AI人格移植", "15"},
		{"文字で絞り込む", "q=通信", "ホログラム通信,量子暗号通信", "2"},
		{"英字は大文字と小文字を区別しない", "q=bci", "ニューラルリンク", "1"},
		{"実現度の範囲で絞り込む", "min_score=80&max_score=90", "ニューラルリンク,量子暗号通信", "2"},
		{"当たらなければ空の配列", "q=ワープ", "", "0"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(t, server, request{method: "GET", path: "/api/v1/techs?" + tc.query})
			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
			}
			if got := names(decode[[]presentation.TechDTO](t, rec)); got != tc.wantNames {
				t.Errorf("got  %s\nwant %s", got, tc.wantNames)
			}
			if got := rec.Header().Get("X-Total-Count"); got != tc.wantTotal {
				t.Errorf("X-Total-Count = %q、期待は %q", got, tc.wantTotal)
			}
			if strings.TrimSpace(rec.Body.String()) == "null" {
				t.Error("0 件のときも null ではなく [] を返す")
			}
		})
	}
}

func TestGetTechs_InvalidParameter(t *testing.T) {
	server := newServer(t, presentation.RouterOptions{})
	for _, query := range []string{
		"limit=0", "limit=101", "limit=abc", "offset=-1", "sort=price",
		"min_score=x", "min_score=80&max_score=20",
	} {
		t.Run(query, func(t *testing.T) {
			rec := do(t, server, request{method: "GET", path: "/api/v1/techs?" + query})
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d", rec.Code)
			}
			if code := decode[presentation.ErrorDTO](t, rec).Error.Code; code != "invalid_parameter" {
				t.Errorf("code = %q", code)
			}
		})
	}
}

// ─── GET /api/v1/techs/{id} ──────────────────────────────────────────────────

func TestGetTechByID(t *testing.T) {
	server := newServer(t, presentation.RouterOptions{})
	tests := []struct {
		path       string
		wantStatus int
		wantCode   string
	}{
		{"/api/v1/techs/8", http.StatusOK, ""},
		{"/api/v1/techs/999", http.StatusNotFound, "not_found"},
		{"/api/v1/techs/abc", http.StatusBadRequest, "invalid_parameter"},
		{"/api/v1/techs/0", http.StatusBadRequest, "invalid_parameter"},
		{"/api/v1/techs/-3", http.StatusBadRequest, "invalid_parameter"},
	}
	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			rec := do(t, server, request{method: "GET", path: tc.path})
			if rec.Code != tc.wantStatus {
				t.Fatalf("expected %d, got %d", tc.wantStatus, rec.Code)
			}
			if tc.wantCode == "" {
				tech := decode[presentation.TechDTO](t, rec)
				if tech.Name != "量子暗号通信" || tech.Level != "reality" {
					t.Errorf("unexpected tech: %+v", tech)
				}
				return
			}
			if code := decode[presentation.ErrorDTO](t, rec).Error.Code; code != tc.wantCode {
				t.Errorf("code = %q、期待は %q", code, tc.wantCode)
			}
		})
	}
}

// ─── GET /api/v1/techs/random, /api/v1/stats ─────────────────────────────────

func TestGetRandomTech(t *testing.T) {
	rec := do(t, newServer(t, presentation.RouterOptions{}), request{method: "GET", path: "/api/v1/techs/random"})

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if tech := decode[presentation.TechDTO](t, rec); tech.ID < 1 || tech.ID > 15 {
		t.Errorf("初期データのどれかが返るはずが: %+v", tech)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q", got)
	}
}

func TestGetStats(t *testing.T) {
	rec := do(t, newServer(t, presentation.RouterOptions{}), request{method: "GET", path: "/api/v1/stats"})

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	stats := decode[presentation.StatsDTO](t, rec)
	// 初期データ 15 件の合計は 703。703 / 15 = 46.86… を小数第 1 位に丸める
	if stats.Count != 15 || stats.Min != 1 || stats.Max != 91 || stats.Average != 46.9 {
		t.Errorf("unexpected stats: %+v", stats)
	}
	if stats.Levels["fiction"] != 5 || stats.Levels["emerging"] != 6 || stats.Levels["reality"] != 4 {
		t.Errorf("unexpected levels: %v", stats.Levels)
	}
}

// ─── 書き込み（POST / PUT / DELETE） ─────────────────────────────────────────

func TestWrites_DisabledWithoutAdminToken(t *testing.T) {
	server := newServer(t, presentation.RouterOptions{}) // トークンを設定していないサーバー
	for _, req := range []request{
		{method: "POST", path: "/api/v1/techs", body: `{"name":"x","score":1}`},
		{method: "PUT", path: "/api/v1/techs/1", body: `{"name":"x","score":1}`},
		{method: "DELETE", path: "/api/v1/techs/1"},
		{method: "POST", path: "/api/v1/techs", body: `{"name":"x","score":1}`, token: "anything"},
	} {
		rec := do(t, server, req)
		if rec.Code != http.StatusForbidden {
			t.Errorf("%s %s: expected 403, got %d", req.method, req.path, rec.Code)
		}
		if code := decode[presentation.ErrorDTO](t, rec).Error.Code; code != "writes_disabled" {
			t.Errorf("%s %s: code = %q", req.method, req.path, code)
		}
	}
}

func TestWrites_RequireValidToken(t *testing.T) {
	server := newServer(t, presentation.RouterOptions{AdminToken: adminToken})
	body := `{"name":"x","score":1}`
	tests := []struct {
		name   string
		header map[string]string
	}{
		{"トークンなし", nil},
		{"違うトークン", map[string]string{"Authorization": "Bearer wrong"}},
		{"Bearer が付いていない", map[string]string{"Authorization": adminToken}},
		{"前半だけ合っている", map[string]string{"Authorization": "Bearer test-admin"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(t, server, request{method: "POST", path: "/api/v1/techs", body: body, header: tc.header})
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401, got %d", rec.Code)
			}
			if rec.Header().Get("WWW-Authenticate") == "" {
				t.Error("WWW-Authenticate が付いていない")
			}
		})
	}

	// 弾かれた書き込みは、1 件も入っていないこと
	rec := do(t, server, request{method: "GET", path: "/api/v1/techs"})
	if got := rec.Header().Get("X-Total-Count"); got != "15" {
		t.Errorf("件数が変わっている: %s", got)
	}
}

func TestWrites_CreateUpdateDelete(t *testing.T) {
	server := newServer(t, presentation.RouterOptions{AdminToken: adminToken})

	// 追加
	rec := do(t, server, request{method: "POST", path: "/api/v1/techs", token: adminToken,
		body: `{"name":" 軌道エレベーター ","description":"宇宙まで届く昇降機","score":20}`})
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	created := decode[presentation.TechDTO](t, rec)
	if created.ID != 16 || created.Name != "軌道エレベーター" || created.Level != "fiction" {
		t.Errorf("unexpected tech: %+v", created)
	}
	if got := rec.Header().Get("Location"); got != "/api/v1/techs/16" {
		t.Errorf("Location = %q", got)
	}

	// 更新（説明を省くと空になる。0 点も入れられる）
	rec = do(t, server, request{method: "PUT", path: "/api/v1/techs/16", token: adminToken,
		body: `{"name":"軌道エレベーター","score":0}`})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if updated := decode[presentation.TechDTO](t, rec); updated.Score != 0 || updated.Description != "" {
		t.Errorf("unexpected tech: %+v", updated)
	}

	// 削除 → もう一度消そうとすると 404
	if rec = do(t, server, request{method: "DELETE", path: "/api/v1/techs/16", token: adminToken}); rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	if rec = do(t, server, request{method: "DELETE", path: "/api/v1/techs/16", token: adminToken}); rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
	if rec = do(t, server, request{method: "PUT", path: "/api/v1/techs/16", token: adminToken, body: `{"name":"x","score":1}`}); rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestWrites_InvalidBody(t *testing.T) {
	server := newServer(t, presentation.RouterOptions{AdminToken: adminToken})
	tests := []struct {
		name      string
		body      string
		wantCode  string
		wantField string
	}{
		{"JSON ではない", `name=x`, "invalid_body", ""},
		{"空", ``, "invalid_body", ""},
		{"知らない項目がある", `{"name":"x","score":1,"admin":true}`, "invalid_body", ""},
		{"型が違う", `{"name":"x","score":"高い"}`, "invalid_body", ""},
		{"大きすぎる", `{"name":"x","score":1,"description":"` + strings.Repeat("あ", 20000) + `"}`, "invalid_body", ""},
		{"名前が無い", `{"score":1}`, "validation_failed", "name"},
		{"実現度が無い", `{"name":"x"}`, "validation_failed", "score"},
		{"名前が空白だけ", `{"name":"  ","score":1}`, "validation_failed", "name"},
		{"実現度がはみ出す", `{"name":"x","score":101}`, "validation_failed", "score"},
		{"説明が長すぎる", `{"name":"x","score":1,"description":"` + strings.Repeat("あ", 501) + `"}`, "validation_failed", "description"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(t, server, request{method: "POST", path: "/api/v1/techs", token: adminToken, body: tc.body})
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
			}
			got := decode[presentation.ErrorDTO](t, rec).Error
			if got.Code != tc.wantCode || got.Field != tc.wantField {
				t.Errorf("code = %q field = %q、期待は %q / %q", got.Code, got.Field, tc.wantCode, tc.wantField)
			}
		})
	}

	rec := do(t, server, request{method: "GET", path: "/api/v1/techs"})
	if got := rec.Header().Get("X-Total-Count"); got != "15" {
		t.Errorf("弾かれた入力が保存されている: %s 件", got)
	}
}
