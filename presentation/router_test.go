package presentation_test

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/nemdull/sf-meter-api/domain"
	"github.com/nemdull/sf-meter-api/presentation"
	"github.com/nemdull/sf-meter-api/usecase"
)

// ─── どのパスにも当たらない・メソッドが違う ──────────────────────────────────

func TestRouter_UnknownPathAndMethod(t *testing.T) {
	server := newServer(t, presentation.RouterOptions{AdminToken: adminToken})
	tests := []struct {
		method, path string
		wantStatus   int
		wantCode     string
	}{
		{"GET", "/", http.StatusNotFound, "not_found"},
		{"GET", "/api/v2/techs", http.StatusNotFound, "not_found"},
		{"GET", "/api/v1/techs/1/extra", http.StatusNotFound, "not_found"},
		{"PATCH", "/api/v1/techs/1", http.StatusMethodNotAllowed, "method_not_allowed"},
		{"DELETE", "/api/v1/stats", http.StatusMethodNotAllowed, "method_not_allowed"},
		// 前のパスは読み取りだけ。書き込みは /api/v1 にしか無い
		{"POST", "/techs", http.StatusMethodNotAllowed, "method_not_allowed"},
	}
	for _, tc := range tests {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			rec := do(t, server, request{method: tc.method, path: tc.path, token: adminToken})
			if rec.Code != tc.wantStatus {
				t.Fatalf("expected %d, got %d", tc.wantStatus, rec.Code)
			}
			if code := decode[presentation.ErrorDTO](t, rec).Error.Code; code != tc.wantCode {
				t.Errorf("code = %q、期待は %q", code, tc.wantCode)
			}
			if tc.wantStatus == http.StatusMethodNotAllowed && rec.Header().Get("Allow") == "" {
				t.Error("405 には、使えるメソッド（Allow）を付ける")
			}
		})
	}
}

// /api/v1 を付ける前のパスも、同じ結果を返すこと
func TestRouter_LegacyPathsStillWork(t *testing.T) {
	server := newServer(t, presentation.RouterOptions{})
	for _, pair := range [][2]string{
		{"/techs", "/api/v1/techs"},
		{"/techs/3", "/api/v1/techs/3"},
		{"/techs?sort=-score&limit=2", "/api/v1/techs?sort=-score&limit=2"},
	} {
		legacy := do(t, server, request{method: "GET", path: pair[0]})
		current := do(t, server, request{method: "GET", path: pair[1]})
		if legacy.Code != http.StatusOK || legacy.Body.String() != current.Body.String() {
			t.Errorf("%s と %s の結果が違う", pair[0], pair[1])
		}
	}
}

// ─── /health ─────────────────────────────────────────────────────────────────

type fakePinger struct{ err error }

func (p fakePinger) Ping(context.Context) error { return p.err }

func TestHealth(t *testing.T) {
	type health struct {
		Status string `json:"status"`
		Store  string `json:"store"`
	}
	tests := []struct {
		name       string
		opts       presentation.RouterOptions
		wantStatus int
		want       health
	}{
		{"メモリ実装", presentation.RouterOptions{}, http.StatusOK, health{"ok", "memory"}},
		{"DB につながる", presentation.RouterOptions{Store: "postgres", Pinger: fakePinger{}}, http.StatusOK, health{"ok", "postgres"}},
		{"DB につながらない", presentation.RouterOptions{Store: "postgres", Pinger: fakePinger{err: errors.New("down")}}, http.StatusServiceUnavailable, health{"unavailable", "postgres"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(t, newServer(t, tc.opts), request{method: "GET", path: "/health"})
			if rec.Code != tc.wantStatus {
				t.Fatalf("expected %d, got %d", tc.wantStatus, rec.Code)
			}
			if got := decode[health](t, rec); got != tc.want {
				t.Errorf("got %+v、want %+v", got, tc.want)
			}
		})
	}
}

// ─── 共通の処理 ──────────────────────────────────────────────────────────────

func TestRequestID_IsAssignedPerRequest(t *testing.T) {
	server := newServer(t, presentation.RouterOptions{})
	// 外から同じ番号を送り込んでも、こちらで振り直す
	spoofed := map[string]string{"X-Request-ID": "from-outside"}
	first := do(t, server, request{method: "GET", path: "/health", header: spoofed}).Header().Get("X-Request-ID")
	second := do(t, server, request{method: "GET", path: "/health", header: spoofed}).Header().Get("X-Request-ID")

	if len(first) != 16 || first == "from-outside" {
		t.Errorf("X-Request-ID = %q", first)
	}
	if first == second {
		t.Error("リクエストごとに違う番号になるはず")
	}
}

