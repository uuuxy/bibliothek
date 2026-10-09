package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Die Sperre, die das Programm den Ehemaligen setzt, hält beim Schulbuch auch die Verlängerung
// nicht an — entschieden am 28.09.2026: „Der Satz der Schule gilt auch für die Ehemaligen, und
// auch bei der Verlängerung." Geprüft an den vier Stellen, die die Frist einer offenen Ausleihe
// umschreiben (die einzigen UPDATE auf rueckgabe_frist): Einzelverlängerung, Frist von Hand,
// Klassenverlängerung, LMF-Plan. Weiter angehalten werden die Sperre von Hand, der
// anonymisierte Datensatz, der Papierkorb und beim Buch der Bücherei auch die Sperre der
// Ehemaligen — wie an der Theke (TestTheke_EhemaligeSperreNichtAmSchulbuch).

// sperrFristLage: vier gesperrte Kinder der 10R1, jedes mit einem offenen Schulbuch, der
// Ehemalige zusätzlich mit einem Buch der Bücherei. Das anonymisierte Kind behält hier seine
// Klasse (die Anonymisierung leert sie oder setzt „ABG"), damit die Klassenwege es überhaupt
// erreichen und der Test die Bedingung prüft, nicht die Klasse.
type sperrFristLage struct {
	pool      *pgxpool.Pool
	srv       *Server
	alteFrist time.Time

	ehemaligSchulbuch, ehemaligBuecherei, vonHandSchulbuch, anonymSchulbuch, papierkorbSchulbuch string
}

// sperrFristFall: eine Ausleihe und ob ihre Frist sich bewegen darf.
type sperrFristFall struct {
	was  string
	id   string
	geht bool
}

func sperrFristLageAnlegen(t *testing.T) *sperrFristLage {
	t.Helper()
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	l := &sperrFristLage{
		pool:      pool,
		srv:       &Server{DB: &db.Database{Pool: pool}},
		alteFrist: fristEnde(2027, time.July, 31),
	}
	l.srv.Uhr = func() time.Time { return time.Date(2026, time.September, 29, 10, 0, 0, 0, schulzeit.Zone()) }

	sperre := func(id, was, sql string) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, id); err != nil {
			t.Fatalf("%s sperren: %v", was, err)
		}
	}
	ehemalig := seedSchueler(t, pool, "S-VSP-1", "Ehemalig", "10R1")
	sperre(ehemalig, "Ehemaligen", `UPDATE leser SET ist_abgaenger = true, ist_gesperrt = true,
		block_reason = 'Automatisierte Abgänger-Sperre (Schuljahreswechsel)' WHERE id = $1`)
	vonHand := seedSchueler(t, pool, "S-VSP-2", "VonHand", "10R1")
	sperre(vonHand, "von Hand", `UPDATE leser SET is_manually_blocked = true, block_reason = 'Buchverlust' WHERE id = $1`)
	anonym := seedSchueler(t, pool, "S-VSP-3", "Anonym", "10R1")
	sperre(anonym, "anonymisiert", `UPDATE leser SET ist_abgaenger = true, ist_gesperrt = true,
		block_reason = 'Abgänger anonymisiert', anonymized_at = NOW() WHERE id = $1`)
	papierkorb := seedSchueler(t, pool, "S-VSP-4", "Papierkorb", "10R1")
	sperre(papierkorb, "Papierkorb", `UPDATE leser SET deleted_at = NOW(), ist_gesperrt = true,
		block_reason = 'Systematisch gelöscht' WHERE id = $1`)

	l.ehemaligSchulbuch = seedAusleihe(t, pool, ehemalig, "LMF Mathe 10 Ehemalig", l.alteFrist)
	l.ehemaligBuecherei = seedAusleihe(t, pool, ehemalig, "Roman Ehemalig", l.alteFrist)
	l.vonHandSchulbuch = seedAusleihe(t, pool, vonHand, "LMF Mathe 10 VonHand", l.alteFrist)
	l.anonymSchulbuch = seedAusleihe(t, pool, anonym, "LMF Mathe 10 Anonym", l.alteFrist)
	l.papierkorbSchulbuch = seedAusleihe(t, pool, papierkorb, "LMF Mathe 10 Papierkorb", l.alteFrist)
	return l
}

// faelle: dieselben fünf Ausleihen für jeden Weg. Nur das Schulbuch des Ehemaligen geht.
func (l *sperrFristLage) faelle() []sperrFristFall {
	return []sperrFristFall{
		{was: "Schulbuch des Ehemaligen", id: l.ehemaligSchulbuch, geht: true},
		{was: "Buch der Bücherei des Ehemaligen", id: l.ehemaligBuecherei},
		{was: "Schulbuch mit Sperre von Hand", id: l.vonHandSchulbuch},
		{was: "Schulbuch des anonymisierten Datensatzes", id: l.anonymSchulbuch},
		{was: "Schulbuch im Papierkorb", id: l.papierkorbSchulbuch},
	}
}

