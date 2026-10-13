package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/jackc/pgx/v5"
)

// SpurenDatenbank ist der kleinste gemeinsame Nenner von pgx.Tx und dem Pool. Query
// steht mit drin, seit eine Tilgung nicht mehr nur schreibt, sondern das Ergebnis ihres
// eigenen DELETE braucht (Vormerkungen, siehe vormerkung_nachruecken.go).
type SpurenDatenbank interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// SpurTilgung ist EINE Anweisung der Spuren-Tilgung, parametrisiert über eine
// Schüler-MENGE. Die Liste darunter ist die einzige Quelle dafür, welche Spuren eines
// Schülers in den Neben-Tabellen stehen — Purge/LUSD-Abgang (ein Schüler, in der Tx,
// Abbruch beim ersten Fehler) und der nächtliche Cron (alle anonymisierten, am Pool,
// weiter beim Fehler) fahren DIESELBEN Statements und unterscheiden sich nur in
// Menge und Fehlerpolitik.
//
// Anlass (31.08.2026): Cron und Purge pflegten die Liste getrennt; dem Cron fehlte
// genau die Lesehistorie — der Klarname (details->>'entleiher') überlebte die
// Anonymisierung (jobs/dsgvo_spuren_paarung_pg_test.go, am alten Stand rot gesehen).
type SpurTilgung struct {
	Beschreibung string
	sql          string // $1 = Schüler-IDs als text[]; $2 = Grund (nur brauchtGrund)
	brauchtGrund bool
	// schritt steht STATT sql, wenn eine Tilgung mehr ist als eine Anweisung — sie
	// bleibt trotzdem ein Eintrag DIESER Liste, damit Purge, LUSD-Abgang und Cron
	// weiterhin dasselbe tun.
	schritt func(ctx context.Context, ex SpurenDatenbank, schuelerIDs []string) (int64, error)
}

// Tilge führt die Anweisung für die gegebene Schüler-Menge aus und meldet die Zahl
// der betroffenen Zeilen.
func (st SpurTilgung) Tilge(ctx context.Context, ex SpurenDatenbank, schuelerIDs []string, grund string) (int64, error) {
	if st.schritt != nil {
		return st.schritt(ctx, ex, schuelerIDs)
	}
	args := []any{schuelerIDs}
	if st.brauchtGrund {
		args = append(args, grund)
	}
	tag, err := ex.Exec(ctx, st.sql, args...)
	return tag.RowsAffected(), err
}

// SpurTilgungen liefert die Statement-Liste für den set-basierten Cron
// (jobs/cron_dsgvo.go). Purge und LUSD-Pfad gehen über TilgeSchuelerSpuren.
func SpurTilgungen() []SpurTilgung { return spurTilgungen }

