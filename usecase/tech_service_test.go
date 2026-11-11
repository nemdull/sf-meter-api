package usecase_test

import (
	"errors"
	"testing"

	"github.com/nemdull/sf-meter-api/domain"
	"github.com/nemdull/sf-meter-api/usecase"
)

// ─── Mock ───────────────────────────────────────────────────────────────────

type mockTechRepo struct {
	techs  []domain.Tech
	forceErr error
}

func (m *mockTechRepo) FindAll() ([]domain.Tech, error) {
	if m.forceErr != nil {
		return nil, m.forceErr
	}
	return m.techs, nil
}

func (m *mockTechRepo) FindByID(id int) (*domain.Tech, error) {
	if m.forceErr != nil {
		return nil, m.forceErr
	}
	for _, t := range m.techs {
		if t.ID == id {
			cp := t
			return &cp, nil
		}
	}
	return nil, nil
}

// ─── GetAll ──────────────────────────────────────────────────────────────────

func TestTechService_GetAll_ReturnsTechs(t *testing.T) {
	repo := &mockTechRepo{
		techs: []domain.Tech{
			{ID: 1, Name: "ドミネーター", Description: "支配射撃体系", Score: 35},
			{ID: 2, Name: "ニューラルリンク", Description: "脳直結インターフェース", Score: 82},
		},
	}
	svc := usecase.NewTechService(repo)

	techs, err := svc.GetAll()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(techs) != 2 {
		t.Errorf("expected 2 techs, got %d", len(techs))
	}
	if techs[0].Name != "ドミネーター" {
		t.Errorf("unexpected first tech: %s", techs[0].Name)
	}
}

func TestTechService_GetAll_Empty(t *testing.T) {
	repo := &mockTechRepo{techs: []domain.Tech{}}
	svc := usecase.NewTechService(repo)

	techs, err := svc.GetAll()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(techs) != 0 {
		t.Errorf("expected empty slice, got %d items", len(techs))
	}
}

func TestTechService_GetAll_RepoError(t *testing.T) {
	repo := &mockTechRepo{forceErr: errors.New("db error")}
	svc := usecase.NewTechService(repo)

	_, err := svc.GetAll()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ─── GetByID ─────────────────────────────────────────────────────────────────

func TestTechService_GetByID_Found(t *testing.T) {
	repo := &mockTechRepo{
		techs: []domain.Tech{
			{ID: 1, Name: "ドミネーター", Description: "支配射撃体系", Score: 35},
		},
	}
	svc := usecase.NewTechService(repo)

	tech, err := svc.GetByID(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tech == nil {
		t.Fatal("expected tech, got nil")
	}
	if tech.Name != "ドミネーター" {
		t.Errorf("unexpected name: %s", tech.Name)
	}
}

func TestTechService_GetByID_NotFound(t *testing.T) {
	repo := &mockTechRepo{techs: []domain.Tech{}}
	svc := usecase.NewTechService(repo)

	tech, err := svc.GetByID(999)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tech != nil {
		t.Errorf("expected nil for non-existent tech, got %+v", tech)
	}
}

func TestTechService_GetByID_RepoError(t *testing.T) {
	repo := &mockTechRepo{forceErr: errors.New("db error")}
	svc := usecase.NewTechService(repo)

	_, err := svc.GetByID(1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ─── Score Validation (ドメインルール) ───────────────────────────────────────

func TestTechService_ScoreRange(t *testing.T) {
	tests := []struct {
		name    string
		score   int
		wantErr bool
	}{
		{"valid low boundary", 0, false},
		{"valid high boundary", 100, false},
		{"valid mid", 50, false},
		{"below range", -1, true},
		{"above range", 101, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tech := domain.Tech{Name: "test", Score: tc.score}
			err := tech.Validate()
			if tc.wantErr && err == nil {
				t.Errorf("expected error for score %d, got nil", tc.score)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error for score %d: %v", tc.score, err)
			}
		})
	}
}
