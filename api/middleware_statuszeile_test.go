package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/apierrors"
)

// Ein Serverfehler steht einmal im Log: als Zeile zum Status mit Route, Pfad und Ursache, auch
// hinter der Kompression, die den Writer einhüllt. Hat der Browser abgebrochen, steht sie als WARN,
// sonst löste jeder Seitenwechsel mitten im Laden die tägliche Alarm-Mail aus; eine abgelaufene
// Frist bleibt ein Fehler, der Server war zu langsam. Ein 4xx ist eine abgelehnte Eingabe (INFO).
func TestStatuszeile_EinFehlerEineZeile(t *testing.T) {
	antworte := func(status int, ursache error) http.HandlerFunc {
		return func(w http.ResponseWriter, _ *http.Request) { apierrors.AntworteFehler(w, status, ursache) }
	}
	ursache := errors.New("verbindung zur datenbank verloren")
	faelle := []struct {
		name string
		// abbruch ist der Grund, mit dem der Kontext der Anfrage schon geendet hat; nil heißt, er läuft.
		abbruch    error
		tuer       http.Handler
		willStufe  string
		willFelder map[string]any
	}{
		{"Serverfehler", nil, antworte(http.StatusInternalServerError, ursache), "ERROR",
			map[string]any{"msg": "Anfrage mit Serverfehler beendet", "route": "GET /api/probe/{id}", "pfad": "/api/probe/42", "fehler": ursache.Error()}},
		{"Fehler hinter MitFehlerantwort", nil, apierrors.MitFehlerantwort(func(http.ResponseWriter, *http.Request) error {
			return apierrors.New(http.StatusServiceUnavailable, "Der Mailserver antwortet nicht", errors.New("smtp: verbindung abgelehnt"))
		}), "ERROR", map[string]any{"route": "GET /api/probe/{id}", "fehler": "smtp: verbindung abgelehnt"}},
		{"Browser hat abgebrochen", context.Canceled, antworte(http.StatusInternalServerError, ursache), "WARN",
			map[string]any{"msg": "Anfrage vom Browser abgebrochen", "fehler": ursache.Error()}},
		{"Frist abgelaufen", context.DeadlineExceeded, antworte(http.StatusServiceUnavailable, ursache), "ERROR",
			map[string]any{"msg": "Anfrage mit Serverfehler beendet"}},
		{"abgelehnte Eingabe", nil, antworte(http.StatusBadRequest, errors.New("barcode unbekannt")), "INFO",
			map[string]any{"msg": "Fehlerantwort", "fehler": "barcode unbekannt"}},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			var protokoll bytes.Buffer
			vorher := slog.Default()
			slog.SetDefault(slog.New(slog.NewJSONHandler(&protokoll, nil)))
			t.Cleanup(func() { slog.SetDefault(vorher) })

			mux := http.NewServeMux()
			mux.Handle("GET /api/probe/{id}", f.tuer)
			anfrage := httptest.NewRequestWithContext(kontextMit(t, f.abbruch), http.MethodGet, "/api/probe/42", nil)
			anfrage.Header.Set("Accept-Encoding", "gzip")
			logMiddleware(kompressionMiddleware(merkeRoute(mux))).ServeHTTP(httptest.NewRecorder(), anfrage)

			var zeile map[string]any
			if err := json.Unmarshal(protokoll.Bytes(), &zeile); err != nil || strings.Count(protokoll.String(), "\n") != 1 || zeile["level"] != f.willStufe {
				t.Fatalf("Log:\n%s\nerwartet genau eine Zeile der Stufe %s", protokoll.String(), f.willStufe)
			}
			for feld, will := range f.willFelder {
				if ist := zeile[feld]; ist != will {
					t.Errorf("Feld %s = %v, erwartet %v", feld, ist, will)
				}
			}
		})
	}
}

// kontextMit liefert einen Kontext, der mit abbruch geendet hat; ohne abbruch läuft er.
func kontextMit(t *testing.T, abbruch error) context.Context {
	t.Helper()
	switch abbruch {
	case context.Canceled:
		ctx, abbrechen := context.WithCancel(context.Background())
		abbrechen()
		return ctx
	case context.DeadlineExceeded:
		ctx, aus := context.WithTimeout(context.Background(), 0)
		t.Cleanup(aus)
		<-ctx.Done()
		return ctx
	}
	return context.Background()
}
