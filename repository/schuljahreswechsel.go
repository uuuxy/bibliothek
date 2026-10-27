package repository

import (
	"context"
	"errors"
	"time"

	"bibliothek/pkg/schulzeit"

	"github.com/jackc/pgx/v5"
)

// schuljahreswechselSperre ist der feste Advisory-Lock-Schlüssel, der gleichzeitige
// Versetzungsläufe hart serialisiert (siehe SperreSchuljahreswechsel). Ein beliebiger,
// aber im Projekt eindeutiger Wert.
const schuljahreswechselSperre int64 = 748_2026

// AktionSchuljahreswechsel ist die Aktion, unter der ein Lauf in audit_logs steht. An ihr
// erkennt FindeJuengstenSchuljahreswechsel den Lauf davor.
const AktionSchuljahreswechsel = "SCHULJAHRESWECHSEL"

// schuljahreswechselAbstandMonate ist der Abstand, den ein Lauf vom vorigen mindestens hat. Der
// Wechsel läuft einmal im Jahr, in den Sommerferien; ein zweiter im selben Halbjahr versetzte alle
// Schüler ein weiteres Mal, und von einem Lauf weiß nur sein Eintrag im Protokoll. Der Abstand
// bleibt in der kürzesten Aufbewahrung des Protokolls (protokollAufbewahrungMonateMinimum), sonst
// löschte der Nachtlauf den Eintrag, bevor er endet.
const schuljahreswechselAbstandMonate = 6

// SperreSchuljahreswechsel hält gleichzeitige Läufe auseinander: Der zweite wartet, bis die
// Transaktion des ersten endet, und sieht dann dessen Eintrag im Protokoll.
func SperreSchuljahreswechsel(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, schuljahreswechselSperre)
	return err
}

// JuengsterSchuljahreswechsel ist ein Lauf, der den nächsten noch sperrt.
type JuengsterSchuljahreswechsel struct {
	Lief        time.Time
	NaechsterAb time.Time
}

// FindeJuengstenSchuljahreswechsel liefert den jüngsten Lauf, nach dem noch kein weiterer geht;
// nil, wenn keiner sperrt. Der Eintrag eines Laufs entsteht in dessen Transaktion; ein
// abgebrochener Lauf hinterlässt keinen. Den Abstand zählt die Uhr der Schule: ein halbes Jahr
// später zur selben Uhrzeit, auch über die Umstellung auf Sommerzeit.
func FindeJuengstenSchuljahreswechsel(ctx context.Context, db Datenbank) (*JuengsterSchuljahreswechsel, error) {
	var lauf JuengsterSchuljahreswechsel
	err := db.QueryRow(ctx, `
		WITH lauf AS (
			SELECT zeitstempel,
			       ((zeitstempel AT TIME ZONE '`+schulzeit.ZonenName+`') + make_interval(months => $2::int))
			           AT TIME ZONE '`+schulzeit.ZonenName+`' AS naechster_ab
			FROM audit_logs
			WHERE aktion = $1
		)
		SELECT zeitstempel, naechster_ab FROM lauf
		WHERE naechster_ab > NOW()
		ORDER BY zeitstempel DESC
		LIMIT 1`, AktionSchuljahreswechsel, schuljahreswechselAbstandMonate).Scan(&lauf.Lief, &lauf.NaechsterAb)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &lauf, nil
}

