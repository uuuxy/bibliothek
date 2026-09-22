package docs

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// Die zweite Hälfte von Raster-Frage 13 — die LESEpfade gegen die Sicht `schueler`.
//
// `schueler` ist seit Migration 124 eine Sicht auf `leser` mit `WHERE art = 'schueler'`.
// schreibpfade_gegen_sicht_test.go führt die Schreibpfade. Gelesen wird gegen die Sicht an
// rund 50 Stellen in 32 Dateien, und die zählte bis zum 21.09.2026 niemand. Ein Lesepfad, der
// die Sicht nimmt, wo er `leser` meint, scheitert nicht — er findet beim Kollegium nur
// nichts. An der Datenbank nachgestellt (OFFEN.md 5.19): Steht eine Vormerkung für einen
// Kollegen, findet die Abfrage, die bei der Rückgabe den Nächsten bedient, null Kandidaten.
//
// Zwei Listen, und der Unterschied ist die Aussage dieser Datei:
//
//   - lesepfadeGeprueft: Für diese Dateien ist BELEGT, dass die Filterung auf Schüler das
//     Gewollte ist. Die Begründung steht daneben.
//   - lesepfadeUngeprueft: Diese Dateien hat noch niemand daraufhin gelesen. Die Liste sagt
//     nicht „in Ordnung", sie sagt „offen" — und sie darf nur schrumpfen. Wer eine Datei
//     prüft, trägt sie mit Begründung nach oben um oder stellt den Pfad auf `leser` um.
//
// Eine Datei, die in KEINER Liste steht und gegen die Sicht liest, ist rot: Ein neuer
// Lesepfad ist eine Frage, und sie wird beim Schreiben beantwortet, nicht später.
//
// BLINDHEIT: Gesucht wird das Textmuster `FROM|JOIN schueler` je Datei mit Zählstand. Nicht
// gesehen werden SQL, das aus Variablen zusammengesetzt wird, ein Komma-Join
// (`FROM a, schueler`), der Name in Anführungszeichen und Lesepfade über ANDERE Sichten.
// Und die Ratsche zählt nur: Ob eine Begründung in lesepfadeGeprueft stimmt, liest ein
// Mensch — am besten an einem PG-Test, der den Pfad mit einer Lehrkraft nachstellt.
var lesepfadeGeprueft = map[string]struct {
	anzahl int
	grund  string
}{
	"api/lusd_apply.go":                      {2, "LUSD-Abgleich sucht über lusd_id — die LUSD kennt nur Schüler"},
	"api/student_promotion.go":               {1, "Versetzung zum Schuljahresende — betrifft ausschließlich Klassen, also Schüler"},
	"jobs/cron_dsgvo.go":                     {2, "Anonymisierung der Abgänger; das Kollegium hat keine Abgangslogik"},
	"internal/littera/schreiber_personen.go": {1, "Zählt nach dem Littera-Schülerlauf die Schüler — das Kollegium zählt derselbe Befehl aus benutzer"},
	// Durchsicht der Theken-Pfade, 21.09.2026 (OFFEN.md 5.19).
	"internal/service/loan_checkout.go":    {2, "Zeilensperre fürs Ausleihlimit nur hinter istSchueler(); Abholfach-Prüfung liest Vormerkungen, und die gibt es nur für Schüler (Tür: VormerkungRepository.Create, TestVormerkungCreate_NurFuerSchueler)"},
	"internal/service/loan_return.go":      {1, "Warteschlange bei der Rückgabe — Vormerkungen gibt es nur für Schüler (Tür: Create)"},
	"repository/vormerkung.go":             {3, "Create prüft gegen die Sicht, DASS es ein Schüler ist; die zwei Listen zeigen deshalb nur Schüler oder Vormerkungen ohne Person"},
	"repository/vormerkung_nachruecken.go": {1, "Nachrücken im Abholfach — Vormerkungen gibt es nur für Schüler (Tür: Create)"},
	"api/ausleihe.go":                      {1, "Massen-Verlängerung der Lernmittel einer KLASSE; ein Kollege leiht auf Dauer (ist_handapparat) und soll keine Klassenfrist bekommen"},
	"repository/mahnwesen_queries.go":      {3, "Gemahnt werden Schüler, nicht das Kollegium (Entscheidung zum Lehrer-Anliegen, Kommentar an der ersten Abfrage); die zwei anderen folgen derselben Auswahl bzw. rechnen über die Klassenstufe"},
	"repository/bescheid.go":               {4, "Bescheide hängen an Forderungen, und ein Kollege bekommt keine (Tür: schaden_melden.go, ohneForderung)"},
	"repository/bescheid_ausstehend.go":    {1, "Offene Forderungen ohne Bescheid — ein Kollege bekommt keine Forderung (Tür: schaden_melden.go)"},
	// Durchsicht der Akte und des Drucks, 21.09.2026.
	"repository/student_profile_queries.go": {1, "Liste der Klassen — Klassen haben nur Schüler"},
	"api/student_create.go":                 {1, "Dublette über Name UND Geburtsdatum gibt es nur bei Schülern; für alle Leser prüft direkt danach pruefeLeserNamensdublette gegen die Tabelle"},
	"api/pdf.go":                            {1, "Elternbrief je Schadensfall — ein Kollege bekommt keine Forderung (Tür: schaden_melden.go)"},
	"api/reports_pdf.go":                    {1, "Mahnbriefe an Eltern — gemahnt werden Schüler, nicht das Kollegium (wie mahnwesen_queries.go)"},
	// Rest der Durchsicht, 22.09.2026 — die Liste der Ungeprüften ist damit leer.
	"api/dsgvo_auskunft.go":                 {1, "Auskunft nur für Schüler — Entscheidung vom 16.09.2026 (OFFEN.md 5.16 A: ausgeblendet statt kaputt); ob Kollegen sie bekommen sollen, ist als Frage in OFFEN.md 5.19 notiert"},
	"api/student_update.go":                 {1, "Vorprüfung der LUSD-ID: chk_leser_nur_schueler_werden_abgaenger erzwingt lusd_id IS NULL für jeden Nicht-Schüler — Sicht und Tabelle liefern hier dasselbe"},
	"cmd/migrate-fotos/main.go":             {1, "Einmal-Werkzeug für Littera-Schülerfotos; eine Nummer ohne Schüler steht als Warnung im Lauf, nicht still"},
	"internal/littera/schreiber.go":         {1, "Zielbestand-Probe zählt die littera:-Herkunft, die nur schreibeSchueler setzt; Kollegen aus Littera werden Konten (benutzer)"},
	"inventur/datenbank_klassen.go":         {2, "Klassengruppen der Inventur — Klassen haben nur Schüler"},
	"jobs/cron_dsgvo_abgaenger.go":          {1, "Löschung der Abgänger; das Kollegium hat keine Abgangslogik (wie cron_dsgvo.go)"},
	"repository/betriebszustand.go":         {3, "Demo-Zähler (scripts/seed_demo.sql legt nur Schüler an, Präfix DEMO-S-), Klassenliste und Wächter Ehemalige (ist_abgaenger gibt es nur bei Schülern)"},
	"repository/lmf_plan.go":                {1, "Klassen im LMF-Plan — Klassen haben nur Schüler"},
	"repository/lmf_termine.go":             {1, "Fristübernahme je KLASSE für Lernmittel; ein Kollege hat keine Klasse und leiht auf Dauer (wie api/ausleihe.go)"},
	"repository/lusd_bestand.go":            {1, "LUSD-Abgleich kennt nur Schüler; die Abfrage sagt es zusätzlich mit art = 'schueler'"},
	"repository/titel_loeschen_wartende.go": {1, "Namen zu Vormerkungen — Vormerkungen gibt es nur für Schüler (Tür: VormerkungRepository.Create)"},
	"api/graduates.go":                      {2, "Abgänger-Liste und -Detail; das Kollegium hat keine Abgangslogik"},
	"api/print.go":                          {3, "Rechnung (ein Kollege bekommt keine Forderung, schaden_melden.go), Mahnung je Klasse (gemahnt werden Schüler) und Kontoauszug (für Kollegen bewusst nicht, OFFEN.md 5.16 A)"},
}