var spurTilgungen = []SpurTilgung{
	{
		// Zugangskonto (Migration 123): Das KONTO bleibt — es ist die Anmeldung samt
		// Rechten und gehoert nicht dem Leser, sondern der Anlage. Was faellt, ist die
		// Verknuepfung: Nach der Tilgung soll keine Anmeldung mehr auf diese getilgte
		// Person zeigen. Der Fremdschluessel steht auf ON DELETE SET NULL und greift erst,
		// wenn die Leserzeile ganz verschwindet; diese Zeile raeumt den Fall davor
		// (Anonymisierung, Zeile bleibt). Idempotent: NULL bleibt NULL.
		Beschreibung: "benutzer (Verknuepfung zur Leserzeile)",
		sql: `UPDATE benutzer SET leser_id = NULL, aktualisiert_am = NOW()
			WHERE leser_id = ANY($1::uuid[])`,
	},
	{
		// Nachbuch-Meldungen (Migration 117): Die Meldung bleibt als Vorgang — Barcode,
		// Ergebnis, Grund —, der Personenbezug fällt: beide Schüler-Spalten auf NULL. Die
		// Fremdschlüssel stehen auf ON DELETE SET NULL, sobald der Datensatz verschwindet;
		// diese Zeile räumt die Anonymisierung davor (anonymized_at, Zeile bleibt).
		// Idempotent: NULL bleibt NULL.
		Beschreibung: "nachbuch_meldungen (Ausleiher und Vorbesitzer)",
		sql: `UPDATE nachbuch_meldungen
			SET ausleiher_schueler_id = CASE WHEN ausleiher_schueler_id = ANY($1::uuid[]) THEN NULL ELSE ausleiher_schueler_id END,
			    vorbesitzer_schueler_id = CASE WHEN vorbesitzer_schueler_id = ANY($1::uuid[]) THEN NULL ELSE vorbesitzer_schueler_id END
			WHERE ausleiher_schueler_id = ANY($1::uuid[]) OR vorbesitzer_schueler_id = ANY($1::uuid[])`,
	},
	{
		// Schadensersatz-Bescheid (Migration 110): Der Brief BLEIBT als Beleg — über seine
		// Referenznummer werden Zahlungen zugeordnet, und Rechnungsunterlagen liegen
		// Jahre. Was fällt, ist der Personenbezug: der Empfänger-Snapshot (Anrede, Name,
		// Anschrift zum Briefdatum). Der Fremdschlüssel wird von ON DELETE SET NULL
		// geleert, sobald der Schülerdatensatz verschwindet; diese Zeile räumt den
		// Klartext, der sonst im JSONB stehen bliebe. Idempotent: leeres Objekt bleibt leer.
		Beschreibung: "schadensersatz_bescheide (Empfänger-Snapshot)",
		sql: `UPDATE schadensersatz_bescheide
			SET empfaenger_snapshot = '{}'::jsonb
			WHERE schueler_id = ANY($1::uuid[]) AND empfaenger_snapshot <> '{}'::jsonb`,
	},
	{
		// Datensatz-Historie: LegeLeserInPapierkorb legt Vor-/Nachname, Klasse und Barcode in
		// details ab; das ganze Objekt wird durch den Anonymisierungs-Marker ersetzt.
		// Idempotent über den Marker.
		Beschreibung: "audit_log (Datensatz-Historie, tabelle='schueler')",
		sql: `UPDATE audit_log
			SET details = jsonb_build_object('anonymisiert', true, 'grund', $2::text)
			WHERE tabelle = 'schueler' AND datensatz_id = ANY($1::uuid[])
			  AND (details IS NULL OR NOT (details ? 'anonymisiert'))`,
		brauchtGrund: true,
	},
	{
		// Lesehistorie: dieselben Zeilen, die die Art.-15-Auskunft dem Schüler zurechnet
		// (repository/dsgvo_auskunft.go: tabelle='ausleihen' AND details->>'schueler_id' = id).
		// Die Buchungshistorie selbst BLEIBT (Nachweis, dass ein Exemplar unterwegs war),
		// nur ihr Personenbezug fällt — dieselben Schlüssel, die auch die
		// Lesehistorie-Befristung entfernt (jobs/cron_dsgvo_lesehistorie.go).
		Beschreibung: "audit_log (Lesehistorie, tabelle='ausleihen')",
		sql: `UPDATE audit_log
			SET details = details - 'schueler_id' - 'entleiher'
			WHERE tabelle = 'ausleihen' AND details->>'schueler_id' = ANY($1::text[])`,
	},
	{
		// Name und Freitext, die neben der Kennung des Lesers in der Datensatz-Historie
		// stehen: die Spur einer offenen Forderung oder Vormerkung, deren Titel gelöscht wurde
		// (schuldner, beschreibung, betrifft). Die Einträge bleiben als Beleg, die Kennung ist
		// danach ein Pseudonym. Bis zum 29.09.2026 fehlte diese Anweisung; die
		// Lesehistorie-Befristung erreicht diese Zeilen nicht (sie arbeitet nur auf
		// tabelle = 'ausleihen'). repository/titel_loeschspur_tilgung_pg_test.go.
		Beschreibung: "audit_log (Personenbezug neben der Leserkennung)",
		sql:          tilgePersonenbezugImProtokoll("audit_log"),
	},
	{
		// Die staatliche LUSD-ID (LUSD_ID_NACHGETRAGEN) und die Ausweis-Barcodes
		// (SCHUELER_ZUSAMMENGEFUEHRT): Nur die Schlüssel mit Personenbezug fallen; Aktion,
		// Zeit und schueler_id (nach Anonymisierung ein Pseudonym) bleiben für die
		// Rechenschaftspflicht erhalten. Der Barcode gehört dazu, weil die anonyme Hülle
		// einen ANON-Barcode bekommt, damit die physische Karte nicht mehr aufgeht — im
		// Verwaltungsprotokoll stand die alte Nummer sonst bis zu 24 Monate neben der UUID
		// (Rasterdurchgang 02.09.2026).
		// Der Grund einer Sperre ebenso: Die Anonymisierung ersetzt block_reason, weil er
		// andere Personen nennen kann, und die Sperr-Tür schreibt ihn als grund ins Protokoll
		// (LESER_GESPERRT, LESER_ENTSPERRT), das Übergehen an der Theke bis zum 24.09.2026 als
		// reason (OVERRIDE_BLOCK). Rasterdurchgang 24.09.2026,
		// api/leser/sperrgrund_tilgung_pg_test.go. Die Schlüssel stehen seit dem 29.09.2026 in
		// protokollSchluesselMitPersonenbezug, eine Liste für beide Protokolle.
		Beschreibung: "audit_logs (Personenbezug neben der Leserkennung)",
		sql:          tilgePersonenbezugImProtokoll("audit_logs"),
	},
	{
		// Einträge über ein früheres Zugangskonto des Lesers (Anlage, Änderung, eigene
		// Anmeldung) tragen das Konto als ziel_id, nicht den Leser. Name und Adresse fallen
		// dort mit denen im Löscheintrag des Kontos (kontoSchluesselMitPersonenbezug).
		// api/konto_eintraege_tilgung_pg_test.go.
		Beschreibung: "audit_logs (Name und Adresse früherer Zugangskonten)",
		sql:          tilgePersonenbezugFruehererKonten(),
	},
	{
		// Die Forderung bleibt als Beleg stehen: Betrag, Datum, bezahlt oder storniert. Was
		// dazu getippt wurde, fällt, weil es die Person oder ihre Familie nennen kann: die
		// Beschreibung und der Grund einer Stornierung. Nur bei erledigten Fällen (ist_bezahlt
		// deckt bezahlt und storniert ab); eine offene Forderung braucht ihre Begründung noch,
		// und dass ihr Leser anonymisiert wird, verhindert schon das Prädikat in loeschfristen.go.
		Beschreibung: "schadensfaelle (Freitext erledigter Fälle)",
		sql: `UPDATE schadensfaelle
			SET beschreibung = '', stornierungsgrund = NULL
			WHERE schueler_id = ANY($1::uuid[]) AND ist_bezahlt = true
			  AND (beschreibung <> '' OR stornierungsgrund IS NOT NULL)`,
	},
	{
		// Vormerkungen: die Freitext-Notiz kann personenbezogen sein, und die Vormerkung
		// eines gelöschten/anonymisierten Schülers ist funktionslos. Beim Purge räumt sie
		// auch der FK-CASCADE — hier steht sie trotzdem, damit der Cron-Pfad (Schüler
		// lebt als anonymisierte Hülle weiter) dieselbe Liste fahren kann.
		//
		// Kein reines DELETE: Lag für das Kind schon ein Exemplar abholbereit, muss es
		// an den nächsten Wartenden gehen — sonst bleibt das Buch auf dem Abholregal
		// liegen und die Schlange rückt nie nach (06.09.2026).
		Beschreibung: "vormerkungen (gelöscht, Warteschlange nachgerückt)",
		schritt:      loescheVormerkungenUndRuecktNach,
	},
}

// TilgeSchuelerSpuren entfernt die Personendaten EINES Schülers aus den Neben-Tabellen —
// gemeinsamer Schritt von LUSD-Abgänger-Anonymisierung (AnonymisiereAbgaenger) und Purge.
// Historie der Lücken: Bis 22.08.2026 hatte nur der Cron-Pfad alle damaligen Statements
// (A3: LUSD-ID überlebte 24 Monate); bis 31.08.2026 fehlte hier die Lesehistorie, danach
// fehlte sie dem Cron — seither ist spurTilgungen die eine Liste für beide.
// Idempotent; muss VOR dem DELETE des Schülers laufen (audit_logs hängt nur per
// details->>'schueler_id' am Schüler, nicht per FK).
func TilgeSchuelerSpuren(ctx context.Context, ex SpurenDatenbank, schuelerID, grund string) error {
	for _, st := range spurTilgungen {
		if _, err := st.Tilge(ctx, ex, []string{schuelerID}, grund); err != nil {
			return fmt.Errorf("tilgung %s: %w", st.Beschreibung, err)
		}
	}
	return nil
}
