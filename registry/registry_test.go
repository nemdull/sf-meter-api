package registry_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/nemdull/sf-meter-api/registry"
)

func TestConfigFromEnv_Defaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("ADMIN_TOKEN", "")
	t.Setenv("CORS_ALLOWED_ORIGINS", "")

	cfg := registry.ConfigFromEnv()
	if cfg.DatabaseURL != "" || cfg.AdminToken != "" {
		t.Errorf("unexpected config: %+v", cfg)
	}
	if !slices.Equal(cfg.AllowedOrigins, []string{"http://localhost:5173", "http://127.0.0.1:5173"}) {
		t.Errorf("指定が無ければ開発サーバーだけを許すはずが: %v", cfg.AllowedOrigins)
	}
}

func TestConfigFromEnv_Origins(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", " https://a.example/ , ,https://b.example")

	got := registry.ConfigFromEnv().AllowedOrigins
	if !slices.Equal(got, []string{"https://a.example", "https://b.example"}) {
		t.Errorf("空白・空の項目・末尾の / を取り除くはずが: %v", got)
	}
}

// DATABASE_URL が無ければ、メモリ実装で組み上がり、そのまま応答できること
func TestNew_MemoryStore(t *testing.T) {
	app, err := registry.New(context.Background(), registry.Config{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = app.Close() })

	if app.Store != "memory" {
		t.Errorf("Store = %q", app.Store)
	}
	rec := httptest.NewRecorder()
	app.Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/techs", nil))
	if rec.Code != http.StatusOK || rec.Header().Get("X-Total-Count") != "15" {
		t.Errorf("status = %d、X-Total-Count = %q", rec.Code, rec.Header().Get("X-Total-Count"))
	}
}

// つながらない DATABASE_URL のときは、黙ってメモリ実装に切り替えたりせず、起動を失敗させること
func TestNew_BadDatabaseURL(t *testing.T) {
	cfg := registry.Config{DatabaseURL: "postgres://nobody@127.0.0.1:1/none?sslmode=disable&connect_timeout=1"}
	if _, err := registry.New(context.Background(), cfg, slog.Default()); err == nil {
		t.Fatal("expected error, got nil")
	}
}
