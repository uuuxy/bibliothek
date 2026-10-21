package repository

// Abfragen, mit denen die Selbstprüfung (internal/bereitschaft) den Zustand der Anlage
// beurteilt.

import (
	"context"
	"time"

	"bibliothek/db"
)

// demoBarcodePraefix ist die Vorsilbe, die scripts/seed_demo.sql jedem angelegten Schüler gibt.
// An ihr entfernt der Aufräum-Block desselben Skripts die Demo-Daten wieder; beide Seiten
// meinen dasselbe.
const demoBarcodePraefix = "DEMO-S-%"

// demoExemplarPraefix ist dasselbe für die Exemplare des Skripts: Wer dessen Aufräum-Block nur
// halb ausführt, hätte sonst Demo-Exemplare im Bestandsbericht und eine grüne Selbstprüfung.
const demoExemplarPraefix = "DEMO-B-%"

// Die Kennungen der drei Beispiel-Lieferanten aus früheren Startdaten. Adresse und
// Kundennummer zählen einzeln: Ein umbenannter Eintrag mit der Beispiel-Adresse schickt
// Bestellungen genauso ins Leere wie das Original.
var (
	beispielLieferantenEmails = []string{"bestellung@klett.de", "service@cornelsen.de", "order@westermann.de"}
	beispielKundennummern     = []string{"K-99281", "C-88123", "W-77441"}
)

// BetriebszustandAbfragen beantwortet Fragen über den Zustand der Anlage.
type BetriebszustandAbfragen struct {
	pool db.Pool
}

// NewBetriebszustandAbfragen liefert die Abfragen zum Betriebszustand am übergebenen Pool.
func NewBetriebszustandAbfragen(pool db.Pool) *BetriebszustandAbfragen {
	return &BetriebszustandAbfragen{pool: pool}
}

// ZaehleDemoSchueler zählt die Schüler aus scripts/seed_demo.sql.
//
// Die Abfrage hat eine eigene kurze Frist: Die Selbstprüfung ist eine Auskunft und hält
// keine Seite auf. Bleibt die Antwort aus, meldet der Rufer „keine Demo-Daten gefunden";
// der Bereich ist nur eine Warnung.
func (a *BetriebszustandAbfragen) ZaehleDemoSchueler(ctx context.Context) (int, error) {
	ctx, abbrechen := context.WithTimeout(ctx, 3*time.Second)
	defer abbrechen()

	var anzahl int
	err := a.pool.QueryRow(ctx,
		`SELECT count(*) FROM schueler WHERE barcode_id LIKE $1`, demoBarcodePraefix).Scan(&anzahl)
	return anzahl, err
}

// ZaehleDemoExemplare zählt die Exemplare aus scripts/seed_demo.sql.
func (a *BetriebszustandAbfragen) ZaehleDemoExemplare(ctx context.Context) (int, error) {
	ctx, abbrechen := context.WithTimeout(ctx, 3*time.Second)
	defer abbrechen()

	var anzahl int
	err := a.pool.QueryRow(ctx,
		`SELECT count(*) FROM buecher_exemplare WHERE barcode_id LIKE $1`, demoExemplarPraefix).Scan(&anzahl)
	return anzahl, err
}

// LadeBeispielLieferanten nennt die Lieferanten, die noch eine Kennung der früheren Startdaten
// tragen. Leer heißt keiner, ein Fehler heißt „nicht erhoben"; der Rufer unterscheidet
// beides, damit ein Fehler der Datenbank nicht als „alles gut" gilt.
func (a *BetriebszustandAbfragen) LadeBeispielLieferanten(ctx context.Context) ([]string, error) {
	ctx, abbrechen := context.WithTimeout(ctx, 3*time.Second)
	defer abbrechen()

	rows, err := a.pool.Query(ctx,
		`SELECT name FROM lieferanten
		  WHERE lower(email) = ANY($1) OR kundennummer = ANY($2)
		  ORDER BY name COLLATE deutsch`, beispielLieferantenEmails, beispielKundennummern)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	namen := []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		namen = append(namen, name)
	}
	return namen, rows.Err()
}

