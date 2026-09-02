package api

import (
	"context"
	"errors"
	"testing"

	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Zusammenführen zweier Datensätze — das Sicherheitsnetz hinter der Vorschau-Paarung,
// am echten Postgres: Vorgänge wandern, die Quelle verschwindet, das Ziel trägt die
// LUSD-frischen Stammdaten, Sperre und Abgänger-Stempel werden bereinigt, und die
// Protokollspuren zeigen auf den verbliebenen Datensatz.

func zfAuftrag(ziel, quelle string) repository.ZusammenfuehrenAuftrag {
	return repository.ZusammenfuehrenAuftrag{ZielID: ziel, QuelleID: quelle, AbgaengerJahr: calculateAbgaengerJahr}
}

func zfZaehle(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestZusammenfuehren_QuelleGehtImZielAuf(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	// Ziel: der alte Datensatz mit Ausweis — vom letzten Jahr bestätigt, jetzt Abgänger in
	// der Karenz (der Export hat ihn unter neuem Namen nicht wiedererkannt).
	ziel := legeUmbSchuelerAn(t, pool, umbSchueler{vorname: "Alt", nachname: "Name", klasse: "07A", barcode: "ZF-1", geb: datum(2012, 3, 3), abgaenger: true})
	// Quelle: der frisch importierte Datensatz — von der LUSD soeben bestätigt, mit
	// Anschrift und Klasse, und mit einer offenen Ausleihe, die schon an ihm hängt.
	quelle := legeUmbSchuelerAn(t, pool, umbSchueler{vorname: "Neu", nachname: "Name", klasse: "08A", barcode: "ZF-2", geb: datum(2012, 3, 3)})
	if _, err := pool.Exec(ctx, `UPDATE schueler SET lusd_bestaetigt_am = NOW(), strasse = 'Neuweg', plz = '61381', ort = 'Friedrichsdorf' WHERE id = $1`, quelle); err != nil {
		t.Fatal(err)
	}
	seedOffeneAusleihe(t, pool, quelle, "ZFQ")
	// Protokollspur der Quelle (Lesehistorie) — muss danach auf das Ziel zeigen.
	if _, err := pool.Exec(ctx, `INSERT INTO audit_log (tabelle, aktion, datensatz_id, akteur, details)
		VALUES ('ausleihen', 'CREATE', gen_random_uuid(), 'USER', jsonb_build_object('schueler_id', $1::text, 'entleiher', 'Neu Name'))`, quelle); err != nil {
		t.Fatal(err)
	}

	erg, err := repository.ZusammenfuehrenSchueler(ctx, pool, zfAuftrag(ziel, quelle))
	if err != nil {
		t.Fatalf("Zusammenführen: %v", err)
	}
	if erg.ZielID != ziel || erg.BarcodeID != "ZF-1" || erg.QuelleBarcode != "ZF-2" || erg.Ausleihen != 1 || erg.Nachname != "Name" || erg.Vorname != "Neu" {
		t.Errorf("Ergebnis falsch: %+v", erg)
	}

	if n := zfZaehle(t, pool, `SELECT count(*) FROM schueler WHERE id = $1`, quelle); n != 0 {
		t.Error("die Quelle existiert noch")
	}
	var vorname, klasse, strasse, grund string
	var abg, gesperrt, seit bool
	if err := pool.QueryRow(ctx, `SELECT vorname, klasse, COALESCE(strasse,''), ist_abgaenger, ist_gesperrt, COALESCE(block_reason,''), abgaenger_seit IS NOT NULL
		FROM schueler WHERE id = $1`, ziel).Scan(&vorname, &klasse, &strasse, &abg, &gesperrt, &grund, &seit); err != nil {
		t.Fatal(err)
	}
	// LUSD führend: Name, Klasse und Anschrift der frischer bestätigten Quelle.
	if vorname != "Neu" || !klassenGleich(klasse, "08A") || strasse != "Neuweg" {
		t.Errorf("Stammdaten nicht von der LUSD-frischen Seite: vorname=%q klasse=%q strasse=%q", vorname, klasse, strasse)
	}
	// Kein Abgänger mehr; die Sperre bleibt wegen der offenen Ausleihe — mit sachlichem Grund.
	if abg || seit || !gesperrt || grund != "Sperre wegen offener Vorgänge" {
		t.Errorf("Status falsch: abg=%v seit=%v gesperrt=%v grund=%q", abg, seit, gesperrt, grund)
	}
	if n := zfZaehle(t, pool, `SELECT count(*) FROM ausleihen WHERE schueler_id = $1 AND rueckgabe_am IS NULL`, ziel); n != 1 {
		t.Errorf("offene Ausleihe nicht gewandert (n=%d)", n)
	}
	if n := zfZaehle(t, pool, `SELECT count(*) FROM audit_log WHERE tabelle='ausleihen' AND details->>'schueler_id' = $1`, ziel); n != 1 {
		t.Errorf("Protokollspur zeigt nicht auf das Ziel (n=%d)", n)
	}
}

// Ohne offene Vorgänge fällt die automatische Abgänger-Sperre des Ziels; eine manuelle
// Sperre bliebe (anderer Grund) — hier der automatische Fall.
func TestZusammenfuehren_HebtAbgaengerSperreOhneVorgaengeAuf(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	ziel := legeUmbSchuelerAn(t, pool, umbSchueler{vorname: "Alt", nachname: "Frei", klasse: "07A", barcode: "ZF-3", geb: datum(2012, 4, 4), abgaenger: true})
	quelle := legeUmbSchuelerAn(t, pool, umbSchueler{vorname: "Alt", nachname: "Frei-Neu", klasse: "08A", barcode: "ZF-4", geb: datum(2012, 4, 4)})

	if _, err := repository.ZusammenfuehrenSchueler(ctx, pool, zfAuftrag(ziel, quelle)); err != nil {
		t.Fatal(err)
	}
	var gesperrt bool
	var grund *string
	if err := pool.QueryRow(ctx, `SELECT ist_gesperrt, block_reason FROM schueler WHERE id = $1`, ziel).Scan(&gesperrt, &grund); err != nil {
		t.Fatal(err)
	}
	if gesperrt || grund != nil {
		t.Errorf("Ziel ohne Vorgänge muss entsperrt sein: gesperrt=%v grund=%v", gesperrt, grund)
	}
}

func TestZusammenfuehren_Abweisungen(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	a := legeUmbSchuelerAn(t, pool, umbSchueler{vorname: "A", nachname: "Eins", klasse: "07A", barcode: "ZF-5", geb: datum(2012, 5, 5)})
	anon := legeUmbSchuelerAn(t, pool, umbSchueler{vorname: "Abgänger", nachname: "Anonymisiert-x", klasse: "ABG", barcode: "ANON-x"})
	if _, err := pool.Exec(ctx, `UPDATE schueler SET anonymized_at = NOW() WHERE id = $1`, anon); err != nil {
		t.Fatal(err)
	}

	if _, err := repository.ZusammenfuehrenSchueler(ctx, pool, zfAuftrag(a, a)); !errors.Is(err, repository.ErrZusammenfuehrenGleich) {
		t.Errorf("gleicher Datensatz: %v", err)
	}
	if _, err := repository.ZusammenfuehrenSchueler(ctx, pool, zfAuftrag(a, anon)); !errors.Is(err, repository.ErrZusammenfuehrenAnonymisiert) {
		t.Errorf("anonymisierte Quelle: %v", err)
	}
	if _, err := repository.ZusammenfuehrenSchueler(ctx, pool, zfAuftrag(a, "00000000-0000-0000-0000-000000000000")); !errors.Is(err, repository.ErrZusammenfuehrenNichtGefunden) {
		t.Errorf("unbekannte Quelle: %v", err)
	}
	if n := zfZaehle(t, pool, `SELECT count(*) FROM schueler WHERE deleted_at IS NULL`); n != 2 {
		t.Errorf("Abweisungen dürfen nichts verändern (n=%d)", n)
	}
}

// Die Kandidatensuche findet Abgänger und Gesperrte, nie den Ausgangsdatensatz selbst
// und nie Anonymisierte.
func TestZusammenfuehren_KandidatenSucheSiehtAbgaenger(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	ich := legeUmbSchuelerAn(t, pool, umbSchueler{vorname: "Mira", nachname: "Suchwort", klasse: "07A", barcode: "ZF-6", geb: datum(2012, 6, 6)})
	legeUmbSchuelerAn(t, pool, umbSchueler{vorname: "Mira", nachname: "Suchwort-Alt", klasse: "06A", barcode: "ZF-7", geb: datum(2012, 6, 6), abgaenger: true})
	anon := legeUmbSchuelerAn(t, pool, umbSchueler{vorname: "Mira", nachname: "Suchwort-Anon", klasse: "ABG", barcode: "ZF-8"})
	if _, err := pool.Exec(ctx, `UPDATE schueler SET anonymized_at = NOW() WHERE id = $1`, anon); err != nil {
		t.Fatal(err)
	}

	treffer, err := repository.SucheZusammenfuehrenKandidaten(ctx, pool, ich, "Suchwort", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(treffer) != 1 || treffer[0].Nachname != "Suchwort-Alt" || !treffer[0].IstAbgaenger || !treffer[0].IstGesperrt {
		t.Errorf("erwartet genau den gesperrten Abgänger: %+v", treffer)
	}
}
