package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// schuljahreswechselLockKey ist der feste Advisory-Lock-Schlüssel, der gleichzeitige
// Versetzungsläufe hart serialisiert (siehe SperreSchuljahreswechsel). Ein beliebiger,
// aber im Projekt eindeutiger Wert — er teilt keinen Namensraum mit den tabellenbasierten
// Sequenz-Locks (advisoryLockKey in sequence_repo.go).
const schuljahreswechselLockKey int64 = 748_2026

// AktionSchuljahreswechsel ist die Aktion, unter der ein Lauf in audit_logs steht. An ihr
// erkennt ZaehleJuengsteSchuljahreswechsel den Lauf davor.
const AktionSchuljahreswechsel = "SCHULJAHRESWECHSEL"

// SperreSchuljahreswechsel hält gleichzeitige Läufe auseinander: Der zweite wartet, bis die
// Transaktion des ersten endet, und sieht dann dessen Eintrag im Protokoll.
func SperreSchuljahreswechsel(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, schuljahreswechselLockKey)
	return err
}

// ZaehleJuengsteSchuljahreswechsel zählt die Läufe der letzten zwölf Stunden. Der Eintrag
// eines Laufs entsteht in dessen Transaktion; ein abgebrochener Lauf hinterlässt keinen.
func ZaehleJuengsteSchuljahreswechsel(ctx context.Context, db DBQueryer) (int, error) {
	var laeufe int
	err := db.QueryRow(ctx, `
		SELECT COUNT(*) FROM audit_logs
		WHERE aktion = 'SCHULJAHRESWECHSEL'
		  AND zeitstempel > NOW() - INTERVAL '12 hours'
	`).Scan(&laeufe)
	return laeufe, err
}

// versetzeSchuelerQuery zählt Klassenbezeichnungen um eine Stufe hoch und markiert
// Abschlussklassen als Abgänger. WER Abschlussklasse ist, sagt allein
// AbschlussklasseSQL — dieselbe Regel, nach der die Abgängerliste
// (api/graduates.go) diese Klassen von Mai bis Juli zum Einsammeln zeigt. Bis zum
// 05.09.2026 stand die Regel hier zweimal als eigener CASE-Block und in der Liste gar
// nicht (dort galt ist_abgaenger — ein anderer Begriff unter demselben Namen).
//
// Wichtige Invarianten:
//   - klasse ist NOT NULL (schema.sql) — Abgänger bekommen 'ABG', exakt wie der
//     LUSD-Import-Pfad (computeLusdChanges), damit beide Wege dieselbe Konvention
//     schreiben.
//   - lpad erhält führende Nullen ('05a' → '06a'), ohne beim Stellenwechsel zu
//     kürzen ('09' → '10', greatest() verhindert lpad-Truncation).
var versetzeSchuelerQuery = `
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
		  AND klasse ~ '^\d+'
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
			                    THEN COALESCE(NULLIF(s.block_reason, ''), '` + AbgaengerSperrgrundSchuljahreswechsel + `')
			                    ELSE s.block_reason END,
			-- Karenz-Uhr (Migration 094): beim ERSTEN Abgang gestempelt, wie im LUSD-Pfad
			-- (sperreAbgaenger) — Import, Job und Wächter rechnen die Anonymisierung daran.
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

// VersetzeSchueler zählt die Klasse jedes aktiven Schülers um eine Stufe hoch, macht die
// Abschlussklassen zu Abgängern und liefert beide Zahlen. zone ist die Zeitzone der Schule;
// in ihr wird das Abgangsjahr bestimmt.
func VersetzeSchueler(ctx context.Context, db DBQueryer, zone string) (versetzt, abgaenger int, err error) {
	err = db.QueryRow(ctx, versetzeSchuelerQuery, zone).Scan(&versetzt, &abgaenger)
	return versetzt, abgaenger, err
}
