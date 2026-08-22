package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/internal/service"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Die Mahn-Kette: Einstellung → Frist am Buch → überfällig → Mail an die Klassenleitung.
//
// Jedes Glied ist geprüft, aber jedes für sich und mit nachgespielten Antworten: Die
// Überfälligkeits-Abfrage kennt bisher nur pgxmock — ihr SQL ist also nie gegen ein echtes
// Postgres gelaufen. Genau dort entscheidet sich, WER gemahnt wird.
//
// Die Übergänge, an denen es still auseinanderlaufen kann:
//
//   - Die Leihfrist kommt aus einer Einstellung. Käme sie in der Datenbank anders an,
//     würde zu früh oder gar nicht gemahnt — beides fällt erst Wochen später auf.
//   - Die Klassenleitung wird über den Klassennamen zugeordnet. Schülerdaten kommen aus
//     dem LUSD-Import, das Mapping tippt ein Mensch: "5A" gegen "5a " ist der Normalfall,
//     nicht die Ausnahme. Ohne Adresse wird die Klasse still übersprungen ("skipped") —
//     der Lauf meldet Erfolg, und die Mahnung geht nie raus.
//   - Jede Lehrkraft darf NUR die eigene Klasse sehen (Datenminimierung).
//   - Die Mail erhöht die Mahnstufe NICHT (Invariante §1, docs/invarianten.md): Das ist ein
//     freundlicher Hinweis vor der Eskalation, nicht die Eskalation selbst.

func schuelerAnlegen(t *testing.T, pool *pgxpool.Pool, vorname, klasse, barcode string) string {
	t.Helper()
	var id string
	err := pool.QueryRow(context.Background(), `
		INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		VALUES ($1, $2, 'Testschueler', $3, 2030) RETURNING id`,
		barcode, vorname, klasse).Scan(&id)
	if err != nil {
		t.Fatalf("Schüler %s anlegen: %v", vorname, err)
	}
	return id
}

func klassenleitung(t *testing.T, pool *pgxpool.Pool, klasse, email string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO klassen_lehrer_mapping (klasse, lehrer_email) VALUES ($1, $2)
		ON CONFLICT (klasse) DO UPDATE SET lehrer_email = EXCLUDED.lehrer_email`, klasse, email)
	if err != nil {
		t.Fatalf("Klassenleitung für %s eintragen: %v", klasse, err)
	}
}

// ausleiheUeberDenDienst leiht ein Exemplar an einen Schüler aus — über denselben Dienst,
// den auch der Scanner benutzt, damit die Frist wirklich aus den Einstellungen kommt und
// nicht aus dem Test.
func ausleiheUeberDenDienst(t *testing.T, pool *pgxpool.Pool, barcode, schuelerID, bearbeiterID string) time.Time {
	t.Helper()
	ctx := context.Background()

	bookRepo := repository.NewBookRepository(pool)
	loanSvc := service.NewLoanService(pool, repository.NewStudentRepository(pool), bookRepo,
		repository.NewLoanRepository(pool), repository.NewAuditRepository(pool))

	exemplar, err := bookRepo.GetCopyByBarcode(ctx, barcode)
	if err != nil {
		t.Fatalf("Exemplar %s laden: %v", barcode, err)
	}
	if _, err := loanSvc.HandleUnifiedCheckout(ctx, exemplar, &schuelerID, nil, bearbeiterID, false); err != nil {
		t.Fatalf("Ausleihe von %s: %v", barcode, err)
	}

	var frist time.Time
	if err := pool.QueryRow(ctx,
		`SELECT rueckgabe_frist FROM ausleihen WHERE exemplar_id = $1 AND rueckgabe_am IS NULL`,
		exemplar.ID).Scan(&frist); err != nil {
		t.Fatalf("Frist lesen: %v", err)
	}
	return frist
}

// inDieVergangenheit macht eine laufende Ausleihe überfällig. Der einzige Weg, "sechs
// Wochen später" zu prüfen, ohne sechs Wochen zu warten.
func inDieVergangenheit(t *testing.T, pool *pgxpool.Pool, schuelerID string, tage int) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		UPDATE ausleihen SET rueckgabe_frist = CURRENT_TIMESTAMP - make_interval(days => $2)
		WHERE schueler_id = $1 AND rueckgabe_am IS NULL`, schuelerID, tage)
	if err != nil {
		t.Fatalf("Ausleihe überfällig machen: %v", err)
	}
}

// ausleihbaresExemplar legt Titel und Exemplar an, das sofort ausgeliehen werden kann.
func ausleihbaresExemplar(t *testing.T, pool *pgxpool.Pool, titel, barcode string) {
	t.Helper()
	titelID := titelMitMeldebestand(t, pool, titel, 0)
	exemplar(t, pool, titelID, barcode, true, "")
}

