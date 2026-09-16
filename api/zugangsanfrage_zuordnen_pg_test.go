package api

import (
	"context"
	"testing"

	"bibliothek/repository"
)

// „Das ist dieselbe Person" — der Weg, der aus zwei Einträgen einen macht.
//
// Der Anlass (16.09.2026): Ein Kollege, der vor der Schul-E-Mail-Pflicht von Hand in die
// Leserdatei eingetragen wurde, meldet sich über „Mein Portal" selbst an. Sein Konto
// entsteht OHNE leser_id (auth/selbstanmeldung.go), der Wächter trg_benutzer_hat_leserzeile
// hängt eine frische Leserzeile daran — mit dem aus der Adresse GERATENEN Namen. Ausweis
// und Bücher hängen am ersten Eintrag, die Anmeldung am zweiten.
//
// Der Rat der Oberfläche lautete bis heute „Anfrage löschen und die Adresse am
// vorhandenen Eintrag nachtragen". Er ist erzwungen — benutzer_email_unique gibt die
// Adresse nicht frei, solange das Anfrage-Konto sie hält — und er hinterlässt die
// Leserzeile des Wächters als Waise: ohne Konto, ohne Ausweis, mit dem geratenen Namen.
//
// Zugeordnet wird deshalb, statt zu löschen: Das vorhandene Konto zieht auf die
// vorhandene Leserzeile um, die Zeile des Wächters geht darin auf. Das Verfahren dafür
// gibt es bereits (repository.ZusammenfuehrenSchueler); dieser Test hält fest, dass es
// den Weg der Selbstanmeldung wirklich trägt — die anderen Zusammenführungs-Tests legen
// ihre Leserzeilen von Hand an und gehen am Wächter vorbei.
func TestZugangsanfrageZuordnen(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	// legeAltbestandAn ist der Kollege, der schon dasteht: Ausweis, ein Buch, KEIN Konto.
	legeAltbestandAn := func(t *testing.T, vorname, nachname, barcode, praefix string) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx,
			`INSERT INTO leser (vorname, nachname, art, barcode_id) VALUES ($1,$2,'lehrkraft',$3) RETURNING id`,
			vorname, nachname, barcode).Scan(&id); err != nil {
			t.Fatalf("Altbestand anlegen: %v", err)
		}
		seedOffeneAusleihe(t, pool, id, praefix)
		return id
	}

	// legeZugangsanfrageAn geht den Weg der Selbstanmeldung: INSERT OHNE leser_id, damit
	// der Wächter die zweite Zeile anlegt. Der Name ist der geratene — „anna.berger@"
	// wird zu „Anna Berger", auch wenn die Person in der Leserdatei anders geschrieben
	// steht. Genau dieser Name darf den richtigen nachher NICHT überschreiben.
	legeZugangsanfrageAn := func(t *testing.T, vorname, nachname, email string) (kontoID, waiseID string) {
		t.Helper()
		if err := pool.QueryRow(ctx, `
			INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv, zugang_beantragt_am)
			VALUES ($1, $2, $3, 'kollegium', false, CURRENT_TIMESTAMP)
			RETURNING id::text, leser_id::text`,
			vorname, nachname, email).Scan(&kontoID, &waiseID); err != nil {
			t.Fatalf("Zugangsanfrage anlegen: %v", err)
		}
		// Ohne diese Prüfung liefe der Test auch dann grün, wenn der Wächter gar nicht
		// mehr anspringt — dann gäbe es keine zweite Zeile, und er misst nichts.
		if waiseID == "" {
			t.Fatal("der Wächter hat keine Leserzeile angelegt — die Annahme dieses Tests stimmt nicht mehr")
		}
		return kontoID, waiseID
	}

	t.Run("das Konto zieht um, die Zeile des Wächters geht auf, der Bestand bleibt", func(t *testing.T) {
		// Die Leserdatei schreibt ihn „Bergér" — die Adresse kennt das Zeichen nicht.
		ziel := legeAltbestandAn(t, "Anna", "Bergér", "ZO-ALT-1", "ZO1")
		konto, waise := legeZugangsanfrageAn(t, "Anna", "Berger", "anna.berger@schule.invalid")

		erg, err := repository.ZusammenfuehrenSchueler(ctx, pool, zfAuftrag(ziel, waise))
		if err != nil {
			t.Fatalf("zuordnen: %v", err)
		}
		if erg.ZielID != ziel {
			t.Fatalf("Ziel ist %q, erwartet den vorhandenen Eintrag %q", erg.ZielID, ziel)
		}

		// 1. Die Waise ist weg — kein zweiter Eintrag bleibt zurück.
		if n := zfZaehle(t, pool, `SELECT count(*) FROM leser WHERE id = $1`, waise); n != 0 {
			t.Errorf("die Zeile des Wächters steht noch da (%d)", n)
		}
		// 2. Das Konto hängt jetzt am vorhandenen Eintrag — sonst meldete sich die Person
		//    an und sähe ihre Bücher nicht.
		var kontoLeser, kontoMail string
		if err := pool.QueryRow(ctx,
			`SELECT COALESCE(leser_id::text,''), email FROM benutzer WHERE id = $1`, konto).
			Scan(&kontoLeser, &kontoMail); err != nil {
			t.Fatalf("Konto lesen: %v", err)
		}
		if kontoLeser != ziel {
			t.Errorf("das Konto zeigt auf %q, erwartet %q", kontoLeser, ziel)
		}
		// 3. Die Adresse ist die echte geblieben: An ihr erkennt die Anmeldung die Person.
		if kontoMail != "anna.berger@schule.invalid" {
			t.Errorf("die Adresse am Konto ist %q", kontoMail)
		}
		// 4. Ausweis, Name und Buch des vorhandenen Eintrags sind unberührt. Der geratene
		//    Name darf den richtigen nicht überschreiben (fuehrend(): ein Kollege hat nie
		//    ein lusd_bestaetigt_am, also bleibt das Ziel führend).
		var vorname, nachname, ausweis string
		if err := pool.QueryRow(ctx,
			`SELECT vorname, nachname, COALESCE(barcode_id,'') FROM leser WHERE id = $1`, ziel).
			Scan(&vorname, &nachname, &ausweis); err != nil {
			t.Fatalf("Ziel lesen: %v", err)
		}
		if nachname != "Bergér" {
			t.Errorf("der Nachname ist %q — der aus der Adresse geratene hat den richtigen überschrieben", nachname)
		}
		if ausweis != "ZO-ALT-1" {
			t.Errorf("der Ausweis ist %q, erwartet ZO-ALT-1", ausweis)
		}
		if n := zfZaehle(t, pool,
			`SELECT count(*) FROM ausleihen WHERE schueler_id = $1 AND rueckgabe_am IS NULL`, ziel); n != 1 {
			t.Errorf("das ausgeliehene Buch hängt nicht mehr am Eintrag (%d)", n)
		}
		// 5. Und am Ende steht die Person genau einmal da.
		if n := zfZaehle(t, pool,
			`SELECT count(*) FROM leser WHERE nachname IN ('Bergér','Berger') AND deleted_at IS NULL`); n != 1 {
			t.Errorf("die Person steht %d mal in der Leserdatei", n)
		}
	})

	// Die Gegenprobe zur Richtung. Sie sagt NICHT, dass die verkehrte Richtung verboten
	// wäre — sie belegt, dass die Prüfungen oben etwas messen: Zusammenführen löscht die
	// QUELLE. Zeigt der Knopf in die falsche Richtung, verschwindet der Eintrag mit
	// Ausweis und Büchern, und der aus der Adresse geratene Name bleibt übrig. Still und
	// nur über den Rückweg-Eintrag im Protokoll zu reparieren.
	t.Run("die Richtung entscheidet, wer überlebt", func(t *testing.T) {
		alt := legeAltbestandAn(t, "Jan", "Küpper", "ZO-ALT-2", "ZO2")
		_, waise := legeZugangsanfrageAn(t, "Jan", "Kuepper", "jan.kuepper@schule.invalid")

		// Verkehrt herum: die Zeile des Wächters als Ziel.
		if _, err := repository.ZusammenfuehrenSchueler(ctx, pool, zfAuftrag(waise, alt)); err != nil {
			t.Fatalf("verkehrte Richtung: %v", err)
		}
		if n := zfZaehle(t, pool, `SELECT count(*) FROM leser WHERE id = $1`, alt); n != 0 {
			t.Fatal("die Gegenprobe misst nicht: der vorhandene Eintrag steht noch da")
		}
		var nachname, ausweis string
		if err := pool.QueryRow(ctx,
			`SELECT nachname, COALESCE(barcode_id,'') FROM leser WHERE id = $1`, waise).
			Scan(&nachname, &ausweis); err != nil {
			t.Fatalf("übrig gebliebene Zeile lesen: %v", err)
		}
		if nachname != "Kuepper" {
			t.Fatalf("die Gegenprobe misst nicht: übrig blieb der Name %q", nachname)
		}
		// Der Ausweis wandert mit (er ist ein Feld der Quelle, das im leeren Ziel
		// auffüllt) — die Person aber steht jetzt unter dem geratenen Namen da.
		_ = ausweis
	})
}

