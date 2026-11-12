package infrastructure_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/nemdull/sf-meter-api/infrastructure"
	"github.com/nemdull/sf-meter-api/infrastructure/repotest"
	"github.com/nemdull/sf-meter-api/usecase"
)

// PostgreSQL 実装にも、メモリ実装と同じテストを当てる。
// 本物の DB が要るので、TEST_DATABASE_URL があるときだけ走る（CI では PostgreSQL を立てて走らせている）。
//
//	docker compose up -d db
//	TEST_DATABASE_URL="postgres://sfmeter:sfmeter@localhost:5432/sfmeter_db?sslmode=disable" go test ./infrastructure/...
//
// テーブルを空にしてから始めるので、消えて困るデータのある DB には向けないこと。
func TestPostgresTechRepo_Contract(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL が無いので、PostgreSQL の結合テストは飛ばす")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	repo, err := infrastructure.NewPostgresTechRepo(ctx, dsn)
	if err != nil {
		t.Fatalf("PostgreSQL につながらない: %v", err)
	}
	t.Cleanup(func() { _ = repo.Close() })

	repotest.Run(t, func(t *testing.T) usecase.TechRepository {
		if err := repo.Reset(ctx, repotest.Fixture...); err != nil {
			t.Fatalf("初期データを入れ直せない: %v", err)
		}
		return repo
	})
}
