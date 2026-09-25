package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/internal/service"
	"bibliothek/repository"
	"bibliothek/sse"
)

// Gemischte Auflagen an der Theke (docs/OFFEN.md 4.18, Stufe 5): Bekommt ein Kind eine Auflage,
// während Kinder seiner Klasse eine andere desselben Buchs haben, trägt die Antwort des Scans
// den Hinweis. Geprüft über den Live-Pfad — POST /api/action mit dem echten Omnibox- und
// Ausleih-Dienst —, denn nur so ist belegt, dass der Hinweis von der Buchung bis in die JSON-
// Antwort reist. Die Regeln der Zählung prüft repository/auflagen_klasse_pg_test.go.
func TestTheke_AuflagenHinweisBeiGemischterKlasse(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `TRUNCATE werke CASCADE`); err != nil {
		t.Fatal(err)
	}

	var mitarbeiter string
	if err := pool.QueryRow(ctx, `INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Theke', 'Auflage', 'theke-auflage@schule.invalid', 'mitarbeiter', true) RETURNING id`).Scan(&mitarbeiter); err != nil {
		t.Fatalf("Mitarbeiter anlegen: %v", err)
	}
	kind := func(barcode, klasse string) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
			VALUES ($1, 'Kind', $1, $2, 2031) RETURNING id`, barcode, klasse).Scan(&id); err != nil {
			t.Fatalf("Schüler %s anlegen: %v", barcode, err)
		}
		return id
	}
	titel := func(auflage string, jahr int) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `INSERT INTO buecher_titel (titel, auflage, erscheinungsjahr, ist_lernmittel)
			VALUES ('Mathe 7', $1, $2, true) RETURNING id`, auflage, jahr).Scan(&id); err != nil {
			t.Fatalf("Titel anlegen: %v", err)
		}
		return id
	}
	alt, neu := titel("3. Aufl.", 2019), titel("4. Aufl.", 2023)
	if _, err := repository.FasseAuflagenZusammen(ctx, pool, alt, neu); err != nil {
		t.Fatal(err)
	}
	for i, t2 := range []string{alt, alt, alt, neu, neu} {
		exemplar(t, pool, t2, fmt.Sprintf("B-518T-%d", i), true, "")
	}

	studentRepo := repository.NewStudentRepository(pool)
	bookRepo := repository.NewBookRepository(pool)
	loanRepo := repository.NewLoanRepository(pool)
	auditRepo := repository.NewAuditRepository(pool)
	loanSvc := service.NewLoanService(pool, studentRepo, bookRepo, loanRepo, auditRepo)
	deviceSvc := service.NewDeviceService(pool, studentRepo, loanRepo, auditRepo)
	omnibox := service.NewOmniboxService(pool, studentRepo, bookRepo, repository.NewUserRepository(pool), loanRepo, loanSvc, deviceSvc)
	srv := &Server{DB: &db.Database{Pool: pool}, Broker: sse.NewBroker()}

	scan := func(barcode, leser string) ActionResponse {
		t.Helper()
		req := httptest.NewRequest("POST", "/api/action",
			strings.NewReader(fmt.Sprintf(`{"query":%q,"active_leser_id":%q}`, barcode, leser)))
		req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey,
			&auth.Claims{Rolle: auth.Role("mitarbeiter"), UserID: mitarbeiter}))
		rec := httptest.NewRecorder()
		srv.ActionHandler(omnibox)(rec, req)
		var antwort ActionResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil || rec.Code != 200 {
			t.Fatalf("Scan %s: %d %s", barcode, rec.Code, rec.Body.String())
		}
		return antwort
	}

	// Zwei Kinder der 7b bekommen die 3. Auflage — kein Hinweis, die Klasse ist einheitlich.
	if a := scan("B-518T-0", kind("S-518T-1", "7b")); a.Type != "ausleihe" || a.AuflagenHinweis != nil {
		t.Fatalf("erste Ausleihe: Typ %q, Hinweis %+v — erwartet Ausleihe ohne Hinweis", a.Type, a.AuflagenHinweis)
	}
	if a := scan("B-518T-1", kind("S-518T-2", "7b")); a.AuflagenHinweis != nil {
		t.Errorf("zweite Ausleihe derselben Auflage: Hinweis %+v — erwartet keiner", a.AuflagenHinweis)
	}

	// Das dritte Kind bekommt die 4. Auflage: Hinweis mit Klasse und den zwei anderen.
	a := scan("B-518T-3", kind("S-518T-3", "7b"))
	if a.Type != "ausleihe" {
		t.Fatalf("Typ %q — die Ausleihe muss gebucht sein, der Hinweis ändert daran nichts", a.Type)
	}
	h := a.AuflagenHinweis
	if h == nil || h.Klasse != "07B" || h.Auflage != "4. Aufl." || len(h.Andere) != 1 ||
		h.Andere[0].Auflage != "3. Aufl." || h.Andere[0].Kinder != 2 {
		t.Errorf("Hinweis %+v — erwartet 07B, 4. Aufl., andere: 3. Aufl. bei 2 Kindern", h)
	}

	// Ein Kind einer anderen Klasse bekommt die 4. Auflage: dort gibt es keine Mischung.
	if a := scan("B-518T-4", kind("S-518T-4", "7c")); a.AuflagenHinweis != nil {
		t.Errorf("7c: Hinweis %+v — erwartet keiner", a.AuflagenHinweis)
	}
}