// pruefeFrist: Bewegt hat sich genau, was gehen darf.
func (l *sperrFristLage) pruefeFrist(t *testing.T, f sperrFristFall) {
	t.Helper()
	bewegt := !fristVon(t, l.pool, f.id).Equal(l.alteFrist)
	if bewegt != f.geht {
		t.Errorf("%s: Frist bewegt=%v, erwartet %v", f.was, bewegt, f.geht)
	}
}

func TestVerlaengerung_EhemaligeSperreNichtAmSchulbuch_Einzeln(t *testing.T) {
	l := sperrFristLageAnlegen(t)
	mux := http.NewServeMux()
	mux.Handle("POST /api/ausleihen/{ausleihe_id}/verlaengern", l.srv.ExtendLoanHandler(&mockSystemSettingsRepo{
		settings: &repository.SystemEinstellungen{FristBuchTage: 28},
	}))
	for _, f := range l.faelle() {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/ausleihen/"+f.id+"/verlaengern", nil))
		soll := http.StatusForbidden
		if f.geht {
			soll = http.StatusOK
		}
		if rec.Code != soll {
			t.Errorf("%s: Status %d, erwartet %d: %s", f.was, rec.Code, soll, rec.Body.String())
		}
		l.pruefeFrist(t, f)
	}
}

func TestVerlaengerung_EhemaligeSperreNichtAmSchulbuch_FristVonHand(t *testing.T) {
	l := sperrFristLageAnlegen(t)
	var adminID string
	if err := l.pool.QueryRow(context.Background(), `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Frist', 'Admin', 'vsp-admin@test.invalid', 'admin', true)
		ON CONFLICT (email) DO UPDATE SET vorname = EXCLUDED.vorname
		RETURNING id`).Scan(&adminID); err != nil {
		t.Fatalf("Test-Admin: %v", err)
	}
	mux := http.NewServeMux()
	mux.Handle("PATCH /api/admin/ausleihen/{id}/faelligkeit", l.srv.OverrideDueDateHandler(repository.NewAuditRepository(l.pool)))
	// Ein Datum in der Zukunft — nur dann prüft der Weg die Sperre (ein Rückruf bleibt erlaubt).
	zukunft := time.Now().AddDate(1, 0, 0).Format("2006-01-02")
	for _, f := range l.faelle() {
		req := httptest.NewRequest(http.MethodPatch, "/api/admin/ausleihen/"+f.id+"/faelligkeit",
			strings.NewReader(`{"faellig_am":"`+zukunft+`"}`))
		req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey,
			&auth.Claims{UserID: adminID, Rolle: auth.RoleAdmin}))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		soll := http.StatusForbidden
		if f.geht {
			soll = http.StatusOK
		}
		if rec.Code != soll {
			t.Errorf("%s: Status %d, erwartet %d: %s", f.was, rec.Code, soll, rec.Body.String())
		}
		l.pruefeFrist(t, f)
	}
}

func TestVerlaengerung_EhemaligeSperreNichtAmSchulbuch_Klasse(t *testing.T) {
	l := sperrFristLageAnlegen(t)
	rec := httptest.NewRecorder()
	l.srv.GlobalExtendLMFHandler()(rec, httptest.NewRequest(http.MethodPost, "/api/ausleihen/global-extend-lmf",
		strings.NewReader(`{"klasse":"10r1","neues_rueckgabe_datum":"2027-06-30"}`)))
	if rec.Code != http.StatusOK {
		t.Fatalf("Klassenverlängerung: %d %s", rec.Code, rec.Body.String())
	}
	var antwort struct {
		UpdatedCount int `json:"updated_count"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
		t.Fatal(err)
	}
	if antwort.UpdatedCount != 1 {
		t.Errorf("verlängert: %d, erwartet genau das Schulbuch des Ehemaligen", antwort.UpdatedCount)
	}
	for _, f := range l.faelle() {
		l.pruefeFrist(t, f)
	}
}

func TestVerlaengerung_EhemaligeSperreNichtAmSchulbuch_LmfPlan(t *testing.T) {
	l := sperrFristLageAnlegen(t)
	von := repository.SchuljahrBeginn(time.Date(2027, time.June, 28, 0, 0, 0, 0, schulzeit.Zone()))
	bis := von.AddDate(1, 0, 0)
	n, err := repository.NewLmfTerminRepository(l.pool).SetzeLernmittelFristFuerKlassenIn(context.Background(),
		l.pool, []string{"10r1"}, fristEnde(2027, time.June, 28), von, bis, nil)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("umgeschrieben: %d, erwartet genau das Schulbuch des Ehemaligen", n)
	}
	for _, f := range l.faelle() {
		l.pruefeFrist(t, f)
	}
}
