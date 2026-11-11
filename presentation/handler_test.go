package presentation_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nemdull/sf-meter-api/domain"
	"github.com/nemdull/sf-meter-api/presentation"
)

// ─── Mock UseCase ────────────────────────────────────────────────────────────

type mockTechUseCase struct {
	techs    []domain.Tech
	findTech *domain.Tech
	err      error
}

func (m *mockTechUseCase) GetAll() ([]domain.Tech, error) {
	return m.techs, m.err
}

func (m *mockTechUseCase) GetByID(id int) (*domain.Tech, error) {
	return m.findTech, m.err
}

// ─── GET /techs ───────────────────────────────────────────────────────────────

func TestGetTechs_OK(t *testing.T) {
	uc := &mockTechUseCase{
		techs: []domain.Tech{
			{ID: 1, Name: "ドミネーター", Score: 35},
			{ID: 2, Name: "ニューラルリンク", Score: 82},
		},
	}
	h := presentation.NewTechHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/techs", nil)
	rec := httptest.NewRecorder()
	h.GetTechs(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var resp []domain.Tech
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp) != 2 {
		t.Errorf("expected 2 techs, got %d", len(resp))
	}
}

func TestGetTechs_Empty(t *testing.T) {
	uc := &mockTechUseCase{techs: []domain.Tech{}}
	h := presentation.NewTechHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/techs", nil)
	rec := httptest.NewRecorder()
	h.GetTechs(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

// ─── GET /techs/{id} ─────────────────────────────────────────────────────────

func TestGetTechByID_Found(t *testing.T) {
	uc := &mockTechUseCase{
		findTech: &domain.Tech{ID: 1, Name: "ドミネーター", Score: 35},
	}
	h := presentation.NewTechHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/techs/1", nil)
	req.SetPathValue("id", "1")
	rec := httptest.NewRecorder()
	h.GetTechByID(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var resp domain.Tech
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Name != "ドミネーター" {
		t.Errorf("unexpected name: %s", resp.Name)
	}
}

func TestGetTechByID_NotFound(t *testing.T) {
	uc := &mockTechUseCase{findTech: nil}
	h := presentation.NewTechHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/techs/999", nil)
	req.SetPathValue("id", "999")
	rec := httptest.NewRecorder()
	h.GetTechByID(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestGetTechByID_InvalidID(t *testing.T) {
	uc := &mockTechUseCase{}
	h := presentation.NewTechHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/techs/abc", nil)
	req.SetPathValue("id", "abc")
	rec := httptest.NewRecorder()
	h.GetTechByID(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}
