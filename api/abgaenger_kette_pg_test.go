package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Die Abgänger-Kette: Abschlussklasse → Liste → Kontoauszug per Mail → Rückgabe → laut
// LUSD weg → Löschung.
//
// Am Ende dieser Kette steht der einzige unumkehrbare Schritt des Systems: Der Datensatz
// wird endgültig entfernt. Was davor schiefgeht, merkt niemand mehr — der Schüler ist weg,
// die Bücher sind weg, und die Zahlen sehen aus wie immer.
//
// Was hier NICHT noch einmal geprüft wird: Sperre, Blockade und Purge selbst; das steht in
// TestAbgaengerRetentionKette. Hier geht es um die Übergänge dazwischen, die keiner ansieht:
//
//   - Liste und Kontoauszug sind ZWEI Abfragen mit zwei verschiedenen Klassen-Vergleichen
//     (die Liste normalisiert per lower/btrim, der Kontoauszug vergleicht exakt). Der
//     Kommentar im Code sagt „Papier und Mail zeigen denselben Stand" — das ist eine
//     Behauptung, kein Test.
//   - Der Kontoauszug geht als EIN PDF JE SCHÜLER raus. Ob in Annas PDF auch Annas Bücher
//     stehen, hat bisher nichts geprüft: Der vorhandene Test zählt Anhänge und erzeugt die
//     PDFs mit einem Platzhalter.
//   - Nach der Rückgabe muss der Abgänger aus BEIDEN Ansichten verschwinden. Bleibt er in
//     einer, bekommt eine Lehrkraft im nächsten Lauf einen Kontoauszug für ein Buch, das
//     längst im Regal steht.
//   - Die Löschung anonymisiert die Ausleihhistorie (schueler_id = NULL), statt sie zu
//     entfernen. Ginge die Zeile mit, verlöre die Bibliothek rückwirkend ihre Ausleihzahlen
//     — und keine Statistik würde es melden, die Summe wäre einfach kleiner.

// abgaengerMitBuch legt einen Schüler einer Abschlussklasse mit genau einem offenen Buch
// an — noch an der Schule (ist_abgaenger = false), so wie die Liste ihn im Juni sieht.
func abgaengerMitBuch(t *testing.T, pool *pgxpool.Pool, barcode, vorname, klasse, titel string) (schuelerID, ausleiheID string) {
	t.Helper()
	schuelerID = seedSchueler(t, pool, barcode, vorname, klasse)
	ausleiheID = seedAusleihe(t, pool, schuelerID, titel, time.Now().AddDate(0, 0, -3))
	return schuelerID, ausleiheID
}

// abgaengerListe ruft GET /api/abgaenger — die Ansicht, aus der heraus versendet wird.
func abgaengerListe(t *testing.T, srv *Server) []AbgaengerZeile {
	t.Helper()
	return abgaengerAntwort(t, srv).Abgaenger
}

func namenAus(liste []AbgaengerZeile) map[string]bool {
	namen := map[string]bool{}
	for _, e := range liste {
		namen[e.Vorname] = true
	}
	return namen
}

func kontoauszugNamen(t *testing.T, srv *Server) map[string]bool {
	t.Helper()
	eintraege, err := srv.queryAbgaengerKontoauszug(context.Background(), "")
	if err != nil {
		t.Fatalf("Kontoauszug laden: %v", err)
	}
	namen := map[string]bool{}
	for _, e := range eintraege {
		namen[e.Schueler.Vorname] = true
	}
	return namen
}

