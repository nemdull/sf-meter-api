package infrastructure_test

import (
	"context"
	"sync"
	"testing"

	"github.com/nemdull/sf-meter-api/domain"
	"github.com/nemdull/sf-meter-api/infrastructure"
	"github.com/nemdull/sf-meter-api/infrastructure/repotest"
	"github.com/nemdull/sf-meter-api/usecase"
)

func TestMemoryTechRepo_Contract(t *testing.T) {
	repotest.Run(t, func(t *testing.T) usecase.TechRepository {
		return infrastructure.NewMemoryTechRepo(repotest.Fixture...)
	})
}

// 同時に読み書きしても壊れないこと。go test -race で、排他の抜けを見つける
func TestMemoryTechRepo_ConcurrentAccess(t *testing.T) {
	repo := infrastructure.NewMemoryTechRepo(repotest.Fixture...)
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := range 50 {
		wg.Add(2)
		go func() {
			defer wg.Done()
			if _, err := repo.Create(ctx, domain.Tech{Name: "同時に追加", Score: i}); err != nil {
				t.Error(err)
			}
		}()
		go func() {
			defer wg.Done()
			if _, _, err := repo.Find(ctx, usecase.AllTechs()); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()

	_, total, err := repo.Find(ctx, usecase.AllTechs())
	if err != nil {
		t.Fatal(err)
	}
	if want := len(repotest.Fixture) + 50; total != want {
		t.Errorf("total = %d、期待は %d", total, want)
	}
}
