// Package registry wires all layers together.
// This is the only place that knows about every concrete type.
// All other packages depend only on interfaces.
package registry

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/nemdull/sf-meter-api/infrastructure"
	"github.com/nemdull/sf-meter-api/presentation"
	"github.com/nemdull/sf-meter-api/usecase"
)

// 開発中の画面（Vite の開発サーバー）から呼べるように、指定が無いときはこの 2 つを許す
var defaultOrigins = []string{"http://localhost:5173", "http://127.0.0.1:5173"}

// Config は、環境変数から読む設定。
type Config struct {
	// DatabaseURL は PostgreSQL の接続文字列。空ならメモリ実装を使う
	DatabaseURL string
	// AdminToken は、追加・更新・削除に要るトークン。空なら書き込みを閉じる
	AdminToken string
	// AllowedOrigins は、ブラウザーから API を呼んでよいオリジン
	AllowedOrigins []string
}

// ConfigFromEnv は、環境変数から設定を組み立てる。
//
//	DATABASE_URL          未設定ならメモリ実装
//	ADMIN_TOKEN           未設定なら書き込みを閉じる
//	CORS_ALLOWED_ORIGINS  カンマ区切り。未設定なら Vite の開発サーバーだけ。"*" でどこからでも
func ConfigFromEnv() Config {
	origins := defaultOrigins
	if raw := strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS")); raw != "" {
		origins = nil
		for _, origin := range strings.Split(raw, ",") {
			if origin = strings.TrimSpace(origin); origin != "" {
				origins = append(origins, strings.TrimSuffix(origin, "/"))
			}
		}
	}
	return Config{
		DatabaseURL:    os.Getenv("DATABASE_URL"),
		AdminToken:     os.Getenv("ADMIN_TOKEN"),
		AllowedOrigins: origins,
	}
}

// App は、組み立て終わったアプリケーション。
type App struct {
	// Handler は、すべてのパスを受け付ける HTTP ハンドラー
	Handler http.Handler
	// Store は、使っている保存先の名前（memory / postgres）
	Store string
	// Close は、保存先との接続を閉じる。サーバーを止めるときに呼ぶ
	Close func() error
}

// New assembles the full dependency chain.
//
//	infrastructure.PostgresTechRepo (or MemoryTechRepo)
//	                 ← usecase.TechService
//	                 ← presentation.TechHandler
//	                 ← presentation.NewRouter
func New(ctx context.Context, cfg Config, logger *slog.Logger) (*App, error) {
	var (
		repo   usecase.TechRepository
		pinger presentation.Pinger
		store  = "memory"
		closer = func() error { return nil }
	)
	if cfg.DatabaseURL != "" {
		pgRepo, err := infrastructure.NewPostgresTechRepo(ctx, cfg.DatabaseURL)
		if err != nil {
			return nil, err
		}
		repo, pinger, store, closer = pgRepo, pgRepo, "postgres", pgRepo.Close
	} else {
		repo = infrastructure.NewMemoryTechRepo(infrastructure.SeedTechs()...)
	}

	svc := usecase.NewTechService(repo)
	handler := presentation.NewRouter(presentation.NewTechHandler(svc), presentation.RouterOptions{
		AdminToken:     cfg.AdminToken,
		AllowedOrigins: cfg.AllowedOrigins,
		Store:          store,
		Pinger:         pinger,
		Logger:         logger,
	})
	return &App{Handler: handler, Store: store, Close: closer}, nil
}