// AdminKonto ist ein aktives Konto mit vollem Zugriff: Es bekommt die Alarm-Mails, und die
// Selbstprüfung zeigt es.
type AdminKonto struct {
	Name  string
	Email string
}

// LadeAktiveAdmins liefert alle aktiven Admin-Konten mit Name und Adresse, für den Versand der
// Alarme und für die Selbstprüfung: Wer die Alarm-Mails bekommt, ist dort auch zu sehen.
func (a *BetriebszustandAbfragen) LadeAktiveAdmins(ctx context.Context) ([]AdminKonto, error) {
	ctx, abbrechen := context.WithTimeout(ctx, 3*time.Second)
	defer abbrechen()

	rows, err := a.pool.Query(ctx, `
		SELECT btrim(vorname || ' ' || nachname), email
		FROM benutzer WHERE rolle = 'admin' AND aktiv = true AND email <> ''
		ORDER BY erstellt_am`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	konten := []AdminKonto{}
	for rows.Next() {
		var k AdminKonto
		if err := rows.Scan(&k.Name, &k.Email); err != nil {
			return nil, err
		}
		konten = append(konten, k)
	}
	return konten, rows.Err()
}

// LadeRollenRechte liefert die geltenden Rechte (role_permissions) je Rolle und Recht, für
// den Abgleich der Selbstprüfung mit der Vorgabe.
func (a *BetriebszustandAbfragen) LadeRollenRechte(ctx context.Context) (map[string]map[string]bool, error) {
	ctx, abbrechen := context.WithTimeout(ctx, 3*time.Second)
	defer abbrechen()

	rows, err := a.pool.Query(ctx, `SELECT role, permission, allowed FROM role_permissions`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	live := map[string]map[string]bool{}
	for rows.Next() {
		var rolle, recht string
		var erlaubt bool
		if err := rows.Scan(&rolle, &recht, &erlaubt); err != nil {
			return nil, err
		}
		if live[rolle] == nil {
			live[rolle] = map[string]bool{}
		}
		live[rolle][recht] = erlaubt
	}
	return live, rows.Err()
}

// LadeKlassenBestand liefert die drei Mengen von Klassennamen, die nur über gleichen Text
// zusammenhängen: die Klassen der aktiven Schüler, die Zeilen der Zuordnung der Klassenlehrer
// und die Klassen der Bücherlisten. Welche Klasse eine Klassenleitung braucht, entscheidet
// die Selbstprüfung; hier wird nur erhoben.
func (a *BetriebszustandAbfragen) LadeKlassenBestand(ctx context.Context) (schueler, zuordnungen, buecherlisten []string, err error) {
	ctx, abbrechen := context.WithTimeout(ctx, 3*time.Second)
	defer abbrechen()

	lies := func(query string) ([]string, error) {
		rows, err := a.pool.Query(ctx, query)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		werte := []string{}
		for rows.Next() {
			var w string
			if err := rows.Scan(&w); err != nil {
				return nil, err
			}
			werte = append(werte, w)
		}
		return werte, rows.Err()
	}

	if schueler, err = lies(`
		SELECT klasse FROM schueler
		WHERE deleted_at IS NULL AND ist_abgaenger = false AND btrim(klasse) <> ''
		GROUP BY klasse
		ORDER BY klasse COLLATE deutsch`); err != nil {
		return nil, nil, nil, err
	}
	if zuordnungen, err = lies(`SELECT klasse FROM klassen_lehrer_mapping ORDER BY klasse COLLATE deutsch`); err != nil {
		return nil, nil, nil, err
	}
	if buecherlisten, err = lies(`SELECT class_name FROM class_books GROUP BY class_name ORDER BY class_name COLLATE deutsch`); err != nil {
		return nil, nil, nil, err
	}
	return schueler, zuordnungen, buecherlisten, nil
}

// LadeEinstellungswert liest einen rohen Wert aus system_einstellungen, etwa das Ergebnis der
// wöchentlichen Restore-Probe als JSON (jobs.RestoreProbeSchluessel). pgx.ErrNoRows kommt
// unverändert zurück; der Rufer entscheidet, was „nie geschrieben" heißt.
func (a *BetriebszustandAbfragen) LadeEinstellungswert(ctx context.Context, schluessel string) (string, error) {
	var wert string
	err := a.pool.QueryRow(ctx,
		`SELECT wert FROM system_einstellungen WHERE schluessel = $1`, schluessel).Scan(&wert)
	return wert, err
}

// SchreibeEinstellungswert legt einen rohen Wert unter seinem Schlüssel ab oder ersetzt ihn:
// das Gegenstück zu LadeEinstellungswert, für Ergebnisse, die das Programm selbst schreibt.
func (a *BetriebszustandAbfragen) SchreibeEinstellungswert(ctx context.Context, schluessel, wert string) error {
	_, err := a.pool.Exec(ctx, `
		INSERT INTO system_einstellungen (schluessel, wert) VALUES ($1, $2)
		ON CONFLICT (schluessel) DO UPDATE SET wert = EXCLUDED.wert, aktualisiert_am = CURRENT_TIMESTAMP`,
		schluessel, wert)
	return err
}

// ZaehleEhemaligeMitOffenenVorgaengen zählt Weggegangene (ist_abgaenger, nicht gelöscht), die
// seit mehr als `tage` Tagen weg sind und noch ein offenes Buch oder eine unbezahlte Forderung
// haben. Der offene Vorgang schützt den Datensatz vor Anonymisierung und Löschung; schließt
// ihn niemand, bleiben Name und Anschrift auf Dauer stehen, ohne dass eine Routine es meldet.
func (a *BetriebszustandAbfragen) ZaehleEhemaligeMitOffenenVorgaengen(ctx context.Context, tage int) (int, error) {
	var n int
	// Dieselbe Uhr wie beim Löschen (sqlKarenzUhr: Abgang, letzte Rückgabe, letzter Abschluss
	// eines Schadens). Ein Abgänger ohne Stempel bliebe sonst unsichtbar, weil der offene
	// Vorgang ihn zugleich vor dem Löschen schützt; und wer vor zehn Tagen an der Theke war,
	// ist noch kein Fall.
	err := a.pool.QueryRow(ctx, `
		SELECT count(*)
		FROM schueler s
		WHERE s.ist_abgaenger = true AND s.deleted_at IS NULL
		  AND `+sqlKarenzUhr("s")+` < now() - make_interval(days => $1)
		  AND (EXISTS (SELECT 1 FROM ausleihen a WHERE a.schueler_id = s.id AND a.rueckgabe_am IS NULL)
		    OR EXISTS (SELECT 1 FROM schadensfaelle d WHERE d.schueler_id = s.id AND d.ist_bezahlt = false))`, tage).Scan(&n)
	return n, err
}

// ZaehleNachbuchMeldungenOffenSeit zählt die Meldungen aus dem Nachbuchen, die seit mehr als
// `tage` Tagen niemand quittiert hat. Eine offene Meldung ist eine Abweichung zwischen dem
// Scan ohne Netz und der Wirklichkeit, die jemand ansehen muss: ein umgebuchtes Buch, eine
// abgewiesene Ausleihe. Bleibt sie liegen, stehen die Beteiligten mit Namen darin, ohne
// Frist; nur quittierte Meldungen haben eine.
func (a *BetriebszustandAbfragen) ZaehleNachbuchMeldungenOffenSeit(ctx context.Context, tage int) (int, error) {
	var n int
	err := a.pool.QueryRow(ctx, `
		SELECT count(*) FROM nachbuch_meldungen
		WHERE quittiert_am IS NULL AND erstellt_am < now() - make_interval(days => $1)`, tage).Scan(&n)
	return n, err
}