// Der zweite Fall des Knopfes: Die Person wurde aus Littera übernommen. Ihr Eintrag hat
// dort bereits ein Konto — mit einer Ersatzadresse unter der Platzhalter-Domäne, weil die
// Übernahme für jedes Konto eine Adresse braucht und die Littera-Leserdatei keine führt.
//
// Bis zum 16.09.2026 brach das Zusammenführen hier ab („beide Datensätze haben ein
// eigenes Zugangskonto"), und der Abbruch war an dieser Stelle falsch: Über eine Adresse
// unter .invalid meldet sich niemand an — die Anmeldung läuft über IMAP gegen den
// Schul-Mailserver. Es standen zwei Konten da, aber nur ein Zugang.
func TestZugangsanfrageZuordnen_LitteraPlatzhalterWeicht(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	// Der übernommene Eintrag: Ausweis, Buch und ein Konto mit Ersatzadresse.
	var ziel, platzhalterKonto string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Ruth', 'Övermann', 'littera-4711@littera.invalid', 'kollegium', true)
		RETURNING id::text, leser_id::text`).Scan(&platzhalterKonto, &ziel); err != nil {
		t.Fatalf("übernommenes Konto anlegen: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`UPDATE leser SET barcode_id = 'ZO-LIT-1' WHERE id = $1`, ziel); err != nil {
		t.Fatalf("Ausweis nachtragen: %v", err)
	}
	seedOffeneAusleihe(t, pool, ziel, "ZOL")

	// Dieselbe Person meldet sich selbst an — zweites Konto, zweite Leserzeile.
	var echtesKonto, waise string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv, zugang_beantragt_am)
		VALUES ('Ruth', 'Oevermann', 'ruth.oevermann@schule.invalid', 'kollegium', false, CURRENT_TIMESTAMP)
		RETURNING id::text, leser_id::text`).Scan(&echtesKonto, &waise); err != nil {
		t.Fatalf("Zugangsanfrage anlegen: %v", err)
	}

	if _, err := repository.ZusammenfuehrenSchueler(ctx, pool, zfAuftrag(ziel, waise)); err != nil {
		t.Fatalf("zuordnen: %v", err)
	}

	// Das Platzhalter-Konto ist weg, das echte hängt am übernommenen Eintrag.
	if n := zfZaehle(t, pool, `SELECT count(*) FROM benutzer WHERE id = $1`, platzhalterKonto); n != 0 {
		t.Errorf("das Platzhalter-Konto steht noch da (%d)", n)
	}
	var kontoLeser string
	if err := pool.QueryRow(ctx,
		`SELECT COALESCE(leser_id::text,'') FROM benutzer WHERE id = $1`, echtesKonto).Scan(&kontoLeser); err != nil {
		t.Fatalf("echtes Konto lesen: %v", err)
	}
	if kontoLeser != ziel {
		t.Errorf("das echte Konto zeigt auf %q, erwartet %q", kontoLeser, ziel)
	}
	// Ausweis und Buch des übernommenen Eintrags sind unberührt geblieben.
	var ausweis string
	if err := pool.QueryRow(ctx,
		`SELECT COALESCE(barcode_id,'') FROM leser WHERE id = $1`, ziel).Scan(&ausweis); err != nil {
		t.Fatalf("Ziel lesen: %v", err)
	}
	if ausweis != "ZO-LIT-1" {
		t.Errorf("der Ausweis ist %q, erwartet ZO-LIT-1", ausweis)
	}
	if n := zfZaehle(t, pool,
		`SELECT count(*) FROM ausleihen WHERE schueler_id = $1 AND rueckgabe_am IS NULL`, ziel); n != 1 {
		t.Errorf("das Buch hängt nicht mehr am Eintrag (%d)", n)
	}
	// Und der Rückweg weiß, dass hier ein Konto gelöscht wurde — es ist das einzige, was
	// der Vorgang löscht statt umzuhängen.
	if n := zfZaehle(t, pool, `SELECT count(*) FROM audit_log
		 WHERE tabelle = 'schueler' AND aktion = 'ZUSAMMENGEFUEHRT' AND datensatz_id = $1::uuid
		   AND details->'gewandert'->'platzhalter_konten' @> to_jsonb($2::text)`, ziel, platzhalterKonto); n != 1 {
		t.Errorf("das gelöschte Platzhalter-Konto steht nicht im Rückweg-Eintrag (%d)", n)
	}
}

