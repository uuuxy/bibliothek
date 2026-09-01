package api

// PII-Antwort-Gate (Betreiber-Entscheidung 01.09.2026, Befund-Register): Die
// PII-Matrix war bis hierhin eine Zusage, die nur ein Dokument behauptet — das
// bestehende Gate (pii_matrix_test.go) prüft Route, Recht und Zeilen-Existenz,
// aber nicht, ob die ANTWORT sich an ihre Stufe hält. Genau diese Bauform
// („Schutz, den nur ein Kommentar behauptet") ist dem Projekt mehrfach auf die
// Füße gefallen. Dieses Gate schließt die Lücke für die GET-Routen:
//
//  1. Es sät eine Welt mit KANARIENWERTEN je PII-Stufe — Namen (Stufe 1),
//     Sperrgrund/LUSD/Geburtsdatum/Schadenstext (Stufe 2), Adresse/Eltern-Mail
//     (Stufe 3) — samt überfälliger Ausleihe, Schadensfall, Vormerkung und
//     Audit-Spur, damit die Antworten die Daten auch wirklich führen KÖNNEN.
//  2. Es ruft jede GET-Route der Matrix über den ECHTEN Router auf (Routes()
//     samt Middleware), und zwar mit GENAU dem Recht ihrer Matrix-Zeile — nicht
//     als Admin: Die Stufe ist „bemessen an dem, was das Recht der Zeile ALLEIN
//     öffnet" (Matrix-Kopf), und Zusatzinhalte hinter view_students (Sperrgrund,
//     Geräte-Ausleihername) dürfen hier gerade NICHT auftauchen.
//  3. Es prüft: Kein Kanarienwert einer Stufe oberhalb der dokumentierten darf
//     in der Antwort stehen (PDF-Ströme werden dafür entpackt, pdfText).
//  4. Gegen das Leere-Antwort-Loch („nichts drin → trivial grün") tragen
//     Schlüsselrouten eine POSITIV-Kontrolle: Dort MUSS der erlaubte Kanarienwert
//     erscheinen, sonst misst das Gate nichts.
//
// Vollständigkeit ist erzwungen: Jede GET-Zeile der Matrix braucht einen Aufruf
// ODER einen begründeten Ausschluss — eine neue GET-Route ohne Einordnung wird
// rot. Nicht-GET-Routen bleiben außerhalb (Aufrufe mit Seiteneffekten gehören
// nicht in ein Lese-Gate); ihre Stufen sichert weiterhin die Handarbeit der
// Matrix plus die bestehenden Gates.

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
	"testing"
	"time"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/repository"
	"bibliothek/sse"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ── Kanarienwerte ────────────────────────────────────────────────────────────
// Frei erfundene, kollisionssichere Werte. Buchdaten (Titel, Buch-Barcodes,
// Klassennamen) sind BEWUSST keine Kanarien: Sie sind Stufe-0-Inhalt und dürfen
// überall stehen.
var kanarienJeStufe = map[int][]string{
	1: {"Pruefkanari", "Zugvogel", "Vogelbeere", "SBK-KANARI-1", "SBK-KANARI-2"},
	2: {"Kanarigrund", "LUSD-KANARI-7", "Kanarischaden", "2013-07-19", "19.07.2013"},
	3: {"Kanariweg", "Kanaristadt", "kanari-eltern@example.org"},
}

// verboteneKanarien liefert alle Werte OBERHALB der dokumentierten Stufe.
func verboteneKanarien(stufe int) []string {
	var out []string
	for s := stufe + 1; s <= 3; s++ {
		out = append(out, kanarienJeStufe[s]...)
	}
	return out
}

// ── Die gesäte Welt ──────────────────────────────────────────────────────────

type kanarienWelt struct {
	mitarbeiterID, sessionToken string
	schuelerID, abgaengerID     string
	titelID, exemplarID         string
	buchBarcode                 string
	schadensfallID              string
}

