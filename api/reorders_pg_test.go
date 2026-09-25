package api

import (
	"context"
	"testing"

	"bibliothek/db"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TestQueryReorders_GesamtNichtVerfuegbar sichert den Meldebestand-Fix ab: Der
// Bestellbedarf richtet sich nach dem BESITZ (gesamt, nicht ausgesondert), nicht nach
// dem gerade verfügbaren Bestand. Sonst würde jeder verliehene Lernmittel-Klassensatz
// (im Schuljahr der Normalfall) als "kritisch nachbestellen" gemeldet — die Liste war
// mit Fehlalarmen geflutet und die vorgeschlagene Menge (Meldebestand − verfügbar) zu hoch.
func TestQueryReorders_GesamtNichtVerfuegbar(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	schueler := seedSchueler(t, pool, "R-1", "Ida", "5a")
	srv := &Server{DB: &db.Database{Pool: pool}}

	// A) Voll verliehener Klassensatz: 30 Exemplare, alle ausgeliehen (verfügbar 0,
	//    gesamt 30), Meldebestand 5 → KEIN Bestellgrund.
	tVoll := titelMitMeldebestand(t, pool, "LMF-Mathe 7", 5)
	for i := 0; i < 30; i++ {
		e := exemplar(t, pool, tVoll, barcodeN("V", i), true, "")
		lendExemplar(t, pool, e, schueler)
	}

	// B) Echter Fehlbestand: nur 3 eigene Exemplare (gesamt 3 < Meldebestand 5).
	tKnapp := titelMitMeldebestand(t, pool, "LMF-Deutsch 5", 5)
	for i := 0; i < 3; i++ {
		exemplar(t, pool, tKnapp, barcodeN("K", i), true, "")
	}

	// C) Bereits bestellt: 2 eigene + 3 "bestellt"-Platzhalter = gesamt 5 → gedeckt,
	//    kein erneuter Bedarf (die "bestellt"-Platzhalter zählen in gesamt mit).
	tBestellt := titelMitMeldebestand(t, pool, "LMF-Bio 6", 5)
	exemplar(t, pool, tBestellt, "BE-1", true, "")
	exemplar(t, pool, tBestellt, "BE-2", true, "")
	for i := 0; i < 3; i++ {
		exemplar(t, pool, tBestellt, barcodeN("BE", i+3), false, "bestellt")
	}

	// Schwelle 5 = früherer Meldebestand-Default; die Auswahllogik bleibt so unverändert.
	reorders, err := srv.queryReorders(ctx, reorderFilterFragmentLMF(), 5)
	if err != nil {
		t.Fatalf("queryReorders: %v", err)
	}

	got := map[string]ReorderTitle{}
	for _, r := range reorders {
		got[r.Titel] = r
	}

	if _, drin := got["LMF-Mathe 7"]; drin {
		t.Error("voll verliehener Klassensatz (gesamt 30) wurde als Bestellbedarf gemeldet — Fehlalarm")
	}
	if _, drin := got["LMF-Bio 6"]; drin {
		t.Error("bereits vollständig bestellter Titel (gesamt 5) wurde erneut gemeldet")
	}
	k, drin := got["LMF-Deutsch 5"]
	if !drin {
		t.Fatal("Titel mit echtem Fehlbestand (gesamt 3 < Meldebestand 5) fehlt in der Liste")
	}
	if k.GesamtBestand != 3 || k.Meldebestand != 5 {
		t.Errorf("Fehlbestand-Zahlen falsch: gesamt=%d meldebestand=%d (erwartet 3 / 5)", k.GesamtBestand, k.Meldebestand)
	}
}

// TestQueryReorders_SchwelleSteuert beweist, dass jetzt die konfigurierbare Schwelle die
// Aufnahme steuert — nicht mehr der pauschale Meldebestand-Default 5. Ein Titel mit
// gesamt 3 erscheint bei Schwelle 5, verschwindet aber bei Schwelle 2.
func TestQueryReorders_SchwelleSteuert(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}

	tKnapp := titelMitMeldebestand(t, pool, "LMF-Physik 8", 5) // meldebestand pauschal 5
	for i := 0; i < 3; i++ {
		exemplar(t, pool, tKnapp, barcodeN("P", i), true, "")
	}

	filter := reorderFilterFragmentLMF()
	r, err := srv.queryReorders(ctx, filter, 5)
	if err != nil {
		t.Fatalf("queryReorders error: %v", err)
	}
	if len(r) != 1 {
		t.Errorf("Schwelle 5: erwartet 1 Treffer (gesamt 3 < 5), waren %d", len(r))
	}

	r2, err := srv.queryReorders(ctx, filter, 2)
	if err != nil {
		t.Fatalf("queryReorders error: %v", err)
	}
	if len(r2) != 0 {
		t.Errorf("Schwelle 2: erwartet 0 Treffer (gesamt 3 ≥ 2) — Schwelle steuert, nicht Meldebestand 5, waren %d", len(r2))
	}
}