// Die Grenze der Ausnahme: ZWEI echte Konten bleiben eine Entscheidung über zwei
// Menschen. Ohne diese Probe wäre nicht belegt, dass die Räumung eng bleibt — sie könnte
// still jedes zweite Konto abräumen.
func TestZugangsanfrageZuordnen_ZweiEchteKontenBrechenWeiterAb(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	var ziel, quelle string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Eva', 'Zwei', 'eva.zwei@schule.invalid', 'kollegium', true)
		RETURNING leser_id::text`).Scan(&ziel); err != nil {
		t.Fatalf("erstes Konto: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Eva', 'Zwei', 'e.zwei@schule.invalid', 'kollegium', true)
		RETURNING leser_id::text`).Scan(&quelle); err != nil {
		t.Fatalf("zweites Konto: %v", err)
	}

	if _, err := repository.ZusammenfuehrenSchueler(ctx, pool, zfAuftrag(ziel, quelle)); err == nil {
		t.Fatal("zwei echte Konten wurden zusammengeführt — eine der beiden Anmeldungen ist verloren")
	}
	// Nichts darf angetastet sein.
	for _, id := range []string{ziel, quelle} {
		if n := zfZaehle(t, pool, `SELECT count(*) FROM benutzer WHERE leser_id = $1`, id); n != 1 {
			t.Errorf("das Konto an %s wurde trotz Abbruch angetastet (%d)", id, n)
		}
	}
}