func baueKanarienWelt(t *testing.T, pool *pgxpool.Pool, a *auth.Authenticator) kanarienWelt {
	t.Helper()
	ctx := context.Background()
	w := kanarienWelt{buchBarcode: "BUCHBC-GATE-1"}

	// Personal-Konto (Mitarbeiter-Rolle: bekommt je Route genau EIN Recht).
	// Name bewusst OHNE Kanarienbezug — Personalnamen sind keine Schülerdaten.
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (barcode_id, vorname, nachname, email, rolle, aktiv)
		VALUES ('GATE-MA-1', 'Greta', 'Gatewart', 'gate-ma@example.org', 'mitarbeiter', true)
		RETURNING id`).Scan(&w.mitarbeiterID); err != nil {
		t.Fatalf("Mitarbeiter anlegen: %v", err)
	}
	token, err := a.GenerateToken(w.mitarbeiterID, "GATE-MA-1", auth.RoleMitarbeiter)
	if err != nil {
		t.Fatalf("Session-Token: %v", err)
	}
	w.sessionToken = token

	// Aktiver Schüler mit ALLEN Stufen: Name/Barcode (1), Geburtsdatum, LUSD-ID
	// und Sperrgrund (2), Adresse und Eltern-Mail (3).
	if err := pool.QueryRow(ctx, `
		INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr,
		                      geburtsdatum, lusd_id, is_manually_blocked, block_reason,
		                      strasse, hausnummer, plz, ort, eltern_email)
		VALUES ('SBK-KANARI-1', 'Pruefkanari', 'Vogelbeere', '05A', 2031,
		        '2013-07-19', 'LUSD-KANARI-7', true, 'Kanarigrund Zahlung offen',
		        'Kanariweg', '7', '61169', 'Kanaristadt', 'kanari-eltern@example.org')
		RETURNING id`).Scan(&w.schuelerID); err != nil {
		t.Fatalf("Schüler anlegen: %v", err)
	}
	// Abgänger mit offener Ausleihe — sonst bleibt die Abgängerliste leer und
	// ihre Stufe-2-Zeile wäre trivial grün.
	if err := pool.QueryRow(ctx, `
		INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, ist_abgaenger)
		VALUES ('SBK-KANARI-2', 'Zugvogel', 'Vogelbeere', '05A', 2026, true)
		RETURNING id`).Scan(&w.abgaengerID); err != nil {
		t.Fatalf("Abgänger anlegen: %v", err)
	}

	// Titel + Exemplar. Titeldaten sind Stufe 0 und absichtlich kanarienfrei.
	if err := pool.QueryRow(ctx, `
		INSERT INTO buecher_titel (titel, autor, medientyp, signatur)
		VALUES ('Antwortgate Testband', 'Verlagshaus', 'Buch', 'GATE 1')
		RETURNING id`).Scan(&w.titelID); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
		VALUES ($1, $2, false) RETURNING id`, w.titelID, w.buchBarcode).Scan(&w.exemplarID); err != nil {
		t.Fatalf("Exemplar anlegen: %v", err)
	}

	// Überfällige offene Ausleihe (Mahnwesen, Ausleiher-Listen) für den Schüler.
	var ausleiheID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO ausleihen (exemplar_id, schueler_id, ausgeliehen_am, rueckgabe_frist, bearbeiter_id)
		VALUES ($1, $2, now() - interval '60 days', now() - interval '30 days', $3)
		RETURNING id`, w.exemplarID, w.schuelerID, w.mitarbeiterID).Scan(&ausleiheID); err != nil {
		t.Fatalf("Ausleihe anlegen: %v", err)
	}
	// Zweites Exemplar als OFFENES Buch des Abgängers — die Abgängerliste joint
	// auf rueckgabe_am IS NULL; ohne offenes Buch bliebe sie leer und ihre
	// Stufe-2-Zeile trivial grün.
	var abgaengerExemplarID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
		VALUES ($1, 'BUCHBC-GATE-2', false) RETURNING id`, w.titelID).Scan(&abgaengerExemplarID); err != nil {
		t.Fatalf("Abgänger-Exemplar anlegen: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO ausleihen (exemplar_id, schueler_id, ausgeliehen_am, rueckgabe_frist, bearbeiter_id)
		VALUES ($1, $2, now() - interval '200 days', now() - interval '100 days', $3)`,
		abgaengerExemplarID, w.abgaengerID, w.mitarbeiterID); err != nil {
		t.Fatalf("Abgänger-Ausleihe anlegen: %v", err)
	}

	// Schadensfall (Gebührenliste, Elternbrief, Rechnung) — MIT ausleihe_id:
	// die Rechnung joint über den Ausleihvorgang.
	if err := pool.QueryRow(ctx, `
		INSERT INTO schadensfaelle (exemplar_id, schueler_id, ausleihe_id, beschreibung, betrag)
		VALUES ($1, $2, $3, 'Kanarischaden Wasserschaden Seite 12', 12.50)
		RETURNING id`, w.exemplarID, w.schuelerID, ausleiheID).Scan(&w.schadensfallID); err != nil {
		t.Fatalf("Schadensfall anlegen: %v", err)
	}

	// Vormerkung (Warteliste nennt Name + Klasse).
	if _, err := pool.Exec(ctx, `
		INSERT INTO vormerkungen (titel_id, schueler_id) VALUES ($1, $2)`,
		w.titelID, w.schuelerID); err != nil {
		t.Fatalf("Vormerkung anlegen: %v", err)
	}

	// Audit-Spur über den ECHTEN Schreiber: /api/audit darf sie nicht ausbreiten,
	// die Tresen-Auskunft muss sie (Stufe 2) zeigen.
	if err := repository.NewAuditRepository(pool).
		LogAusleihe(ctx, w.exemplarID, w.schuelerID, "", w.mitarbeiterID); err != nil {
		t.Fatalf("Audit-Spur schreiben: %v", err)
	}

	t.Cleanup(func() {
		aufraeumen(t, pool, `DELETE FROM audit_logs WHERE aktion = 'TRESEN_AUSKUNFT' AND details->>'barcode' = $1`, w.buchBarcode)
		aufraeumen(t, pool, `DELETE FROM audit_log WHERE datensatz_id = $1`, w.exemplarID)
		aufraeumen(t, pool, `DELETE FROM vormerkungen WHERE titel_id = $1`, w.titelID)
		aufraeumen(t, pool, `DELETE FROM schadensfaelle WHERE id = $1`, w.schadensfallID)
		aufraeumen(t, pool, `DELETE FROM ausleihen WHERE exemplar_id IN (SELECT id FROM buecher_exemplare WHERE titel_id = $1)`, w.titelID)
		aufraeumen(t, pool, `DELETE FROM buecher_exemplare WHERE titel_id = $1`, w.titelID)
		aufraeumen(t, pool, `DELETE FROM buecher_titel WHERE id = $1`, w.titelID)
		aufraeumen(t, pool, `DELETE FROM schueler WHERE id IN ($1, $2)`, w.schuelerID, w.abgaengerID)
		// Die MITARBEITER-Rechte auf die Seed-Vorgabe zurückstellen — das Gate
		// hat sie je Route auf genau ein Recht verengt, spätere Tests im selben
		// Binary sollen wieder den Normalzustand sehen.
		aufraeumen(t, pool, `DELETE FROM role_permissions WHERE role = 'MITARBEITER'`)
		for _, e := range db.RechteVorgabe {
			if e.Role == "MITARBEITER" {
				aufraeumen(t, pool, `INSERT INTO role_permissions (role, permission, allowed)
					VALUES ($1, $2, $3) ON CONFLICT (role, permission) DO NOTHING`, e.Role, e.Permission, e.Allowed)
			}
		}
		aufraeumen(t, pool, `DELETE FROM benutzer WHERE id = $1`, w.mitarbeiterID)
		InvalidatePermissionCache()
	})
	return w
}

// ── Aufruf-Verzeichnis ───────────────────────────────────────────────────────

// piiAufruf beschreibt, WIE eine Matrix-Zeile aufgerufen wird. Positiv nennt
// Kanarienwerte, die in der Antwort STEHEN MÜSSEN — die Gegenprobe gegen leere
// Antworten, ohne die dieses Gate nichts misst.
type piiAufruf struct {
	URL     string
	Positiv []string
}

// bauePIIAufrufe füllt Pfad-Parameter mit den gesäten Objekten. Schlüssel ist
// die Route EXAKT wie in der Matrix.
func bauePIIAufrufe(w kanarienWelt) map[string]piiAufruf {
	return map[string]piiAufruf{
		// routes_students.go
		"GET /api/schueler":                         {URL: "/api/schueler", Positiv: []string{"Vogelbeere"}},
		"GET /api/schueler/{id}":                    {URL: "/api/schueler/" + w.schuelerID, Positiv: []string{"Kanariweg", "kanari-eltern@example.org"}},
		"GET /api/schueler/{id}/dsgvo-auskunft":     {URL: "/api/schueler/" + w.schuelerID + "/dsgvo-auskunft", Positiv: []string{"Kanariweg"}},
		"GET /api/schueler/{id}/dsgvo-auskunft/pdf": {URL: "/api/schueler/" + w.schuelerID + "/dsgvo-auskunft/pdf"},
		"GET /api/schueler/deleted":                 {URL: "/api/schueler/deleted"},
		"GET /api/schueler/{barcode_id}/photo":      {URL: "/api/schueler/SBK-KANARI-1/photo"},
		"GET /api/klassen":                          {URL: "/api/klassen"},
		"GET /api/klassen-mapping":                  {URL: "/api/klassen-mapping"},
		"GET /api/abgaenger":                        {URL: "/api/abgaenger", Positiv: []string{"Zugvogel"}},
		"GET /api/abgaenger/pdf":                    {URL: "/api/abgaenger/pdf"},
		"GET /api/schadensfaelle/{id}/pdf":          {URL: "/api/schadensfaelle/" + w.schadensfallID + "/pdf", Positiv: []string{"Vogelbeere"}},
		"GET /api/schueler/{id}/schadensfaelle":     {URL: "/api/schueler/" + w.schuelerID + "/schadensfaelle", Positiv: []string{"Kanarischaden"}},
		"GET /api/mahnwesen":                        {URL: "/api/mahnwesen", Positiv: []string{"Vogelbeere"}},
		"GET /api/mahnwesen/ueberfaellig_jahrgang":  {URL: "/api/mahnwesen/ueberfaellig_jahrgang"},
		"GET /api/mahnwesen/pdf":                    {URL: "/api/mahnwesen/pdf"},

		// routes_books.go
		"GET /api/buecher/titel/{id}/exemplare":      {URL: "/api/buecher/titel/" + w.titelID + "/exemplare"},
		"GET /api/buecher/titel/{id}/ausleiher":      {URL: "/api/buecher/titel/" + w.titelID + "/ausleiher", Positiv: []string{"Vogelbeere"}},
		"GET /api/buecher/titel/{id}/historie":       {URL: "/api/buecher/titel/" + w.titelID + "/historie"},
		"GET /api/buecher/titel/{id}/etiketten":      {URL: "/api/buecher/titel/" + w.titelID + "/etiketten"},
		"GET /api/exemplare/etiketten-offen":         {URL: "/api/exemplare/etiketten-offen"},
		"GET /api/exemplare/etiketten-offen/anzahl":  {URL: "/api/exemplare/etiketten-offen/anzahl"},
		"GET /api/vormerkungen":                      {URL: "/api/vormerkungen?titel_id=" + w.titelID, Positiv: []string{"Pruefkanari"}},
		"GET /api/reservierungen/klassensatz":        {URL: "/api/reservierungen/klassensatz"},
		"GET /api/reservierungen/klassensatz/anzahl": {URL: "/api/reservierungen/klassensatz/anzahl"},
		"GET /api/reservierungen/klassensatz/offen":  {URL: "/api/reservierungen/klassensatz/offen"},
		"GET /api/reservierungen/klassensatz/eigene": {URL: "/api/reservierungen/klassensatz/eigene"},
		"GET /api/geraete":                           {URL: "/api/geraete"},
		"GET /api/anliegen/eigene":                   {URL: "/api/anliegen/eigene"},
		"GET /api/anliegen/offen":                    {URL: "/api/anliegen/offen"},
		"GET /api/anliegen/anzahl":                   {URL: "/api/anliegen/anzahl"},

		// routes_misc.go
		"GET /api/public/opac/suche":                             {URL: "/api/public/opac/suche?q=Antwortgate"},
		"GET /api/monitor/slides":                                {URL: "/api/monitor/slides"},
		"GET /api/public/bestellung/{token}":                     {URL: "/api/public/bestellung/gate-dummy-token"},
		"GET /api/public/bestellung/{token}/etiketten/{groesse}": {URL: "/api/public/bestellung/gate-dummy-token/etiketten/klein"},
		"GET /api/search":                                        {URL: "/api/search?q=Vogelbeere", Positiv: []string{"Pruefkanari"}},
		"GET /api/inventur/sessions":                             {URL: "/api/inventur/sessions"},
		"GET /api/inventur/abgeschlossen":                        {URL: "/api/inventur/abgeschlossen"},
		"GET /api/inventur/fehlbestand":                          {URL: "/api/inventur/fehlbestand"},

		// routes_orders.go
		"GET /api/bestellungen/konfiguration": {URL: "/api/bestellungen/konfiguration"},
		"GET /api/bestellungen":               {URL: "/api/bestellungen"},
		"GET /api/bestellungen/pdf":           {URL: "/api/bestellungen/pdf"},
		"GET /api/bestellhistorie":            {URL: "/api/bestellhistorie"},
		"GET /api/bestellhistorie/uebersicht": {URL: "/api/bestellhistorie/uebersicht"},
		"GET /api/bestellhistorie/bericht":    {URL: "/api/bestellhistorie/bericht?von=2026-01-01&bis=2026-12-31"},
		"GET /api/bestellhistorie/{id}":       {URL: "/api/bestellhistorie/00000000-0000-0000-0000-000000000000"},
		"GET /api/lieferanten":                {URL: "/api/lieferanten"},
		"GET /api/bestellungen/zulauf":        {URL: "/api/bestellungen/zulauf"},

		// routes_system.go
		"GET /api/benutzer":                          {URL: "/api/benutzer"},
		"GET /api/einstellungen":                     {URL: "/api/einstellungen"},
		"GET /api/einstellungen/sitzung":             {URL: "/api/einstellungen/sitzung"},
		"GET /api/ausweis-layout":                    {URL: "/api/ausweis-layout"},
		"GET /api/admin/settings/mail":               {URL: "/api/admin/settings/mail"},
		"GET /api/admin/permissions":                 {URL: "/api/admin/permissions"},
		"GET /api/admin/system/backup-status":        {URL: "/api/admin/system/backup-status"},
		"GET /api/admin/system/betriebsbereitschaft": {URL: "/api/admin/system/betriebsbereitschaft"},
		"GET /api/audit":                             {URL: "/api/audit"},
		"GET /api/audit/tresen-auskunft":             {URL: "/api/audit/tresen-auskunft?barcode=" + w.buchBarcode, Positiv: []string{"Vogelbeere"}},
		"GET /api/mail-templates":                    {URL: "/api/mail-templates"},
		// Anschrift als Positiv-Kontrolle: Der Elternbrief ist ein DIN-5008-
		// Fensterkuvert-Postbrief (Stufe 3, seit 01.09.2026) — verliert er die
		// Anschrift wieder, ist er für den Postversand unbrauchbar, und das soll
		// dieses Gate melden, nicht die Sekretärin am Kuvertiertisch.
		"GET /api/reports/overdue-pdf":             {URL: "/api/reports/overdue-pdf", Positiv: []string{"Vogelbeere", "Kanariweg", "Kanaristadt"}},
		"GET /api/print/rechnung/{schueler_id}":    {URL: "/api/print/rechnung/" + w.schuelerID, Positiv: []string{"Vogelbeere"}},
		"GET /api/print/mahnung/klasse/{klasse}":   {URL: "/api/print/mahnung/klasse/" + url.PathEscape("05A")},
		"GET /api/print/kontoauszug/{schueler_id}": {URL: "/api/print/kontoauszug/" + w.schuelerID, Positiv: []string{"Vogelbeere"}},
		"GET /api/dashboard/summary":               {URL: "/api/dashboard/summary"},
		"GET /api/statistiken":                     {URL: "/api/statistiken"},
		"GET /api/systematics":                     {URL: "/api/systematics"},
		"GET /api/faecher":                         {URL: "/api/faecher"},
		"GET /api/readergroups":                    {URL: "/api/readergroups"},
		"GET /api/admin/auditlog":                  {URL: "/api/admin/auditlog"},
		"GET /api/barcode/next":                    {URL: "/api/barcode/next"},
		"GET /api/barcode":                         {URL: "/api/barcode?content=BUCHBC-GATE-1"},
		"GET /api/print/etikett/{id}":              {URL: "/api/print/etikett/" + w.exemplarID},
		"GET /api/signaturen":                      {URL: "/api/signaturen"},
		"GET /api/signaturen/buecher":              {URL: "/api/signaturen/buecher?signatur=GATE"},

		// router.go
		"GET /api/images/cover": {URL: "/api/images/cover"},
		"GET /api/csrf-token":   {URL: "/api/csrf-token"},
		"GET /api/auth/me":      {URL: "/api/auth/me"},
		"GET /health":           {URL: "/health"},

		// inventur/api_routen.go
		"GET /uploads/":                 {URL: "/uploads/"},
		"GET /api/books":                {URL: "/api/books"},
		"GET /api/books/{id}":           {URL: "/api/books/" + w.titelID},
		"GET /api/class-books":          {URL: "/api/class-books"},
		"GET /api/portal/lernmittel":    {URL: "/api/portal/lernmittel"},
		"GET /api/portal/klassensaetze": {URL: "/api/portal/klassensaetze"},
		"GET /api/admin/":               {URL: "/api/admin/"},
	}
}

// erwarteterFehlstatus: Aufrufe, deren ehrliches Ergebnis KEIN 200 ist — mit dem
// exakt erwarteten Status. Alle anderen Aufrufe müssen < 400 antworten: Sonst
// liefe ein Tippfehler im Verzeichnis still ins SPA-Fallback (404) und die Route
// wäre für immer trivial grün.
var erwarteterFehlstatus = map[string]int{
	"GET /api/schueler/{barcode_id}/photo":                   404, // kein Foto gesät — Fotos liegen AES-verschlüsselt in der DB, eigener Prüfpfad
	"GET /api/bestellhistorie/{id}":                          404, // bewusst die Nullen-UUID: nur der Fehlerpfad ist ohne Bestellwelt erreichbar
	"GET /api/public/bestellung/{token}":                     404, // Dummy-Token — ein echter entstünde nur über den Bestell-Schreibpfad
	"GET /api/public/bestellung/{token}/etiketten/{groesse}": 404, // dito
	"GET /api/admin/":                                        404, // Mount-Sammelpfad des Inventur-Mux — der nackte Pfad trifft keine innere Route; die inneren stehen einzeln in der Matrix
	"GET /uploads/":                                          404, // FileServer auf leerem Cover-Verzeichnis, kein Index — im Gate-Lauf liegen dort keine Dateien
	"GET /api/inventur/fehlbestand":                          400, // verlangt session_id einer echten Inventur; ohne Inventur-Welt ist der Parameter-Fehlerpfad die ehrliche Antwort
}

// bewusstAusgelassen: GET-Zeilen der Matrix, die dieses Gate mit Begründung
// nicht aufruft. Jede Begründung muss den Verzicht tragen — „zu aufwendig"
// stünde hier zu Recht schlecht.
var bewusstAusgelassen = map[string]string{
	"GET /events": "SSE-Stream ohne Ende — ein Lese-Gate würde hängen; der Stream trägt laut " +
		"eigener Registrierung nur operative Events (UUIDs, Buchdaten) und hat ein eigenes Gate (sse-Paket).",
	"GET /swagger/": "nur unter APP_ENV=local/development registriert — im Gate-Lauf existiert die Route nicht; " +
		"Inhalt ist die generierte API-Doku (docs.go), keine Datenbankantwort.",
	"GET /swagger": "Redirect auf /swagger/, gleiche Begründung.",
	"GET /api/lookup/": "Proxy für EXTERNE ISBN-Metadaten — ein Aufruf mit echter ISBN ginge ins Internet; " +
		"die Antwort sind fremde Titeldaten ohne jeden Schülerbezug (Stufe 0 per Konstruktion).",
}

// ── Der Aufruf-Apparat ───────────────────────────────────────────────────────

// setzeGenauEinRecht stellt die MITARBEITER-Rolle auf exakt ein erlaubtes Recht.
// So zeigt jede Antwort, was das Recht ihrer Matrix-Zeile ALLEIN öffnet.
func setzeGenauEinRecht(t *testing.T, pool *pgxpool.Pool, recht string) {
	t.Helper()
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `DELETE FROM role_permissions WHERE role = 'MITARBEITER'`); err != nil {
		t.Fatalf("Rechte leeren: %v", err)
	}
	if recht != "" {
		if _, err := pool.Exec(ctx, `
			INSERT INTO role_permissions (role, permission, allowed) VALUES ('MITARBEITER', $1, true)`,
			recht); err != nil {
			t.Fatalf("Recht %q setzen: %v", recht, err)
		}
	}
	InvalidatePermissionCache()
}

// rechtFuerZeile übersetzt die Rechte-Spalte der Matrix in (Fachrecht, Cookie?).
func rechtFuerZeile(recht string) (permission string, mitSitzung bool) {
	switch recht {
	case "öffentlich", "Token":
		return "", false
	case "Sitzung", "selbst-prüfend":
		return "", true
	case "inventur:view_books":
		return "view_books", true
	case "inventur:edit_books":
		return "edit_books", true
	default:
		return recht, true
	}
}

// antwortText liest den Rumpf; PDF-Ströme werden entpackt (pdfText, dieselbe
// Technik wie die Etiketten-Gates), damit „im PDF steht die Adresse" nicht
// hinter FlateDecode unsichtbar bleibt.
func antwortText(t *testing.T, resp *httptest.ResponseRecorder) string {
	t.Helper()
	roh, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Antwort lesen: %v", err)
	}
	if bytesIndex := strings.Index(string(roh), "%PDF"); bytesIndex >= 0 {
		return pdfText(t, roh)
	}
	return string(roh)
}

func TestPIIAntwortenHaltenIhreStufe(t *testing.T) {
	pool := pgTestPool(t)

	// Rate-Limit hochdrehen: ~80 Requests fallen hier in dieselbe Sekunde, das
	// Produktionsbudget (50/s je IP) wäre ein falsches Rot.
	t.Setenv("RATE_LIMIT", "100000")

	// role_permissions entsteht im echten Betrieb durch InitPermissions (db/seed.go),
	// nicht durch schema.sql — derselbe Weg hier, damit das Gate den Live-Aufbau fährt.
	if err := (&db.Database{Pool: pool}).InitPermissions(context.Background()); err != nil {
		t.Fatalf("InitPermissions: %v", err)
	}

	authenticator, err := auth.NewAuthenticator(
		"pii-antwort-gate-testgeheimnis-mind-32-bytes!", pool, time.Hour)
	if err != nil {
		t.Fatalf("Authenticator: %v", err)
	}
	srv := NewServer(&db.Database{Pool: pool}, authenticator, sse.NewBroker(), false)
	router := srv.Routes()

	welt := baueKanarienWelt(t, pool, authenticator)
	aufrufe := bauePIIAufrufe(welt)
	matrix := leseMatrix(t)

	// Vollständigkeit in beide Richtungen: jede GET-Zeile eingeordnet, kein
	// Eintrag ohne Matrix-Zeile (sonst veraltet das Verzeichnis lautlos).
	getZeilen := map[string]matrixZeile{}
	for route, z := range matrix {
		if strings.HasPrefix(route, "GET ") {
			getZeilen[route] = z
		}
	}
	for route := range aufrufe {
		if _, ok := getZeilen[route]; !ok {
			t.Errorf("Aufruf-Verzeichnis führt %q, die Matrix kennt diese GET-Zeile nicht (mehr) — Eintrag entfernen oder Pfad korrigieren.", route)
		}
	}
	for route, grund := range bewusstAusgelassen {
		if _, ok := getZeilen[route]; !ok {
			t.Errorf("Ausschluss-Liste führt %q ohne Matrix-Zeile — Eintrag entfernen.", route)
		}
		if strings.TrimSpace(grund) == "" {
			t.Errorf("Ausschluss %q ohne Begründung.", route)
		}
	}

	routen := make([]string, 0, len(getZeilen))
	for route := range getZeilen {
		routen = append(routen, route)
	}
	sort.Strings(routen)

	for _, route := range routen {
		zeile := getZeilen[route]
		if _, ok := bewusstAusgelassen[route]; ok {
			continue
		}
		aufruf, ok := aufrufe[route]
		if !ok {
			t.Errorf("GET-Route %q ist im PII-Antwort-Gate nicht eingeordnet.\n"+
				"→ Aufruf in bauePIIAufrufe ergänzen (Pfad-Parameter aus der Kanarienwelt) "+
				"oder mit Begründung in bewusstAusgelassen aufnehmen.", route)
			continue
		}

		stufe := int(zeile.Stufe[0] - '0')
		permission, mitSitzung := rechtFuerZeile(zeile.Recht)
		setzeGenauEinRecht(t, pool, permission)

		req := httptest.NewRequest(http.MethodGet, aufruf.URL, nil)
		if mitSitzung {
			req.AddCookie(&http.Cookie{Name: "session_token", Value: welt.sessionToken})
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		text := antwortText(t, rec)

		// Kern des Gates: nichts oberhalb der dokumentierten Stufe.
		for _, kanarie := range verboteneKanarien(stufe) {
			if strings.Contains(text, kanarie) {
				t.Errorf("%s (Stufe %d, Recht %s): Antwort enthält %q — Daten OBERHALB der dokumentierten Stufe.\n"+
					"→ Entweder blendet der Handler zu wenig aus, oder die Matrix-Stufe ist falsch. Status %d.",
					route, stufe, zeile.Recht, kanarie, rec.Code)
			}
		}

		// Positiv-Kontrolle: Diese Routen MÜSSEN ihre erlaubten Daten zeigen —
		// sonst prüfte das Gate leere Antworten und meldete ewig grün.
		for _, kanarie := range aufruf.Positiv {
			if !strings.Contains(text, kanarie) {
				t.Errorf("%s: Positiv-Kontrolle %q fehlt in der Antwort (Status %d) — die Kanarienwelt erreicht diese Route nicht (mehr); Seed oder Aufruf reparieren, sonst misst das Gate hier nichts.\nAntwort-Anfang: %.200s",
					route, kanarie, rec.Code, text)
			}
		}
		// Status-Härtung gegen still ins Leere laufende Aufrufe (SPA-404, 403
		// wegen falsch übersetztem Recht, 500 wegen kaputtem Seed).
		if erwartet, ok := erwarteterFehlstatus[route]; ok {
			if rec.Code != erwartet {
				t.Errorf("%s: erwartet Status %d (siehe erwarteterFehlstatus), war %d", route, erwartet, rec.Code)
			}
		} else if rec.Code >= 400 {
			t.Errorf("%s: Status %d — der Aufruf erreicht die Route nicht (Tippfehler im Verzeichnis, fehlendes Recht oder kaputter Seed); Antwort-Anfang: %.200s", route, rec.Code, text)
		}
	}
}

// TestPIIAntwortGateErkenntVerstoss ist die Gegenprobe am Detektor: Ein Gate,
// dessen Kanarien nie anschlagen können, meldet ewig „alles gut". Wir stellen
// die Verletzung nach, die das Gate fangen soll — eine Stufe-0-Route, deren
// Antwort einen Stufe-3-Wert trüge, MUSS den Vergleich reißen.
func TestPIIAntwortGateErkenntVerstoss(t *testing.T) {
	for _, verboten := range verboteneKanarien(0) {
		if !strings.Contains((`{"schueler":[{"strasse":"Kanariweg","ort":"Kanaristadt","eltern":"kanari-eltern@example.org","grund":"Kanarigrund","lusd":"LUSD-KANARI-7","geb":"2013-07-19","gebDe":"19.07.2013","schaden":"Kanarischaden","name":"Pruefkanari Zugvogel Vogelbeere","bc":"SBK-KANARI-1 SBK-KANARI-2"}]}`), verboten) {
			t.Errorf("Kanarie %q würde in einer JSON-Antwort nicht erkannt — Suchlogik prüfen", verboten)
		}
	}
	if len(verboteneKanarien(3)) != 0 {
		t.Error("Stufe 3 darf alles — verboteneKanarien(3) muss leer sein")
	}
	if len(verboteneKanarien(0)) != len(kanarienJeStufe[1])+len(kanarienJeStufe[2])+len(kanarienJeStufe[3]) {
		t.Error("verboteneKanarien(0) muss ALLE Kanarien umfassen")
	}
}
