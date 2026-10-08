package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"bibliothek/pkg/closeutil"
	"bibliothek/pkg/schulzeit"

	"github.com/jackc/pgx/v5"
)

// Die Anweisungen des LUSD-Imports. Jede läuft in der Transaktion des Laufs; Reihenfolge und
// Regeln (wer adoptiert, angelegt, aktualisiert oder zum Abgänger wird) stehen beim Aufrufer.

// lusdImportSperre ist der Schlüssel der Sperre, die gleichzeitige Importe einreiht. Kein
// anderer Advisory-Schlüssel im Projekt nutzt diesen Nummernkreis.
const lusdImportSperre int64 = 750_2026

// SperreLusdImport reiht den Lauf hinter einem laufenden Import ein, bis die Transaktion
// endet. Zwei Importe zugleich arbeiteten auf sich überholenden Ständen: Kollidierende
// Ausweisnummern oder LUSD-IDs brächen den zweiten ab, und der eine könnte einen Abgänger
// anonymisieren, den der andere gerade wieder aktiviert.
func SperreLusdImport(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, lusdImportSperre)
	return err
}

// FindeAktivenSchuelerNachLusdID nennt die nicht gelöschte Schülerzeile zu einer LUSD-ID oder
// "", wenn es keine gibt; mehr als eine lässt der Teilindex uniq_schueler_lusd_id_active nicht
// zu. So findet der Lauf einen Rückkehrer, der nicht unter den Aktiven steht, dessen Zeile die
// LUSD-ID aber noch belegt.
func FindeAktivenSchuelerNachLusdID(ctx context.Context, tx pgx.Tx, lusdID string) (string, error) {
	var id string
	err := tx.QueryRow(ctx,
		"SELECT id FROM schueler WHERE lusd_id = $1 AND deleted_at IS NULL LIMIT 1", lusdID,
	).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return id, err
}

// AdoptiereLusdWaise heftet die LUSD-ID an einen Schüler, der noch keine echte trägt, und
// trägt ein fehlendes Geburtsdatum nach. Mit leerer LUSD-ID (Abgleich über den Namen) bleibt
// die ID unberührt.
//
// Die Bedingung schützt doppelt: `lusd_id IS NULL OR lusd_id LIKE 'littera:%'` hält gegen
// einen Wettlauf, in dem der Schüler inzwischen eine echte ID bekam (die Herkunftsmarke der
// Littera-Übernahme darf überschrieben werden). `NOT EXISTS` verhindert die Kollision am
// Teilindex uniq_schueler_lusd_id_active, falls die ID inzwischen eine andere aktive Zeile
// hält. Greift der Schutz, bleibt die Zeile unverändert und der Import läuft weiter; die
// Zeile der Datei nimmt dann den Weg des Rückkehrers oder des Neuzugangs.
func AdoptiereLusdWaise(ctx context.Context, tx pgx.Tx, schuelerID, lusdID, geburtsdatum string) error {
	var geb *string
	if geburtsdatum != "" {
		geb = &geburtsdatum
	}
	_, err := tx.Exec(ctx, `
			UPDATE schueler SET
				lusd_id = COALESCE(NULLIF($1, ''), lusd_id),
				geburtsdatum = COALESCE(geburtsdatum, $4::date),
				aktualisiert_am = NOW()
			WHERE id = $2 AND ($1 = '' OR lusd_id IS NULL OR lusd_id LIKE $3)
			  AND NOT EXISTS (SELECT 1 FROM schueler WHERE $1 <> '' AND lusd_id = $1 AND deleted_at IS NULL)`,
		lusdID, schuelerID, LitteraHerkunftPraefix+"%", geb)
	return err
}

// LusdNeuzugang ist ein Schüler, den der Import neu anlegt. Leere Texte werden als NULL
// gespeichert, auch die LUSD-ID: Ein Leerstring belegte den Teilindex
// uniq_schueler_lusd_id_active, und der zweite Neuzugang ohne ID bräche den Import ab.
type LusdNeuzugang struct {
	Ausweisnummer                              string
	Vorname, Nachname, Klasse                  string
	AbgaengerJahr                              int
	LusdID                                     string
	Geburtsdatum, EintrittAm                   *time.Time
	Strasse, Hausnummer, PLZ, Ort, ElternEmail string
}

