package repository

// betriebszustand.go — Abfragen, die die Selbstprüfung (api/betriebsbereitschaft.go)
// braucht, um den Zustand der Anlage zu beurteilen.
//
// Eigene Datei statt einer Zeile im Handler: Handler lesen und schreiben über repository/,
// dort steht jede Regel genau einmal. Das ist im Projekt eine geprüfte Invariante
// (api/schichtung_test.go) — und sie hat mich hier zu Recht erwischt.

import (
	"context"
	"time"

	"bibliothek/db"
)

// demoBarcodePraefix ist das Präfix, das scripts/seed_demo.sql jedem angelegten Schüler
// gibt. Es ist zugleich der Schlüssel, an dem der CLEANUP-Block desselben Skripts die
// Demo-Daten wieder entfernt — beide Seiten müssen dasselbe meinen.
const demoBarcodePraefix = "DEMO-S-%"

// demoExemplarPraefix ist das Gegenstück für die Exemplare desselben Skripts. Bis zum
// 07.09.2026 zählte die Selbstprüfung nur Schüler — wer den DELETE-Block halb ausführte,
// stand mit 300 Demo-Exemplaren im Bestandsbericht und einer grünen Selbstprüfung da.
const demoExemplarPraefix = "DEMO-B-%"

// Kennungen der drei BEISPIEL-Lieferanten, die db/seed.go vom 30.05. bis zum 07.09.2026
// beim ersten Start anlegte — gewollte Startdaten der ersten Bauwoche, damit das
// Bestellwesen ohne Vorarbeit bedienbar war (Migration 107 löscht das exakte Tripel).
// Adresse und Kundennummer einzeln, nicht das Tripel: Ein umbenannter Eintrag mit der
// Beispiel-Adresse schickt Bestellungen genauso ins Leere wie das Original.
var (
	beispielLieferantenEmails = []string{"bestellung@klett.de", "service@cornelsen.de", "order@westermann.de"}
	beispielKundennummern     = []string{"K-99281", "C-88123", "W-77441"}
)

// BetriebszustandRepository beantwortet Fragen über den Zustand des Bestandes.
type BetriebszustandRepository struct {
	pool db.PgxPoolIface
}

// NewBetriebszustandRepository bindet das Repository an einen Pool.
func NewBetriebszustandRepository(pool db.PgxPoolIface) *BetriebszustandRepository {
	return &BetriebszustandRepository{pool: pool}
}

// ZaehleDemoSchueler liefert die Anzahl der Datensätze aus scripts/seed_demo.sql.
//
// Eigenes kurzes Zeitlimit: Die Selbstprüfung ist eine Auskunft, kein Arbeitsschritt — sie
// darf keine Seite blockieren. Bleibt die Antwort aus, meldet der Aufrufer lieber „keine
// Demo-Daten gefunden" als gar nichts; der Bereich ist ohnehin nur eine Warnung.
func (r *BetriebszustandRepository) ZaehleDemoSchueler(ctx context.Context) (int, error) {
	ctx, abbrechen := context.WithTimeout(ctx, 3*time.Second)
	defer abbrechen()

	var anzahl int
	err := r.pool.QueryRow(ctx,
		`SELECT count(*) FROM schueler WHERE barcode_id LIKE $1`, demoBarcodePraefix).Scan(&anzahl)
	return anzahl, err
}

// ZaehleDemoExemplare liefert die Anzahl der Exemplare aus scripts/seed_demo.sql.
func (r *BetriebszustandRepository) ZaehleDemoExemplare(ctx context.Context) (int, error) {
	ctx, abbrechen := context.WithTimeout(ctx, 3*time.Second)
	defer abbrechen()

	var anzahl int
	err := r.pool.QueryRow(ctx,
		`SELECT count(*) FROM buecher_exemplare WHERE barcode_id LIKE $1`, demoExemplarPraefix).Scan(&anzahl)
	return anzahl, err
}