func TestMahnkette_FristAusDerEinstellungBisZurMailAnDieKlassenleitung(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	// 1. Die Einstellung, aus der die Frist entsteht. Bewusst kein Vielfaches von 7 und
	//    kein Standardwert: Ein übersehener Fallback (21 Tage) fiele sonst nicht auf.
	if _, err := pool.Exec(ctx, `
		INSERT INTO system_einstellungen (schluessel, wert) VALUES ('frist_buch_tage', '13')
		ON CONFLICT (schluessel) DO UPDATE SET wert = '13'`); err != nil {
		t.Fatalf("Leihfrist setzen: %v", err)
	}

	// Kein LMF im Titel: Schulbücher laufen bis zum Stichtag im Sommer, nicht nach Tagen.
	ausleihbaresExemplar(t, pool, "Roman ohne Kennzeichen", "B-MAHN-1")
	ausleihbaresExemplar(t, pool, "Zweiter Roman", "B-MAHN-2")

	// Schreibweise absichtlich verschieden: Schülerdaten "5A", Mapping "5a " mit
	// Leerzeichen — genau die Kombination, die aus dem LUSD-Import und einem Formular
	// entsteht. Seit Migration 079 kanonisiert das Klassen-Vokabular beide beim
	// Schreiben auf DIESELBE registrierte Schreibweise; die Mail muss trotzdem (bzw.
	// gerade deshalb) ankommen. Welche Schreibweise gewinnt, entscheidet der
	// Erstschreiber — der Test liest sie unten aus der DB, wie es auch die UI tut.
	anna := schuelerAnlegen(t, pool, "Anna", "5A", "S-MAHN-1")
	ben := schuelerAnlegen(t, pool, "Ben", "6B", "S-MAHN-2")
	klassenleitung(t, pool, "5a ", "leitung5a@schule.invalid")
	klassenleitung(t, pool, "6B", "leitung6b@schule.invalid")

	var annaKlasse string
	if err := pool.QueryRow(ctx, `SELECT klasse FROM schueler WHERE id = $1::uuid`, anna).Scan(&annaKlasse); err != nil {
		t.Fatalf("kanonisierte Klasse lesen: %v", err)
	}

	// 2. Ausleihe über den echten Dienst → die Frist muss aus der Einstellung stammen.
	bearbeiter := adminFuerAudit(t, pool)
	frist := ausleiheUeberDenDienst(t, pool, "B-MAHN-1", anna, bearbeiter)
	// Beide Seiten durch DIESELBE Produktionsdefinition schicken (TagesEndeInSchulzeitzone),
	// statt Kalendertage in der Zeitzone des Testrechners zu vergleichen: Die Frist entsteht
	// in der Schul-Zeitzone (Europe/Berlin), der Test lief mit time.Now() in der Zeitzone des
	// Runners. Auf dem UTC-CI sind das ab 22:00 UTC zwei verschiedene Kalendertage — genau
	// daran war dieser Test in der Nacht auf den 22.08.2026 rot (erwartet 03.09., war 04.09.),
	// und nur in diesem Zwei-Stunden-Fenster. Die Schwestern in
	// internal/service/loan_rules_test.go (sameDay) wurden beim Zeit-Sweep am 19.08.2026 schon
	// so normalisiert; dieser Test hier wurde damals übersehen.
	// Nachstellbar ohne Warten auf Mitternacht: TZ=Pacific/Midway go test ./api/ -run TestMahnkette
	erwartet := service.TagesEndeInSchulzeitzone(time.Now().AddDate(0, 0, 13))
	if !service.TagesEndeInSchulzeitzone(frist).Equal(erwartet) {
		t.Fatalf("Rückgabefrist = %s, erwartet den %s (13 Tage aus der Einstellung)",
			frist.Format("02.01.2006"), erwartet.Format("02.01.2006"))
	}
	ausleiheUeberDenDienst(t, pool, "B-MAHN-2", ben, bearbeiter)

	// 3. Überfällig werden lassen — und prüfen, wen die Abfrage findet.
	inDieVergangenheit(t, pool, anna, 9)
	inDieVergangenheit(t, pool, ben, 2)

	mahnRepo := repository.NewMahnwesenRepository(pool)
	klassen, err := mahnRepo.QueryUeberfaelligeNachKlasse(ctx, "")
	if err != nil {
		t.Fatalf("Überfällige laden: %v", err)
	}
	if len(klassen) != 2 {
		t.Fatalf("überfällige Klassen = %d, erwartet 2", len(klassen))
	}
	for _, kl := range klassen {
		if kl.LehrerEmail == "" {
			t.Fatalf("Klasse %q ohne Klassenleitung — der Lauf überspränge sie als 'skipped', "+
				"und die Mahnung ginge nie raus (Schreibweise 5A/5a?)", kl.Klasse)
		}
	}

	// 4. Der Lauf selbst: NUR Annas Klasse auswählen (in der Schreibweise, die die
	// Kanonisierung gespeichert hat — die UI bietet exakt diese Liste an).
	sitzungen := mailAbfangen(t)
	srv := &Server{DB: &db.Database{Pool: pool}}

	req := httptest.NewRequest(http.MethodPost, "/api/mail/send-bulk-overdue",
		strings.NewReader(fmt.Sprintf(`{"klassen":[%q]}`, annaKlasse)))
	req.Header.Set("Content-Type", "application/json")
	// Ohne Claims schreibt der Handler kein Audit — und dieser Lauf MUSS auditiert werden.
	req = req.WithContext(context.WithValue(ctx, auth.ClaimsContextKey,
		&auth.Claims{UserID: bearbeiter, Rolle: auth.RoleAdmin}))

	rec := httptest.NewRecorder()
	srv.SendBulkOverdueHandler(mahnRepo)(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Mahnlauf: Status %d — %s", rec.Code, rec.Body.String())
	}
	// BEIDE Zahlen, nicht nur sent_count: Der Testserver nimmt genau EINE Sitzung an, ein
	// zweiter Versand scheitert also an der Verbindung und zählt als "skipped". Ohne diese
	// Zeile bestünde der Test auch dann, wenn der Lauf an ALLE Klassen ginge — die Mail an
	// die falsche Klassenleitung wäre schon raus, und der Test bliebe grün.
	if antwort := rec.Body.String(); !strings.Contains(antwort, `"sent_count":1`) ||
		!strings.Contains(antwort, `"skipped_count":0`) {
		t.Fatalf("Mahnlauf soll genau eine Mail verschicken und keine überspringen: %s", antwort)
	}

	// 5. Was wirklich rausging.
	nachricht := warteAufMail(t, sitzungen)
	if !strings.Contains(nachricht, "leitung5a@schule.invalid") {
		t.Errorf("die Mail ging nicht an die Klassenleitung der 5A:\n%s", kopf(nachricht))
	}
	if strings.Contains(nachricht, "leitung6b@schule.invalid") {
		t.Error("die Mail nennt die Klassenleitung einer NICHT gewählten Klasse")
	}
	// Datenminimierung: In der Mail an die 5A hat der Name aus der 6B nichts zu suchen.
	// Der Anhang ist das Mahn-PDF; die Namen stehen dort komprimiert, deshalb wird der
	// PDF-Inhalt gelesen und nicht der Rohtext.
	texte := strings.Join(pdfTexte(t, pdfAusMail(t, nachricht)), "\n")
	if !strings.Contains(texte, "Anna") {
		t.Errorf("die gemahnte Schülerin steht nicht im PDF:\n%s", texte)
	}
	if strings.Contains(texte, "Ben") {
		t.Error("das Mahn-PDF der 5A enthält einen Schüler der 6B — klassenübergreifende Offenlegung")
	}

	// 6. Invariante: Die Mail eskaliert nicht.
	var stufe int
	if err := pool.QueryRow(ctx,
		`SELECT coalesce(max(mahnstufe), 0) FROM ausleihen WHERE rueckgabe_am IS NULL`).Scan(&stufe); err != nil {
		t.Fatalf("Mahnstufe lesen: %v", err)
	}
	if stufe != 0 {
		t.Errorf("Mahnstufe = %d nach dem Mail-Versand, erwartet 0 — die Stufe gehört an den "+
			"PDF-Druck, nicht an den freundlichen Hinweis (Invariante §1)", stufe)
	}

	// 7. Der Lauf ist auditiert — Absicht und Ergebnis.
	var eintraege int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM audit_logs WHERE aktion = 'BULK_OVERDUE_MAIL'`).Scan(&eintraege); err != nil {
		t.Fatalf("Audit lesen: %v", err)
	}
	if eintraege != 2 {
		t.Errorf("Audit-Einträge = %d, erwartet 2 (Absicht vor dem Versand, Ergebnis danach)", eintraege)
	}
}

// pdfAusMail holt den ersten PDF-Anhang aus der Nachricht.
func pdfAusMail(t *testing.T, nachricht string) []byte {
	t.Helper()
	return []byte(dateiAusMail(t, nachricht, "application/pdf"))
}

// adminFuerAudit legt den Benutzer an, auf den der Audit-Eintrag zeigt.
func adminFuerAudit(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	var id string
	err := pool.QueryRow(context.Background(), `
		INSERT INTO benutzer (email, vorname, nachname, rolle, aktiv)
		VALUES ('mahnlauf@schule.invalid', 'Mahn', 'Lauf', 'admin', true)
		ON CONFLICT (email) DO UPDATE SET aktiv = true
		RETURNING id`).Scan(&id)
	if err != nil {
		t.Fatalf("Admin anlegen: %v", err)
	}
	return id
}