// sqlVersetzeSchueler zählt Klassenbezeichnungen um eine Stufe hoch und markiert
// Abschlussklassen als Abgänger. Gelesen werden nur Klassen mit einem Jahrgang von 1 bis 13
// (jahrgangImBereichSQL); die übrigen nennt LadeKlassenAusserhalbDerVersetzung. Wer Abschlussklasse ist, sagt allein AbschlussklasseSQL,
// dieselbe Regel, nach der die Abgängerliste (api/mahnwesen/abgaenger.go) diese Klassen von Mai
// bis Juli zum Einsammeln zeigt.
//
// Zwei Regeln hält die Anweisung:
//   - klasse ist NOT NULL (schema.sql): Abgänger bekommen 'ABG' wie beim LUSD-Import
//     (berechneLUSDAenderungen), damit beide Wege dieselbe Konvention schreiben.
//   - lpad erhält führende Nullen ('05a' → '06a'), ohne beim Stellenwechsel zu kürzen
//     ('09' → '10'; greatest() verhindert, dass lpad abschneidet).
var sqlVersetzeSchueler = `
	WITH parsed AS (
		SELECT id,
			   klasse,
			   substring(klasse from '^\d+') AS old_digits,
			   (substring(klasse from '^\d+')::int + 1) AS new_grade,
			   substring(klasse from '^\d+(.*)$') AS new_suffix,
			   ` + AbschlussklasseSQL("klasse") + ` AS is_graduating
		FROM schueler
		WHERE ist_abgaenger = false
		  AND deleted_at IS NULL
		  AND ` + jahrgangImBereichSQL("klasse") + `
	),
	calculated AS (
		SELECT id,
			   (lpad(new_grade::text, greatest(length(old_digits), length(new_grade::text)), '0') || new_suffix) AS new_klasse,
			   is_graduating
		FROM parsed
	),
	updated AS (
		UPDATE schueler s
		SET
			klasse = CASE WHEN c.is_graduating THEN 'ABG' ELSE c.new_klasse END,
			ist_abgaenger = c.is_graduating,
			ist_gesperrt = CASE WHEN c.is_graduating THEN true ELSE s.ist_gesperrt END,
			-- Abgänger werden gesperrt → chk_schueler_block_reason verlangt einen Grund.
			-- Ein vorhandener (z. B. manueller) Grund bleibt erhalten.
			block_reason = CASE WHEN c.is_graduating
			                    THEN COALESCE(NULLIF(s.block_reason, ''), '` + abgaengerSperrgrundSchuljahreswechsel + `')
			                    ELSE s.block_reason END,
			-- Karenz-Uhr: beim ersten Abgang gestempelt, wie im LUSD-Pfad
			-- (SperreAbgaenger) — Import, Job und Wächter rechnen die Anonymisierung daran.
			abgaenger_seit = CASE WHEN c.is_graduating THEN COALESCE(s.abgaenger_seit, NOW()) ELSE s.abgaenger_seit END,
			abgaenger_jahr = CASE
				WHEN c.is_graduating THEN EXTRACT(YEAR FROM NOW() AT TIME ZONE $1)
				ELSE s.abgaenger_jahr
			END,
			aktualisiert_am = CURRENT_TIMESTAMP
		FROM calculated c
		WHERE s.id = c.id
		RETURNING c.is_graduating
	)
	SELECT
		COUNT(*) FILTER (WHERE is_graduating = false) AS versetzt,
		COUNT(*) FILTER (WHERE is_graduating = true) AS abgaenger
	FROM updated;
`

// VersetzeSchueler zählt die Klasse jedes aktiven Schülers, deren Name mit einem Jahrgang von 1
// bis 13 beginnt, um eine Stufe hoch, macht die Abschlussklassen zu Abgängern und liefert beide Zahlen. zone
// ist die Zeitzone der Schule; in ihr wird das Abgangsjahr bestimmt.
func VersetzeSchueler(ctx context.Context, db Datenbank, zone string) (versetzt, abgaenger int, err error) {
	err = db.QueryRow(ctx, sqlVersetzeSchueler, zone).Scan(&versetzt, &abgaenger)
	return versetzt, abgaenger, err
}

// LadeKlassenAusserhalbDerVersetzung nennt die Klassen aktiver Schüler, die mit einer Zahl
// beginnen, aber mit keinem Jahrgang von 1 bis 13. Die Versetzung lässt sie stehen; die
// Vorschau nennt sie, damit sie vorher berichtigt werden. Eine Zeile je Klasse.
func LadeKlassenAusserhalbDerVersetzung(ctx context.Context, db Datenbank) ([]string, error) {
	rows, err := db.Query(ctx, `
		SELECT klasse FROM schueler
		WHERE ist_abgaenger = false
		  AND deleted_at IS NULL
		  AND klasse ~ '^\d'
		  AND NOT `+jahrgangImBereichSQL("klasse")+`
		GROUP BY klasse
		ORDER BY klasse COLLATE deutsch`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}