// BeispielLieferanten nennt die Lieferanten, die noch eine Kennung der alten
// Startdaten tragen. Leer = keiner; ein Fehler heißt „nicht erhoben" — der
// Aufrufer unterscheidet das von der leeren Liste, damit ein Datenbankfehler nicht als
// „alles gut" durchgeht.
func (r *BetriebszustandRepository) BeispielLieferanten(ctx context.Context) ([]string, error) {
	ctx, abbrechen := context.WithTimeout(ctx, 3*time.Second)
	defer abbrechen()

	rows, err := r.pool.Query(ctx,
		`SELECT name FROM lieferanten
		  WHERE lower(email) = ANY($1) OR kundennummer = ANY($2)
		  ORDER BY name`, beispielLieferantenEmails, beispielKundennummern)
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

// AdminKonto ist ein aktives Konto mit Vollzugriff — Empfänger der Alarm-Mails
// und Gegenstand des Sichtbarkeits-Befunds der Selbstprüfung.
type AdminKonto struct {
	Name  string
	Email string
}

// AktiveAdmins liefert alle aktiven Admin-Konten (Name + Adresse) — EINE Quelle
// für den Alarm-Versand UND den Betriebsbereitschafts-Befund. Der Vorfall vom
// 16.08.2026 (Alarm-Mail an ein dem Betreiber unbekanntes Admin-Konto) hat
// gezeigt: Die Admin-Liste muss sichtbar sein, nicht nur benutzt werden.
func (r *BetriebszustandRepository) AktiveAdmins(ctx context.Context) ([]AdminKonto, error) {
	ctx, abbrechen := context.WithTimeout(ctx, 3*time.Second)
	defer abbrechen()

	rows, err := r.pool.Query(ctx, `
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

// LadeRollenRechte liefert die Live-Rechte (role_permissions) als Rolle→Recht→erlaubt.
// Für den Vorgabe-Abgleich der Selbstprüfung; gleiche Zeitlimit-Begründung wie oben.
func (r *BetriebszustandRepository) LadeRollenRechte(ctx context.Context) (map[string]map[string]bool, error) {
	ctx, abbrechen := context.WithTimeout(ctx, 3*time.Second)
	defer abbrechen()

	rows, err := r.pool.Query(ctx, `SELECT role, permission, allowed FROM role_permissions`)
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

// KlassenBestand liefert die drei Mengen von Klassennamen, die nur über
// Text-Gleichheit zusammenhängen (Befund F3): die Klassen der aktiven Schüler
// (nur echte Stufen-Namen, keine Sonderwerte wie 'ABG'), die Zeilen der
// Klassenlehrer-Zuordnung und die Klassen der LMF-Bücherlisten. Die
// Selbstprüfung rechnet daraus den Drift — hier wird nur erhoben.
func (r *BetriebszustandRepository) KlassenBestand(ctx context.Context) (schueler, zuordnungen, buecherlisten []string, err error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	lies := func(query string) ([]string, error) {
		rows, err := r.pool.Query(ctx, query)
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
		SELECT DISTINCT klasse FROM schueler
		WHERE deleted_at IS NULL AND ist_abgaenger = false AND klasse ~ '^\d'
		ORDER BY klasse`); err != nil {
		return nil, nil, nil, err
	}
	if zuordnungen, err = lies(`SELECT klasse FROM klassen_lehrer_mapping ORDER BY klasse`); err != nil {
		return nil, nil, nil, err
	}
	if buecherlisten, err = lies(`SELECT DISTINCT class_name FROM class_books ORDER BY class_name`); err != nil {
		return nil, nil, nil, err
	}
	return schueler, zuordnungen, buecherlisten, nil
}

// LadeEinstellungswert liest einen rohen Wert aus system_einstellungen — z. B. das
// JSON-Ergebnis der wöchentlichen Restore-Probe (jobs.RestoreProbeSchluessel).
// pgx.ErrNoRows wird durchgereicht; der Aufrufer entscheidet, was „nie geschrieben" heißt.
func (r *BetriebszustandRepository) LadeEinstellungswert(ctx context.Context, schluessel string) (string, error) {
	var wert string
	err := r.pool.QueryRow(ctx,
		`SELECT wert FROM system_einstellungen WHERE schluessel = $1`, schluessel).Scan(&wert)
	return wert, err
}

// ZaehleEhemaligeMitOffenenVorgaengen zählt Weggegangene (ist_abgaenger, nicht
// gelöscht), die seit mehr als `tage` Tagen weg sind und noch einen offenen Vorgang
// haben — ein nie zurückgegebenes Buch oder eine unbezahlte Forderung. Genau diese
// Vorgänge schützen den Datensatz vor Anonymisierung und Löschung (Retention-Blockade);
// schließt sie niemand, bleibt der Name mit Anschrift auf Dauer stehen, und keine
// Routine meldet es (Register 05.09.2026, Entscheidung 1: Befund statt Automatismus).
func (r *BetriebszustandRepository) ZaehleEhemaligeMitOffenenVorgaengen(ctx context.Context, tage int) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `
		SELECT count(*)
		FROM schueler s
		WHERE s.ist_abgaenger = true AND s.deleted_at IS NULL
		  AND s.abgaenger_seit IS NOT NULL
		  AND s.abgaenger_seit < now() - make_interval(days => $1)
		  AND (EXISTS (SELECT 1 FROM ausleihen a WHERE a.schueler_id = s.id AND a.rueckgabe_am IS NULL)
		    OR EXISTS (SELECT 1 FROM schadensfaelle d WHERE d.schueler_id = s.id AND d.ist_bezahlt = false))`, tage).Scan(&n)
	return n, err
}