// TestQueryReorders_LMFNurInSignatur sichert den Signatur-Fix ab (05.08.2026): Der
// Regelfall bei manueller Neuanlage über die Admin-Oberfläche ist ein Klartext-Titel
// ("Mathematik Neue Wege 9") mit dem LMF-Kennzeichen NUR in der Signatur
// ("LMF Ma" — Auto-Vorschlag). Vor dieser Änderung prüfte queryReorders ausschliesslich
// den Titel und liess solche Bücher nie unter die Bestellbedarf-Schwelle fallen, egal
// wie knapp der Bestand war.
func TestQueryReorders_LMFNurInSignatur(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}

	tID := titelMitSignatur(t, pool, "Mathematik Neue Wege 9", "LMF Ma", 5)
	for i := 0; i < 3; i++ {
		exemplar(t, pool, tID, barcodeN("S", i), true, "")
	}

	reorders, err := srv.queryReorders(ctx, reorderFilterFragmentLMF(), 5)
	if err != nil {
		t.Fatalf("queryReorders: %v", err)
	}

	got := map[string]ReorderTitle{}
	for _, r := range reorders {
		got[r.Titel] = r
	}
	if _, drin := got["Mathematik Neue Wege 9"]; !drin {
		t.Error("LMF-Kennzeichen nur in der Signatur wurde nicht erkannt — Titel fehlt in der Bestellbedarf-Liste")
	}
}

// reorderFilterFragmentLMF liefert das Default-(LMF-)Filterfragment ohne HTTP-Request.
func reorderFilterFragmentLMF() string {
	frag, _ := resolveBestandsFilter("lmf")
	return frag
}