// lesepfadeUngeprueft: seit dem 22.09.2026 leer (28 Dateien am 21./22.09. durchgesehen). NUR SCHRUMPFEN —
// eine neue Datei gehört nach oben, mit Begründung, nicht hierher.
var lesepfadeUngeprueft = map[string]int{}

var musterLesepfad = regexp.MustCompile(`(?i)\b(FROM|JOIN)\s+schueler\b`)

func TestLesepfadeGegenDieSicht_NeueZeileIstEineFrage(t *testing.T) {
	gefunden := map[string]int{}
	wurzel := ".."
	err := filepath.Walk(wurzel, func(pfad string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if name := info.Name(); name == "node_modules" || name == ".git" || name == "frontend" || name == "tmp" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(pfad, ".go") || strings.HasSuffix(pfad, "_test.go") {
			return nil
		}
		roh, err := os.ReadFile(pfad) //nolint:gosec // Repo-eigene Quellen
		if err != nil {
			return err
		}
		// Kommentare zuerst entfernen: Ein Satz wie „liest FROM schueler" in einem Kommentar
		// zählte sonst mit. `schueler_fotos` ist eine eigene Tabelle — die Wortgrenze trennt.
		quelle := musterZeilenKomm.ReplaceAllString(string(roh), "")
		if n := len(musterLesepfad.FindAllString(quelle, -1)); n > 0 {
			rel := strings.TrimPrefix(filepath.ToSlash(strings.TrimPrefix(pfad, wurzel)), "/")
			gefunden[rel] = n
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Baum lesen: %v", err)
	}
	// Nicht-leer-Garantie: Greift das Muster ins Leere, wäre alles Folgende still grün.
	if len(gefunden) < 10 {
		t.Fatalf("nur %d Dateien mit Lesepfaden gegen die Sicht erkannt — das Muster greift vermutlich nicht mehr", len(gefunden))
	}

	var dateien []string
	for datei := range gefunden {
		dateien = append(dateien, datei)
	}
	sort.Strings(dateien)

	for _, datei := range dateien {
		ist := gefunden[datei]
		geprueft, istGeprueft := lesepfadeGeprueft[datei]
		offen, istOffen := lesepfadeUngeprueft[datei]
		switch {
		case istGeprueft && istOffen:
			t.Errorf("%s steht in BEIDEN Listen — geprüft oder ungeprüft, nicht beides.", datei)
		case istGeprueft && ist != geprueft.anzahl:
			t.Errorf("%s liest %d-mal gegen die Sicht `schueler`, geprüft sind %d (%s).\n"+
				"→ Gilt die Begründung auch für die neue Stelle? Dann die Zahl nachziehen — sonst gegen `leser` lesen.",
				datei, ist, geprueft.anzahl, geprueft.grund)
		case istOffen && ist > offen:
			t.Errorf("%s liest %d-mal gegen die Sicht `schueler`, im ungeprüften Bestand stehen %d — die Liste darf nur schrumpfen.\n"+
				"→ Die neue Stelle prüfen: Sollen Lehrkräfte hier wirklich unsichtbar sein? Sonst gegen `leser` lesen.",
				datei, ist, offen)
		case istOffen && ist < offen:
			t.Errorf("%s liest nur noch %d-mal gegen die Sicht, der Eintrag sagt %d — die Zahl nachziehen.", datei, ist, offen)
		case !istGeprueft && !istOffen:
			t.Errorf("%s liest %d-mal gegen die Sicht `schueler` und steht in keiner Liste.\n"+
				"→ `schueler` ist eine SICHT mit WHERE art = 'schueler': Lehrkräfte findet sie nicht. Ist das hier gewollt, "+
				"mit Begründung in lesepfadeGeprueft eintragen — sonst gegen `leser` lesen.", datei, ist)
		}
	}
	for datei := range lesepfadeGeprueft {
		if gefunden[datei] == 0 {
			t.Errorf("lesepfadeGeprueft führt %s, die Datei liest aber nicht mehr gegen die Sicht — austragen.", datei)
		}
	}
	for datei := range lesepfadeUngeprueft {
		if gefunden[datei] == 0 {
			t.Errorf("lesepfadeUngeprueft führt %s, die Datei liest aber nicht mehr gegen die Sicht — austragen.", datei)
		}
	}
}