func TestCORS(t *testing.T) {
	const web = "https://sf-meter.example"
	server := newServer(t, presentation.RouterOptions{AllowedOrigins: []string{web}})

	t.Run("許したオリジンには、読んでよいと返す", func(t *testing.T) {
		rec := do(t, server, request{method: "GET", path: "/api/v1/techs", header: map[string]string{"Origin": web}})
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != web {
			t.Errorf("Access-Control-Allow-Origin = %q", got)
		}
		if got := rec.Header().Get("Access-Control-Expose-Headers"); !strings.Contains(got, "X-Total-Count") {
			t.Errorf("総件数のヘッダーを画面から読めない: %q", got)
		}
		if got := rec.Header().Get("Vary"); got != "Origin" {
			t.Errorf("Vary = %q", got)
		}
	})

	t.Run("許していないオリジンには、何も付けない", func(t *testing.T) {
		rec := do(t, server, request{method: "GET", path: "/api/v1/techs", header: map[string]string{"Origin": "https://evil.example"}})
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Errorf("Access-Control-Allow-Origin = %q", got)
		}
		if rec.Code != http.StatusOK {
			t.Errorf("ブラウザー以外からの呼び出しまで止めてはいけない: %d", rec.Code)
		}
	})

	t.Run("事前確認（preflight）に答える", func(t *testing.T) {
		rec := do(t, server, request{method: "OPTIONS", path: "/api/v1/techs", header: map[string]string{
			"Origin": web, "Access-Control-Request-Method": "POST", "Access-Control-Request-Headers": "authorization, content-type",
		}})
		if rec.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d", rec.Code)
		}
		if got := rec.Header().Get("Access-Control-Allow-Headers"); !strings.Contains(got, "Authorization") {
			t.Errorf("Access-Control-Allow-Headers = %q", got)
		}
	})

	t.Run("* なら、どこからでも読める", func(t *testing.T) {
		open := newServer(t, presentation.RouterOptions{AllowedOrigins: []string{"*"}})
		rec := do(t, open, request{method: "GET", path: "/api/v1/techs", header: map[string]string{"Origin": "https://anywhere.example"}})
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
			t.Errorf("Access-Control-Allow-Origin = %q", got)
		}
	})
}

// brokenUseCase は、何を呼んでも失敗する（または panic する）usecase
type brokenUseCase struct {
	presentation.TechUseCase
	panics bool
}

func (b brokenUseCase) List(context.Context, usecase.TechQuery) (usecase.TechList, error) {
	if b.panics {
		panic("想定していない状態")
	}
	return usecase.TechList{}, errors.New("pq: password authentication failed for user \"sfmeter\"")
}

func (b brokenUseCase) Get(context.Context, int) (domain.Tech, error) {
	return domain.Tech{}, errors.New("connection refused")
}

// 中で起きたエラーの中身（接続先やユーザー名など）を、外に出さないこと
func TestInternalError_DoesNotLeakDetails(t *testing.T) {
	server := presentation.NewRouter(presentation.NewTechHandler(brokenUseCase{}), presentation.RouterOptions{Logger: quiet})

	for _, path := range []string{"/api/v1/techs", "/api/v1/techs/1"} {
		rec := do(t, server, request{method: "GET", path: path})
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("%s: expected 500, got %d", path, rec.Code)
		}
		if body := rec.Body.String(); strings.Contains(body, "sfmeter") || strings.Contains(body, "refused") {
			t.Errorf("%s: 中のエラーが外に出ている: %s", path, body)
		}
		if code := decode[presentation.ErrorDTO](t, rec).Error.Code; code != "internal_error" {
			t.Errorf("%s: code = %q", path, code)
		}
	}
}

func TestPanic_IsRecovered(t *testing.T) {
	server := presentation.NewRouter(presentation.NewTechHandler(brokenUseCase{panics: true}), presentation.RouterOptions{Logger: quiet})

	rec := do(t, server, request{method: "GET", path: "/api/v1/techs"})
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
	// panic のあとも、次のリクエストに答えられること
	if rec = do(t, server, request{method: "GET", path: "/health"}); rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

// ─── 仕様書とのずれ ──────────────────────────────────────────────────────────

// 公開しているパスが、docs/openapi.yaml にすべて書かれていること。
// パスを足したのに仕様書を直し忘れる、を止める（以前、README と実装でパスが食い違っていたため）
func TestRoutes_AreDocumentedInOpenAPI(t *testing.T) {
	spec, err := os.ReadFile("../docs/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	// 仕様書を「パス → その下に書かれたメソッド」に分ける
	documented := map[string]map[string]bool{}
	var current string
	for _, line := range strings.Split(string(spec), "\n") {
		switch {
		case strings.HasPrefix(line, "  /") && strings.HasSuffix(line, ":"):
			current = strings.TrimSuffix(strings.TrimSpace(line), ":")
			documented[current] = map[string]bool{}
		case current != "" && strings.HasPrefix(line, "    ") && !strings.HasPrefix(line, "     "):
			documented[current][strings.ToUpper(strings.TrimSuffix(strings.TrimSpace(line), ":"))] = true
		case line != "" && !strings.HasPrefix(line, " "):
			current = "" // paths: の節を抜けた
		}
	}

	for _, route := range presentation.Routes() {
		method, path, _ := strings.Cut(route, " ")
		if !documented[path][method] {
			t.Errorf("%s が docs/openapi.yaml に書かれていない", route)
		}
	}
	routes := strings.Join(presentation.Routes(), "\n")
	for path, methods := range documented {
		for method := range methods {
			if method == "PARAMETERS" {
				continue
			}
			if !strings.Contains(routes, method+" "+path) {
				t.Errorf("docs/openapi.yaml にある %s %s が、実装に無い", method, path)
			}
		}
	}
}
