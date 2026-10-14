package mahnwesen

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/internal/pdftest"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/smtptest"
	"bibliothek/internal/testdaten"
	"bibliothek/internal/tuertest"
	"bibliothek/pkg/lmfplan"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Die Mahn-Kette: Einstellung → Frist am Buch → überfällig → Mail an die Klassenleitung.
//
// Jedes Glied ist geprüft, aber jedes für sich und mit nachgespielten Antworten: Die
// Überfälligkeits-Abfrage kennt bisher nur pgxmock — ihr SQL ist also nie gegen ein echtes
// Postgres gelaufen. Genau dort entscheidet sich, wer gemahnt wird.
//
// Die Übergänge, an denen es still auseinanderlaufen kann:
//
//   - Die Leihfrist kommt aus einer Einstellung. Käme sie in der Datenbank anders an,
//     würde zu früh oder gar nicht gemahnt — beides fällt erst Wochen später auf.
//   - Die Klassenleitung wird über den Klassennamen zugeordnet. Schülerdaten kommen aus
//     dem LUSD-Import, die Klassenleitungen tippt ein Mensch: "5A" gegen "5a " ist der Normalfall,
//     nicht die Ausnahme. Ohne Adresse wird die Klasse still übersprungen ("skipped") —
//     der Lauf meldet Erfolg, und die Mahnung geht nie raus.
//   - Jede Lehrkraft darf nur die eigene Klasse sehen (Datenminimierung).
//   - Die Mail erhöht die Mahnstufe nicht (Invariante §1, docs/invarianten.md): Das ist ein
//     freundlicher Hinweis vor der Eskalation, nicht die Eskalation selbst.

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

