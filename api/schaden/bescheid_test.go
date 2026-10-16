package schaden

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/repository"
)

// mockBescheidAbfragen implementiert das repository.BescheidAbfragen Interface für Tests.
type mockBescheidAbfragen struct {
	ladeListeFunc func(ctx context.Context, offen bool) ([]repository.Bescheid, error)
}

func (m *mockBescheidAbfragen) LegeAn(ctx context.Context, e repository.BescheidEingabe) (*repository.Bescheid, error) {
	return nil, nil
}
func (m *mockBescheidAbfragen) LadeListe(ctx context.Context, nurOffen bool) ([]repository.Bescheid, error) {
	if m.ladeListeFunc != nil {
		return m.ladeListeFunc(ctx, nurOffen)
	}
	return nil, nil
}
func (m *mockBescheidAbfragen) LadeZuSchueler(ctx context.Context, schuelerID string) ([]repository.Bescheid, error) {
	return nil, nil
}
func (m *mockBescheidAbfragen) Lade(ctx context.Context, id string) (*repository.Bescheid, error) {
	return nil, nil
}
func (m *mockBescheidAbfragen) LadeEmpfaengerSnapshot(ctx context.Context, id string) (map[string]string, error) {
	return nil, nil
}
func (m *mockBescheidAbfragen) LadeAbsenderSnapshot(ctx context.Context, id string) (map[string]string, error) {
	return nil, nil
}
func (m *mockBescheidAbfragen) LadePositionen(ctx context.Context, id string) ([]repository.BescheidBriefPosition, error) {
	return nil, nil
}
func (m *mockBescheidAbfragen) VermerkeDruck(ctx context.Context, id string) error { return nil }
func (m *mockBescheidAbfragen) Uebergebe(ctx context.Context, id string) error     { return nil }
func (m *mockBescheidAbfragen) LadeEmpfaenger(ctx context.Context, schuelerID string) (repository.BescheidEmpfaengerDaten, error) {
	return repository.BescheidEmpfaengerDaten{}, nil
}
func (m *mockBescheidAbfragen) LadeOffeneForderungen(ctx context.Context, schuelerID string) ([]repository.OffeneForderung, error) {
	return nil, nil
}
func (m *mockBescheidAbfragen) LadeForderungenOhneBescheid(ctx context.Context) ([]repository.ForderungOhneBescheid, error) {
	return nil, nil
}
func (m *mockBescheidAbfragen) LadeUeberfaelligeAusleihen(ctx context.Context, schuelerID string) ([]repository.UeberfaelligeAusleihe, error) {
	return nil, nil
}
func (m *mockBescheidAbfragen) LadeGroessenFuerExemplar(ctx context.Context, exemplarID string) (repository.ErsatzwertGroessen, error) {
	return repository.ErsatzwertGroessen{}, nil
}
func (m *mockBescheidAbfragen) LadeGroessenFuerTitel(ctx context.Context, titelID string) (map[string]repository.ErsatzwertGroessen, error) {
	return nil, nil
}

func TestBescheide_Success(t *testing.T) {
	mockRepo := &mockBescheidAbfragen{
		ladeListeFunc: func(ctx context.Context, offen bool) ([]repository.Bescheid, error) {
			if offen {
				return []repository.Bescheid{{ID: "1", Status: "OFFEN"}}, nil
			}
			return []repository.Bescheid{{ID: "1", Status: "OFFEN"}, {ID: "2", Status: "BEZAHLT"}}, nil
		},
	}

	tueren := &Tueren{}
	handler := tueren.Bescheide(mockRepo)

	// Test mit status=offen
	req := httptest.NewRequest(http.MethodGet, "/bescheide?status=offen", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler gab falschen Statuscode zurück: got %v want %v", status, http.StatusOK)
	}

	expectedBody := `[{"id":"1","mittel":"","referenznummer":"","kassenjahr":0,"laufende_nr":0,"brief_datum":"0001-01-01T00:00:00Z","frist_bis":"0001-01-01T00:00:00Z","gesamtbetrag":0,"status":"OFFEN","frist_abgelaufen":false,"rueckgabe_nach_uebergabe":false,"anzahl_positionen":0}]`
	if strings.TrimSpace(rr.Body.String()) != expectedBody {
		t.Errorf("Handler gab falschen Body zurück:\n  got:  %v\n  want: %v", rr.Body.String(), expectedBody)
	}
}

func TestBescheide_Error(t *testing.T) {
	mockRepo := &mockBescheidAbfragen{
		ladeListeFunc: func(ctx context.Context, offen bool) ([]repository.Bescheid, error) {
			return nil, errors.New("datenbankfehler")
		},
	}

	tueren := &Tueren{}
	handler := tueren.Bescheide(mockRepo)

	req := httptest.NewRequest(http.MethodGet, "/bescheide", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("Handler gab falschen Statuscode zurück: got %v want %v", status, http.StatusInternalServerError)
	}
}
