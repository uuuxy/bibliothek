package inventur

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"slices"
	"strings"
	"testing"

	"bibliothek/internal/pdftest"
	"bibliothek/internal/pgtest"
)

// Der Ausdruck der Schulbücher über seine Tür. Abfrage und Blatt haben eigene Tests
// (TestLernmittelJeFach_ZaehltNurSchulbuecher, lernmittel_pdf_test.go); hier zählt, was die Tür
// dazwischen tut: Fach und Filter der Ansicht aus der Adresse lesen, an die Abfrage geben und im
// Kopf des Blatts nennen. Ohne den Hinweis im Kopf sähe ein gefilterter Auszug wie der volle
// Bestand aus.
func TestPortalLernmittelExport_FachUndFilterGeltenUndStehenImKopf(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	tueren := &Tueren{abfragen: NewTitelAbfragen(pool), metadaten: offlineMetadatenClient()}

	// Das Fach eines Titels verweist auf die Systematik; die Titel fallen deshalb vor ihren Fächern.
	raeume := func() {
		for _, sql := range []string{
			`DELETE FROM buecher_titel WHERE titel LIKE 'Exportprobe%'`,
			`DELETE FROM systematik_kategorien WHERE kuerzel LIKE 'EXPORTPROBE-%'`,
		} {
			if _, err := pool.Exec(ctx, sql); err != nil {
				t.Errorf("aufräumen: %v", err)
			}
		}
	}
	raeume()
	t.Cleanup(raeume)
	if _, err := pool.Exec(ctx, `INSERT INTO systematik_kategorien (kuerzel, bezeichnung) VALUES
		('EXPORTPROBE-EK', 'Exportprobe Erdkunde'), ('EXPORTPROBE-PH', 'Exportprobe Physik')`); err != nil {
		t.Fatalf("Probe-Fächer anlegen: %v", err)
	}
	// Jeder Titel scheidet an genau einer Einschränkung der ersten Anfrage aus.
	if _, err := pool.Exec(ctx, `
		INSERT INTO buecher_titel (titel, subject, ist_lernmittel, jahrgang_von, jahrgang_bis, track) VALUES
			('Exportprobe Atlas',     'Exportprobe Erdkunde', true, 7, 7, NULL),
			('Exportprobe Weltkarte', 'Exportprobe Erdkunde', true, 9, 9, NULL),
			('Exportprobe Globus',    'Exportprobe Erdkunde', true, 7, 7, 'Gymnasium'),
			('Exportprobe Optik',     'Exportprobe Physik',   true, 7, 7, NULL)`); err != nil {
		t.Fatalf("Probe-Titel anlegen: %v", err)
	}

	exportiere := func(adresse string) (*httptest.ResponseRecorder, []string) {
		t.Helper()
		rec := httptest.NewRecorder()
		tueren.PortalLernmittelExport(rec, httptest.NewRequest(http.MethodGet, adresse, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("PortalLernmittelExport(%s) = %d %s, erwartet 200", adresse, rec.Code, rec.Body.String())
		}
		if got := rec.Header().Get("Content-Type"); got != "application/pdf" {
			t.Errorf("PortalLernmittelExport(%s): Content-Type %q, erwartet application/pdf", adresse, got)
		}
		pdftest.IstPDF(t, rec.Body.Bytes(), "Schulbuch-Export")
		return rec, pdftest.Texte(t, rec.Body.Bytes())
	}
	// standzeile ist die Zeile unter der Überschrift: Filter, Zahl der Titel, Stand. Das Datum
	// bleibt außen vor, damit ein Lauf um Mitternacht nicht scheitert.
	standzeile := func(texte []string) string {
		t.Helper()
		for _, s := range texte {
			if i := strings.Index(s, " Titel · Stand "); i >= 0 {
				return s[:i+len(" Titel · Stand ")]
			}
		}
		t.Fatalf("keine Standzeile auf dem Blatt: %q", texte)
		return ""
	}
	pruefeTitel := func(texte []string, mit, ohne []string) {
		t.Helper()
		gedruckt := strings.Join(texte, "\n")
		for _, titel := range mit {
			if !strings.Contains(gedruckt, titel) {
				t.Errorf("Export ohne %q. Gedruckt: %s", titel, gedruckt)
			}
		}
		for _, titel := range ohne {
			if strings.Contains(gedruckt, titel) {
				t.Errorf("Export mit %q, den die Auswahl ausschließt. Gedruckt: %s", titel, gedruckt)
			}
		}
	}

	// Ein Fach mit allen drei Filtern: Jahrgang, „ohne Schulzweig" und Suche.
	rec, texte := exportiere("/api/portal/lernmittel/export?fach=Exportprobe+Erdkunde&jahrgang=7&zweig=-&q=Exportprobe")
	dateiname := regexp.MustCompile(`^attachment; filename="schulbuecher_exportprobe-erdkunde_\d{4}-\d{2}-\d{2}\.pdf"$`)
	if got := rec.Header().Get("Content-Disposition"); !dateiname.MatchString(got) {
		t.Errorf("Content-Disposition = %q, erwartet den Dateinamen mit dem Fach", got)
	}
	if !slices.Contains(texte, "Schulbücher · Exportprobe Erdkunde") {
		t.Errorf("Überschrift ohne das Fach. Gedruckt: %q", texte)
	}
	if got, want := standzeile(texte), "Jahrgang 7 · ohne Schulzweig · Suche: Exportprobe · 1 Titel · Stand "; got != want {
		t.Errorf("Standzeile = %q, erwartet %q", got, want)
	}
	pruefeTitel(texte, []string{"Exportprobe Atlas"},
		[]string{"Exportprobe Weltkarte", "Exportprobe Globus", "Exportprobe Optik"})

	// Ohne ?fach alle Fächer, und ein Schulzweig nennt sich mit Namen. Ein Titel ohne Zweig gilt
	// für jeden Zweig und bleibt deshalb in der Auswahl.
	rec, texte = exportiere("/api/portal/lernmittel/export?jahrgang=7&zweig=Gymnasium&q=Exportprobe")
	dateiname = regexp.MustCompile(`^attachment; filename="schulbuecher_\d{4}-\d{2}-\d{2}\.pdf"$`)
	if got := rec.Header().Get("Content-Disposition"); !dateiname.MatchString(got) {
		t.Errorf("Content-Disposition = %q, erwartet den Dateinamen ohne Fach", got)
	}
	if !slices.Contains(texte, "Schulbücher") {
		t.Errorf("Überschrift über alle Fächer fehlt. Gedruckt: %q", texte)
	}
	if got, want := standzeile(texte), "Jahrgang 7 · Gymnasium · Suche: Exportprobe · 3 Titel · Stand "; got != want {
		t.Errorf("Standzeile = %q, erwartet %q", got, want)
	}
	pruefeTitel(texte, []string{"Exportprobe Atlas", "Exportprobe Globus", "Exportprobe Optik"},
		[]string{"Exportprobe Weltkarte"})
}
