package api_test

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

	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/routertest"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// Über den ganzen Router: Ein Serverfehler steht einmal im Log, als Zeile zum Status mit der Route
// und der Ursache aus der Tür. Der Test der Kette (middleware_statuszeile_test.go) baut sie selbst
// und sähe nicht, ob ihre Glieder im Router hängen.

const lesefehler = "verbindung zur datenbank verloren"

// poolMitLesefehler antwortet auf das Lesen des Ausweis-Designs mit einem Fehler der Datenbank,
// sobald er scharf ist. Sitzung und Rechte laufen am echten Pool.
type poolMitLesefehler struct {
	db.Pool
	scharf *bool
}

func (p poolMitLesefehler) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if *p.scharf && strings.Contains(sql, "FROM system_einstellungen") && len(args) > 0 && args[0] == repository.AusweisLayoutSchluessel {
		return zeileMitFehler{}
	}
	return p.Pool.QueryRow(ctx, sql, args...)
}

type zeileMitFehler struct{}

func (zeileMitFehler) Scan(...any) error { return errors.New(lesefehler) }

func TestStatuszeile_UeberDenRouter(t *testing.T) {
	pool := pgtest.Pool(t)
	scharf := true
	const konto = "statuszeile-router@example.org"
	_, sitzung, router := routertest.MitSitzungUeber(t, pool, poolMitLesefehler{Pool: pool, scharf: &scharf},
		konto, "Stella", "Status")
	t.Cleanup(func() { pgtest.RaeumeAuf(t, pool, `DELETE FROM benutzer WHERE email = $1`, konto) })

	var protokoll bytes.Buffer
	vorher := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&protokoll, nil)))
	t.Cleanup(func() { slog.SetDefault(vorher) })

	anfrage := httptest.NewRequest(http.MethodGet, "/api/ausweis-layout", nil)
	anfrage.AddCookie(&http.Cookie{Name: "session_token", Value: sitzung})
	antwort := httptest.NewRecorder()
	router.ServeHTTP(antwort, anfrage)
	if antwort.Code != http.StatusInternalServerError {
		t.Fatalf("Status %d, erwartet 500: %.200s", antwort.Code, antwort.Body.String())
	}

	var zeilen []map[string]any
	for _, roh := range strings.Split(strings.TrimSpace(protokoll.String()), "\n") {
		if !strings.Contains(roh, lesefehler) {
			continue
		}
		var zeile map[string]any
		if err := json.Unmarshal([]byte(roh), &zeile); err != nil {
			t.Fatalf("Logzeile %q: %v", roh, err)
		}
		zeilen = append(zeilen, zeile)
	}
	if len(zeilen) != 1 {
		t.Fatalf("%d Zeilen mit der Ursache, erwartet eine:\n%s", len(zeilen), protokoll.String())
	}
	will := map[string]any{"level": "ERROR", "msg": "Anfrage mit Serverfehler beendet", "route": "GET /api/ausweis-layout"}
	for feld, wert := range will {
		if zeilen[0][feld] != wert {
			t.Errorf("Feld %s = %v, erwartet %v", feld, zeilen[0][feld], wert)
		}
	}
}