func TestMahnkette_FristAusDerEinstellungBisZurMailAnDieKlassenleitung(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()

	// 1. Die Einstellung, aus der die Frist entsteht. Bewusst kein Vielfaches von 7 und
	//    kein Standardwert: Ein übersehener Fallback (21 Tage) fiele sonst nicht auf.
	// Fristen rücken auf den nächsten Schultag; am 24.09.2026 landen 13 und 21 Tage beide in
	// den Herbstferien und damit auf demselben Montag. Deshalb ab 13 die erste Tageszahl, deren
	// Frist sich von der des Fallbacks unterscheidet.
	tage := 13
	for tage%7 == 0 || lmfplan.Hessen().Tagesfrist(time.Now(), tage).Equal(lmfplan.Hessen().Tagesfrist(time.Now(), 21)) {
		tage++
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO system_einstellungen (schluessel, wert) VALUES ('frist_buch_tage', $1)
		ON CONFLICT (schluessel) DO UPDATE SET wert = $1`, fmt.Sprint(tage)); err != nil {
		t.Fatalf("Leihfrist setzen: %v", err)
	}

	// Kein LMF im Titel: Schulbücher laufen bis zum Stichtag im Sommer, nicht nach Tagen.
	testdaten.LegeAusleihbaresExemplarAn(t, pool, "Roman ohne Kennzeichen", "B-MAHN-1")
	testdaten.LegeAusleihbaresExemplarAn(t, pool, "Zweiter Roman", "B-MAHN-2")

	// Schreibweise absichtlich verschieden: Schülerdaten "5A", Klassenleitung "5a " mit
	// Leerzeichen — genau die Kombination, die aus dem LUSD-Import und einem Formular
	// entsteht. Seit Migration 079 kanonisiert das Klassen-Vokabular beide beim
	// Schreiben auf dieselbe registrierte Schreibweise; die Mail muss trotzdem (bzw.
	// gerade deshalb) ankommen. Welche Schreibweise gewinnt, entscheidet der
	// Erstschreiber — der Test liest sie unten aus der DB, wie es auch die UI tut.
	anna := testdaten.LegeSchuelerAn(t, pool, "S-MAHN-1", "Anna", "5A")
	ben := testdaten.LegeSchuelerAn(t, pool, "S-MAHN-2", "Ben", "6B")
	testdaten.SetzeKlassenleitung(t, pool, "5a ", "leitung5a@schule.invalid")
	testdaten.SetzeKlassenleitung(t, pool, "6B", "leitung6b@schule.invalid")

	var annaKlasse string
	if err := pool.QueryRow(ctx, `SELECT klasse FROM schueler WHERE id = $1::uuid`, anna).Scan(&annaKlasse); err != nil {
		t.Fatalf("kanonisierte Klasse lesen: %v", err)
	}

	// 2. Ausleihe über den echten Dienst → die Frist muss aus der Einstellung stammen.
	bearbeiter := testdaten.LegeBearbeiterAn(t, pool)
	frist := testdaten.BucheAusleiheUeberDienst(t, pool, "B-MAHN-1", anna, bearbeiter)
	// Beide Seiten durch dieselbe Definition des Betriebs schicken
	// (lmfplan.Ferientabelle.Tagesfrist), statt Kalendertage in der Zeitzone des Testrechners zu
	// vergleichen: Die Frist entsteht in der Schul-Zeitzone (Europe/Berlin), time.Now() steht in
	// der Zeitzone des Runners. Auf dem UTC-CI sind das ab 22:00 UTC zwei verschiedene
	// Kalendertage. internal/service/frist_test.go (istGleicherTag) normalisiert ebenso.
	// Nachstellbar ohne Warten auf Mitternacht:
	// TZ=Pacific/Midway go test ./api/mahnwesen/ -run TestMahnkette
	// AddDate nach der Umrechnung in die Schulzeitzone, sonst rechnet der Test im Kalender des
	// Runners und weicht am Ende der Sommerzeit (24.10., 22–23 UTC) um einen Tag ab. Tagesfrist
	// rechnet genau so.
	erwartet := lmfplan.Hessen().Tagesfrist(time.Now(), tage)
	if !schulzeit.TagesEnde(frist).Equal(erwartet) {
		t.Fatalf("Rückgabefrist = %s, erwartet den %s (%d Tage aus der Einstellung)",
			frist.Format("02.01.2006"), erwartet.Format("02.01.2006"), tage)
	}
	testdaten.BucheAusleiheUeberDienst(t, pool, "B-MAHN-2", ben, bearbeiter)

	// 3. Überfällig werden lassen — und prüfen, wen die Abfrage findet.
	inDieVergangenheit(t, pool, anna, 9)
	inDieVergangenheit(t, pool, ben, 2)

	mahnRepo := repository.NewMahnwesenAbfragen(pool)
	klassen, err := mahnRepo.LadeUeberfaelligeNachKlasse(ctx, "")
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

	// 4. Der Lauf selbst: nur Annas Klasse auswählen (in der Schreibweise, die die
	// Kanonisierung gespeichert hat — die UI bietet exakt diese Liste an).
	sitzungen := tuertest.MailAbfangen(t)
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Database{Pool: pool}}}

	req := httptest.NewRequest(http.MethodPost, "/api/mail/send-bulk-overdue",
		strings.NewReader(fmt.Sprintf(`{"klassen":[%q]}`, annaKlasse)))
	req.Header.Set("Content-Type", "application/json")
	// Ohne Claims schreibt die Tür keinen Eintrag ins Protokoll, und dieser Lauf braucht einen.
	req = req.WithContext(context.WithValue(ctx, auth.ClaimsContextKey,
		&auth.Claims{UserID: bearbeiter, Rolle: auth.RoleAdmin}))

	rec := httptest.NewRecorder()
	srv.MahnlistenVersenden(mahnRepo)(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Mahnlauf: Status %d — %s", rec.Code, rec.Body.String())
	}
	// Beide Zahlen, nicht nur sent_count: Der Testserver nimmt genau eine Sitzung an, ein
	// zweiter Versand scheitert also an der Verbindung und zählt als "skipped". Ohne diese
	// Zeile bestünde der Test auch dann, wenn der Lauf an alle Klassen ginge — die Mail an
	// die falsche Klassenleitung wäre schon raus, und der Test bliebe grün.
	if antwort := rec.Body.String(); !strings.Contains(antwort, `"sent_count":1`) ||
		!strings.Contains(antwort, `"skipped_count":0`) {
		t.Fatalf("Mahnlauf soll genau eine Mail verschicken und keine überspringen: %s", antwort)
	}

	// 5. Was wirklich rausging.
	nachricht := smtptest.WarteAufMail(t, sitzungen)
	if !strings.Contains(nachricht, "leitung5a@schule.invalid") {
		t.Errorf("die Mail ging nicht an die Klassenleitung der 5A:\n%s", smtptest.Kopf(nachricht))
	}
	if strings.Contains(nachricht, "leitung6b@schule.invalid") {
		t.Error("die Mail nennt die Klassenleitung einer NICHT gewählten Klasse")
	}
	// Datenminimierung: In der Mail an die 5A hat der Name aus der 6B nichts zu suchen.
	// Der Anhang ist das Mahn-PDF; die Namen stehen dort komprimiert, deshalb wird der
	// PDF-Inhalt gelesen und nicht der Rohtext.
	texte := strings.Join(pdftest.Texte(t, smtptest.PDFAusMail(t, nachricht)), "\n")
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