func TestAbgaengerkette_VomLaufzettelBisZurGeloeschtenAkte(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	sitzungen := mailAbfangen(t)
	srv := &Server{DB: &db.Database{Pool: pool}, Uhr: saisonUhr}

	// Zwei Abgänger derselben Abschlussklasse und einer aus einer anderen — plus die
	// Schreibweise, an der sich schon zweimal etwas aufgehängt hat: Schülerdaten "9H",
	// Mapping "9h ".
	anna, annaAusleihe := abgaengerMitBuch(t, pool, "S-ABG-1", "Anna", "9H", "Annas Buch")
	ben, benAusleihe := abgaengerMitBuch(t, pool, "S-ABG-2", "Ben", "9H", "Bens Buch")
	abgaengerMitBuch(t, pool, "S-ABG-3", "Cem", "10R1", "Cems Buch")
	klassenleitung(t, pool, "9h ", "leitung9h@schule.invalid")
	klassenleitung(t, pool, "10R1", "leitung10r1@schule.invalid")

	// 1. Beide Ansichten müssen dieselben Personen kennen.
	liste := abgaengerListe(t, srv)
	ausListe, ausKonto := namenAus(liste), kontoauszugNamen(t, srv)
	for _, name := range []string{"Anna", "Ben", "Cem"} {
		if !ausListe[name] || !ausKonto[name] {
			t.Fatalf("%s steht in Liste=%v / Kontoauszug=%v — Papier und Mail zeigen verschiedene Stände",
				name, ausListe[name], ausKonto[name])
		}
	}

	// Die Liste muss die Klassenleitung schon VOR dem Versand nennen, sonst kann niemand
	// sehen, dass eine Klasse gleich stillschweigend übersprungen wird.
	for _, e := range liste {
		if e.Vorname == "Anna" && e.LehrerEmail != "leitung9h@schule.invalid" {
			t.Errorf("Klassenleitung der 9H fehlt in der Liste: %v (Schreibweise 9H/9h?)", e.LehrerEmail)
		}
	}

	// 2. Versand NUR für die 9H.
	req := httptest.NewRequest(http.MethodPost, "/api/abgaenger/mail", strings.NewReader(`{"klassen":["09H"]}`))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(ctx, auth.ClaimsContextKey,
		&auth.Claims{UserID: adminFuerAudit(t, pool), Rolle: auth.RoleAdmin}))
	rec := httptest.NewRecorder()
	srv.SendAbgaengerKontoauszuegeHandler()(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Versand: Status %d — %s", rec.Code, rec.Body.String())
	}
	// Beide Zahlen: Ohne skipped_count bestünde der Test auch, wenn zusätzlich an die 10R1
	// ginge — die zweite Mail scheiterte am Testserver und zählte still als übersprungen.
	if antwort := rec.Body.String(); !strings.Contains(antwort, `"sent_count":1`) ||
		!strings.Contains(antwort, `"skipped_count":0`) {
		t.Fatalf("erwartet genau eine Mail ohne Übersprungene: %s", antwort)
	}

	// 3. Was wirklich rausging: eine Mail, zwei PDFs, jedes mit den Büchern SEINES Schülers.
	nachricht := warteAufMail(t, sitzungen)
	if !strings.Contains(nachricht, "leitung9h@schule.invalid") {
		t.Errorf("Mail ging nicht an die Klassenleitung der 9H:\n%s", kopf(nachricht))
	}
	if strings.Contains(nachricht, "leitung10r1@schule.invalid") {
		t.Error("die Mail nennt die Klassenleitung einer nicht gewählten Klasse")
	}

	annaPDF := pdfTexte(t, []byte(dateiAusMail(t, nachricht, "Anna")))
	pruefeKontoauszug(t, annaPDF, "Anna", "Annas Buch", []string{"Bens Buch", "Cems Buch"})
	benPDF := pdfTexte(t, []byte(dateiAusMail(t, nachricht, "Ben")))
	pruefeKontoauszug(t, benPDF, "Ben", "Bens Buch", []string{"Annas Buch", "Cems Buch"})

	// 4. Anna gibt ihr Buch zurück — und verschwindet aus BEIDEN Ansichten.
	if _, err := pool.Exec(ctx, `UPDATE ausleihen SET rueckgabe_am = now() WHERE id = $1`, annaAusleihe); err != nil {
		t.Fatalf("Rückgabe: %v", err)
	}
	ausListe, ausKonto = namenAus(abgaengerListe(t, srv)), kontoauszugNamen(t, srv)
	if ausListe["Anna"] || ausKonto["Anna"] {
		t.Errorf("Anna steht nach der Rückgabe weiter in Liste=%v / Kontoauszug=%v — die Klassenleitung "+
			"bekäme einen Kontoauszug für ein Buch im Regal", ausListe["Anna"], ausKonto["Anna"])
	}
	if !ausListe["Ben"] || !ausKonto["Ben"] {
		t.Error("Ben ist mit Annas Rückgabe aus der Ansicht gefallen")
	}

	// 5. Der unumkehrbare Schritt — und was er stehen lassen MUSS. Vorher geht die Kohorte
	// von der Schule: Der LUSD-Import findet Anna und Ben im Herbst nicht mehr und lässt sie
	// so zurück (Flag + ABG). Erst DANN ist die Löschung überhaupt Thema.
	if _, err := pool.Exec(ctx, `UPDATE schueler SET ist_abgaenger = true, klasse = 'ABG', abgaenger_seit = now()
		WHERE id = ANY($1)`, []string{anna, ben}); err != nil {
		t.Fatalf("Abgang laut LUSD: %v", err)
	}
	auditRepo := repository.NewAuditRepository(pool)
	if err := auditRepo.PurgeAbgaenger(ctx, anna, ""); err != nil {
		t.Fatalf("Purge nach Rückgabe: %v", err)
	}
	if err := auditRepo.PurgeAbgaenger(ctx, ben, ""); err == nil {
		t.Error("Ben wurde trotz offenem Buch endgültig gelöscht")
	}

	// Die Ausleihe bleibt als anonyme Zeile stehen: Sie ist die Ausleihstatistik der
	// Bibliothek. Verschwände sie mit dem Schüler, sänken die Zahlen rückwirkend, ohne
	// dass irgendwo etwas fehlschlüge.
	var schuelerID *string
	var exemplarVorhanden bool
	err := pool.QueryRow(ctx, `
		SELECT a.schueler_id, e.id IS NOT NULL
		FROM ausleihen a JOIN buecher_exemplare e ON e.id = a.exemplar_id
		WHERE a.id = $1`, annaAusleihe).Scan(&schuelerID, &exemplarVorhanden)
	if err != nil {
		t.Fatalf("Ausleihhistorie nach dem Löschen verloren: %v", err)
	}
	if schuelerID != nil {
		t.Errorf("die Ausleihe zeigt nach der Löschung weiter auf einen Schüler (%q) — die Akte "+
			"ist gelöscht, der Personenbezug aber nicht", *schuelerID)
	}
	if !exemplarVorhanden {
		t.Error("die Ausleihe hat ihr Exemplar verloren")
	}

	// Und Bens Buch ist weiterhin unterwegs — die Löschung hat nichts danebengeräumt.
	var offen int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM ausleihen WHERE id = $1 AND rueckgabe_am IS NULL`, benAusleihe).Scan(&offen); err != nil {
		t.Fatalf("Bens Ausleihe prüfen: %v", err)
	}
	if offen != 1 {
		t.Error("Bens offene Ausleihe ist beim Löschen von Annas Akte mitverschwunden")
	}
}

// pruefeKontoauszug prüft ein einzelnes Schüler-PDF auf seinen Inhalt.
func pruefeKontoauszug(t *testing.T, texte []string, name, eigenesBuch string, fremdeBuecher []string) {
	t.Helper()
	inhalt := strings.Join(texte, "\n")
	if !strings.Contains(inhalt, name) {
		t.Errorf("%s steht nicht auf dem eigenen Kontoauszug:\n%s", name, inhalt)
	}
	if !strings.Contains(inhalt, eigenesBuch) {
		t.Errorf("%q fehlt auf dem Kontoauszug von %s — das Blatt nennt nicht, was zurückzugeben ist:\n%s",
			eigenesBuch, name, inhalt)
	}
	for _, fremd := range fremdeBuecher {
		if strings.Contains(inhalt, fremd) {
			t.Errorf("%q steht auf dem Kontoauszug von %s — fremde Ausleihdaten auf dem Blatt", fremd, name)
		}
	}
}