func lendExemplar(t *testing.T, pool *pgxpool.Pool, exemplarID, schuelerID string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist)
		 VALUES ($1, $2, CURRENT_DATE)`, exemplarID, schuelerID); err != nil {
		t.Fatalf("Ausleihe anlegen: %v", err)
	}
}

func barcodeN(prefix string, i int) string {
	return prefix + "-" + string(rune('a'+i%26)) + string(rune('0'+i/26))
}

// TestQueryReorders_ZaehltAmBuch: Zwei Auflagen desselben Buchs (Migration 148,
// docs/OFFEN.md 4.18) stehen als EINE Zeile. Ihre Summe steht gegen die Schwelle, gezeigt
// wird die neueste Auflage — die wird bestellt —, und die Zeile trägt die Aufschlüsselung.
// Getrennt gezählt lag jede für sich unter der Schwelle, und dasselbe Buch stand zweimal da,
// auch wenn beide zusammen reichten. Ein Titel ohne Werk bleibt, wie er war.
func TestQueryReorders_ZaehltAmBuch(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	// CASCADE nimmt buecher_titel mit (werk_id verweist auf werke) — deshalb vor dem Anlegen.
	if _, err := pool.Exec(ctx, `TRUNCATE werke CASCADE`); err != nil {
		t.Fatal(err)
	}
	srv := &Server{DB: &db.Database{Pool: pool}}

	alt := titelMitMeldebestand(t, pool, "LMF-Lambacher Schweizer 7", 5)
	neu := titelMitMeldebestand(t, pool, "LMF-Lambacher Schweizer 7 (Neubearbeitung)", 5)
	einzeln := titelMitMeldebestand(t, pool, "LMF-Physik 8", 5)
	if _, err := pool.Exec(ctx, `
		UPDATE buecher_titel
		SET erscheinungsjahr = CASE WHEN id = $1::uuid THEN 2019 ELSE 2023 END,
		    auflage = CASE WHEN id = $1::uuid THEN '3. Aufl.' ELSE '4. Aufl.' END,
		    isbn = CASE WHEN id = $1::uuid THEN '9783120000019' ELSE '9783120000026' END
		WHERE id IN ($1::uuid, $2::uuid)`, alt, neu); err != nil {
		t.Fatalf("Auflagen beschriften: %v", err)
	}
	exemplar(t, pool, alt, "ALT-1", true, "")
	exemplar(t, pool, alt, "ALT-2", true, "")
	exemplar(t, pool, neu, "NEU-1", true, "")
	exemplar(t, pool, einzeln, "EIN-1", true, "")
	if _, err := repository.FasseAuflagenZusammen(ctx, pool, alt, neu); err != nil {
		t.Fatalf("zusammenfassen: %v", err)
	}
	filter := reorderFilterFragmentLMF()
	nachID := func(liste []ReorderTitle) map[string]ReorderTitle {
		m := map[string]ReorderTitle{}
		for _, r := range liste {
			m[r.ID] = r
		}
		return m
	}

	// Schwelle 3: Zusammen sind es 3 — kein Bedarf. Getrennt stünden beide da (2 und 1).
	r, err := srv.queryReorders(ctx, filter, 3)
	if err != nil {
		t.Fatalf("queryReorders: %v", err)
	}
	got := nachID(r)
	if _, drin := got[alt]; drin {
		t.Error("Schwelle 3: die alte Auflage (2) steht allein da — gezählt wird am Buch (2 + 1 = 3)")
	}
	if _, drin := got[neu]; drin {
		t.Error("Schwelle 3: die neue Auflage (1) steht allein da — gezählt wird am Buch (2 + 1 = 3)")
	}
	if e, drin := got[einzeln]; !drin || e.GesamtBestand != 1 || e.Auflagen != nil {
		t.Errorf("Schwelle 3: Titel ohne Werk %+v — erwartet wie bisher: da, Bestand 1, ohne Aufschlüsselung", e)
	}

	// Schwelle 5: eine Zeile für das Buch — die neueste Auflage, die Summe, die Aufschlüsselung.
	r, err = srv.queryReorders(ctx, filter, 5)
	if err != nil {
		t.Fatalf("queryReorders: %v", err)
	}
	got = nachID(r)
	if _, drin := got[alt]; drin {
		t.Error("Schwelle 5: die alte Auflage hat eine eigene Zeile — das Buch steht zweimal da")
	}
	z, drin := got[neu]
	if !drin {
		t.Fatal("Schwelle 5: keine Zeile für das Buch (Summe 3 < 5)")
	}
	if z.GesamtBestand != 3 || z.VerfuegbarBestand != 3 || z.ISBN != "9783120000026" {
		t.Errorf("Zeile des Buchs: gesamt %d, verfügbar %d, ISBN %q — erwartet 3, 3 und die ISBN der neuesten Auflage",
			z.GesamtBestand, z.VerfuegbarBestand, z.ISBN)
	}
	if len(z.Auflagen) != 2 || z.Auflagen[0].ID != neu || z.Auflagen[0].GesamtBestand != 1 ||
		z.Auflagen[1].ID != alt || z.Auflagen[1].GesamtBestand != 2 || z.Auflagen[1].ISBN != "9783120000019" ||
		z.Auflagen[1].Auflage != "3. Aufl." || z.Auflagen[1].Erscheinungsjahr != 2019 {
		t.Errorf("Aufschlüsselung %+v — erwartet die neue (1) vor der alten (2, 3. Aufl. 2019, mit ISBN)", z.Auflagen)
	}
}