// LegeLusdSchuelerAn legt den Neuzugang an. lusd_bestaetigt_am wird gesetzt: Der Schüler
// kommt aus dem Export und gilt damit als von der LUSD geführt.
func LegeLusdSchuelerAn(ctx context.Context, tx pgx.Tx, n LusdNeuzugang) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO schueler
			(barcode_id, vorname, nachname, klasse, abgaenger_jahr, lusd_id, geburtsdatum,
			 strasse, hausnummer, plz, ort, eltern_email, lusd_bestaetigt_am, schul_eintritt_am)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW(), $13)`,
		n.Ausweisnummer, n.Vorname, n.Nachname, n.Klasse, n.AbgaengerJahr,
		leererStringAlsNull(n.LusdID), n.Geburtsdatum,
		leererStringAlsNull(n.Strasse), leererStringAlsNull(n.Hausnummer), leererStringAlsNull(n.PLZ),
		leererStringAlsNull(n.Ort), leererStringAlsNull(n.ElternEmail), n.EintrittAm)
	return err
}

// LusdAktualisierung trägt, was der Export zu einem Schüler des Bestands sagt. Geburtsdatum
// ist nur bei einem bestätigten Umbenennungs-Paar gesetzt: Sonst könnte eine Korrektur aus der
// LUSD am Teilindex unique_schueler_name_gebdatum kollidieren und den ganzen Lauf abbrechen.
type LusdAktualisierung struct {
	SchuelerID                                 string
	Vorname, Nachname, Klasse                  string
	Strasse, Hausnummer, PLZ, Ort, ElternEmail string
	EintrittAm, Geburtsdatum                   *time.Time
}

// AktualisiereLusdBestand übernimmt Klasse, Name und Kontaktdaten aus dem Export, alle Zeilen
// als ein Batch. Ein leerer Wert des Exports überschreibt nichts (COALESCE(NULLIF(...))); ein
// Export ohne Adressspalten löscht also keine Adressen.
//
// Wer wieder im Export steht, ist kein Abgänger mehr: Der Name kommt immer aus dem Export, und
// der Abgänger-Stand wird zurückgesetzt. Ein anonymisierter Abgänger wird entsperrt, er hatte
// keine offenen Vorgänge. Eine automatische Sperre fällt, wenn weder eine Ausleihe noch ein
// unbezahlter Schaden offen ist; sonst bleibt sie mit dem Grund AbgaengerSperrgrundOffen, an
// dessen Präfix spätere Wege die Automatik erkennen. Eine Sperre aus anderem Grund bleibt
// unberührt. Die CASE-Ausdrücke lesen die alten Werte der Zeile, weil Postgres die rechte
// Seite vor der Zuweisung auswertet.
//
// Die Verbindung bleibt bis zum Close vom Batch belegt; der Aufrufer nutzt die Transaktion
// danach weiter. Close steht deshalb vor jedem Rückweg, und sein Fehler wird gemeldet: Er
// nennt, was beim Abräumen der übrigen Ergebnisse auffiel.
func AktualisiereLusdBestand(ctx context.Context, tx pgx.Tx, zeilen []LusdAktualisierung) error {
	batch := &pgx.Batch{}
	for _, z := range zeilen {
		batch.Queue(`
		UPDATE schueler SET
			vorname      = COALESCE(NULLIF($1, ''), vorname),
			nachname     = COALESCE(NULLIF($2, ''), nachname),
			schul_eintritt_am = COALESCE($10::date, schul_eintritt_am),
			geburtsdatum = COALESCE($11::date, geburtsdatum),
			-- Wie die sieben Felder ringsum: ein LEERER Exportwert darf den Bestand nicht
			-- loeschen. klasse stand hier als EINZIGES ungeschuetzt da (seit 4219a2e, nie
			-- bewusst entschieden). Dieselbe Bugklasse hat 96c2f8c im Buch-Importer bereits
			-- behoben (autor/verlag/jahr). Heute faengt lusd_parser.go:82 eine leere Klasse
			-- schon beim Parsen ab — aber diese Zeile ist die zweite Tuer zum selben Zustand,
			-- und die Klasse ist die Angabe, an der Ausleihlimit, Mahnweg und Abgaengerlauf
			-- haengen. Ein leerer Wert hier ist nie eine gewollte Aussage.
			klasse       = COALESCE(NULLIF($3, ''), klasse),
			strasse      = COALESCE(NULLIF($4, ''), strasse),
			hausnummer   = COALESCE(NULLIF($5, ''), hausnummer),
			plz          = COALESCE(NULLIF($6, ''), plz),
			ort          = COALESCE(NULLIF($7, ''), ort),
			eltern_email = COALESCE(NULLIF($8, ''), eltern_email),
			-- Im Export wiedergefunden: das Gedächtnis für den Namensmodus (Migration 084).
			lusd_bestaetigt_am = NOW(),
			ist_abgaenger = false,
			abgaenger_seit = NULL,
			ist_gesperrt = CASE
				WHEN vorname = 'Abgänger' AND nachname LIKE 'Anonymisiert-%' THEN false
				WHEN `+SQLAbgaengerSperreAutomatisch+`
				     AND NOT EXISTS (SELECT 1 FROM ausleihen WHERE schueler_id = $9 AND rueckgabe_am IS NULL)
				     AND NOT EXISTS (SELECT 1 FROM schadensfaelle WHERE schueler_id = $9 AND ist_bezahlt = false)
				THEN false
				ELSE ist_gesperrt END,
			-- block_reason konsistent zu ist_gesperrt setzen: chk_schueler_block_reason
			-- verlangt einen Grund NUR solange ist_gesperrt = true.
			block_reason = CASE
				WHEN vorname = 'Abgänger' AND nachname LIKE 'Anonymisiert-%' THEN NULL
				WHEN `+SQLAbgaengerSperreAutomatisch+`
				     AND NOT EXISTS (SELECT 1 FROM ausleihen WHERE schueler_id = $9 AND rueckgabe_am IS NULL)
				     AND NOT EXISTS (SELECT 1 FROM schadensfaelle WHERE schueler_id = $9 AND ist_bezahlt = false)
				THEN NULL
				WHEN `+SQLAbgaengerSperreAutomatisch+`
				THEN '`+AbgaengerSperrgrundOffen+`'
				ELSE block_reason END,
			aktualisiert_am = NOW()
		WHERE id = $9`,
			z.Vorname, z.Nachname, z.Klasse,
			z.Strasse, z.Hausnummer, z.PLZ, z.Ort, z.ElternEmail, z.SchuelerID,
			z.EintrittAm, z.Geburtsdatum)
	}

	br := tx.SendBatch(ctx, batch)

	for _, z := range zeilen {
		if _, err := br.Exec(); err != nil {
			closeutil.LogClose(br, "LUSD-Batch nach Fehler")
			return fmt.Errorf("schüler %s konnte nicht aktualisiert werden: %w", z.SchuelerID, err)
		}
	}

	if err := br.Close(); err != nil {
		return fmt.Errorf("LUSD-Sammelaktualisierung abschließen: %w", err)
	}
	return nil
}

// LoescheWartendeVormerkungen löscht die wartenden Vormerkungen der genannten Schüler. Wer die
// Schule verlässt, holt kein reserviertes Buch mehr ab; seine Vormerkung hielte den Titel für
// die anderen besetzt.
func LoescheWartendeVormerkungen(ctx context.Context, tx pgx.Tx, schuelerIDs []string) error {
	_, err := tx.Exec(ctx,
		"DELETE FROM vormerkungen WHERE schueler_id = ANY($1) AND status = 'wartend'", schuelerIDs,
	)
	return err
}

// ZaehleOffeneAusleihenJeSchueler zählt je Schüler die nicht zurückgegebenen Ausleihen.
func ZaehleOffeneAusleihenJeSchueler(ctx context.Context, tx pgx.Tx, schuelerIDs []string) (map[string]int, error) {
	return zaehleJeSchueler(ctx, tx, "SELECT schueler_id, COUNT(*) FROM ausleihen WHERE schueler_id = ANY($1) AND rueckgabe_am IS NULL GROUP BY schueler_id", schuelerIDs)
}

// ZaehleOffeneSchaedenJeSchueler zählt je Schüler die unbezahlten Schadensfälle. Eine Stornierung
// setzt ist_bezahlt, deshalb genügt die eine Bedingung.
func ZaehleOffeneSchaedenJeSchueler(ctx context.Context, tx pgx.Tx, schuelerIDs []string) (map[string]int, error) {
	return zaehleJeSchueler(ctx, tx, "SELECT schueler_id, COUNT(*) FROM schadensfaelle WHERE schueler_id = ANY($1) AND ist_bezahlt = false GROUP BY schueler_id", schuelerIDs)
}

// zaehleJeSchueler führt eine Zählabfrage mit den Spalten schueler_id und Anzahl aus und
// liest sie ganz, bevor die Transaktion weiterbenutzt wird.
func zaehleJeSchueler(ctx context.Context, tx pgx.Tx, abfrage string, schuelerIDs []string) (map[string]int, error) {
	zahlen := make(map[string]int, len(schuelerIDs))
	rows, err := tx.Query(ctx, abfrage, schuelerIDs)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var sID string
		var count int
		if err := rows.Scan(&sID, &count); err != nil {
			rows.Close()
			return nil, err
		}
		zahlen[sID] = count
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return zahlen, nil
}

// SperreAbgaenger sperrt einen Abgänger und lässt Name und Kontaktdaten stehen: Mahnung und
// Rechnung laufen noch, oder die Karenzzeit läuft.
//
// Ein vorhandener Sperrgrund bleibt, sonst gilt der genannte; chk_schueler_block_reason
// verlangt einen. abgaenger_jahr wird das Jahr des Abgangs in der Zeitzone der Schule: Der
// Wert vom Anlegen liegt in der Zukunft, und der Löschjob (abgaenger_jahr vor dem Stichjahr)
// erfasste den Abgänger sonst nie; in der Zone der Datenbank gerechnet fiele ein Import in
// der ersten Stunde des Jahres ins Vorjahr. abgaenger_seit wird beim ersten Abgang gesetzt und
// danach nicht verschoben: Die Karenz läuft ab dem Tag, an dem der Schüler aus dem Export fiel.
func SperreAbgaenger(ctx context.Context, tx pgx.Tx, schuelerID, grund string) error {
	_, err := tx.Exec(ctx,
		`UPDATE schueler SET ist_abgaenger = true, ist_gesperrt = true,
		        block_reason = COALESCE(NULLIF(block_reason, ''), $2),
		        abgaenger_seit = COALESCE(abgaenger_seit, NOW()),
		        abgaenger_jahr = EXTRACT(YEAR FROM NOW() AT TIME ZONE $3)::int, aktualisiert_am = NOW()
		 WHERE id = $1`,
		schuelerID, grund, schulzeit.Zone().String())
	return err
}

// AnonymisiereAbgaenger entfernt die Personendaten eines Abgängers ohne offene Vorgänge:
// Foto, Name, Anschrift, E-Mail, Geburtsdatum, Schuleintritt, LUSD-ID und Ausweisnummer. Die
// Kennung der Zeile steht im Nachnamen, damit die Eindeutigkeit der Namen hält.
//
// Der Sperrgrund wird ein fester Text, weil ein alter Grund Personenbezug tragen kann.
// anonymized_at und die anonyme Ausweisnummer setzt auch der nächtliche Lauf
// (RunGDPRAnonymizeOldData); an anonymized_at hängen dessen Nachlauf über die Nebentabellen und
// die Sperre des Papierkorbs. abgaenger_jahr wird wie in SperreAbgaenger das Jahr des Abgangs,
// damit der Löschjob die Zeile später entfernt. Die Spuren in den Nebentabellen tilgt
// TilgeSchuelerSpuren in derselben Transaktion.
func AnonymisiereAbgaenger(ctx context.Context, tx pgx.Tx, schuelerID string) error {
	if _, err := tx.Exec(ctx, "DELETE FROM schueler_fotos WHERE schueler_id = $1", schuelerID); err != nil {
		return err
	}

	anonymisiertName := fmt.Sprintf("Anonymisiert-%s", schuelerID)
	if _, err := tx.Exec(ctx, `
		UPDATE schueler SET
			vorname = 'Abgänger', nachname = $1, klasse = 'ABG',
			strasse = NULL, hausnummer = NULL, plz = NULL, ort = NULL, eltern_email = NULL,
			geburtsdatum = NULL, schul_eintritt_am = NULL, lusd_id = NULL,
			barcode_id = 'ANON-' || id::text, anonymized_at = NOW(),
			abgaenger_seit = COALESCE(abgaenger_seit, NOW()),
			ist_abgaenger = true, ist_gesperrt = true, block_reason = 'Abgänger anonymisiert',
			abgaenger_jahr = EXTRACT(YEAR FROM NOW() AT TIME ZONE $3)::int, aktualisiert_am = NOW()
		WHERE id = $2`,
		anonymisiertName, schuelerID, schulzeit.Zone().String()); err != nil {
		return err
	}
	return TilgeSchuelerSpuren(ctx, tx, schuelerID, "DSGVO-Anonymisierung (LUSD-Abgang)")
}
