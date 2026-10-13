package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/jackc/pgx/v5"
)

// SpurenDatenbank ist, woran eine Tilgung läuft: eine Transaktion oder der Pool. Query gehört
// dazu, weil eine Tilgung das Ergebnis ihres eigenen DELETE brauchen kann (Vormerkungen,
// vormerkung_nachruecken.go).
type SpurenDatenbank interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// SpurTilgung ist eine Anweisung der Tilgung über eine Menge von Lesern. Die Liste darunter ist
// die einzige Quelle dafür, welche Spuren eines Lesers in den Nebentabellen stehen: Das
// endgültige Löschen und der Abgang aus der LUSD (ein Leser, in der Transaktion, Abbruch beim
// ersten Fehler) und der nächtliche Lauf (alle anonymisierten, am Pool, weiter beim Fehler)
// fahren dieselben Anweisungen. Mit zwei Listen fehlte dem Lauf die Lesehistorie, und der Name
// überlebte die Anonymisierung (jobs/dsgvo_spuren_paarung_pg_test.go).
type SpurTilgung struct {
	Beschreibung string
	sql          string // $1 = Schüler-IDs als text[]; $2 = Grund (nur brauchtGrund)
	brauchtGrund bool
	// schritt steht statt sql, wenn eine Tilgung mehr ist als eine Anweisung. Sie bleibt ein
	// Eintrag dieser Liste, damit alle Wege dasselbe tun.
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

// SpurTilgungen liefert die Liste für den nächtlichen Lauf, der über Mengen arbeitet
// (jobs/cron_dsgvo.go). Das endgültige Löschen und der Abgang gehen über TilgeSchuelerSpuren.
func SpurTilgungen() []SpurTilgung { return spurTilgungen }

var spurTilgungen = []SpurTilgung{
	{
		// Zugangskonto: Das Konto bleibt, es ist die Anmeldung samt Rechten und gehört der
		// Anlage. Was fällt, ist die Verknüpfung: Nach der Tilgung zeigt keine Anmeldung mehr
		// auf die getilgte Person. Der Fremdschlüssel (ON DELETE SET NULL) greift erst, wenn
		// die Leserzeile verschwindet; diese Anweisung räumt den Fall davor, in dem die Zeile
		// anonymisiert stehen bleibt. Wiederholbar: NULL bleibt NULL.
		Beschreibung: "benutzer (Verknuepfung zur Leserzeile)",
		sql: `UPDATE benutzer SET leser_id = NULL, aktualisiert_am = NOW()
			WHERE leser_id = ANY($1::uuid[])`,
	},
	{
		// Nachbuch-Meldungen: Die Meldung bleibt als Vorgang (Barcode, Ergebnis, Grund), der
		// Personenbezug fällt, beide Spalten mit dem Leser werden NULL. Wie beim Konto räumt
		// die Anweisung den Fall, in dem die Zeile anonymisiert stehen bleibt. Wiederholbar.
		Beschreibung: "nachbuch_meldungen (Ausleiher und Vorbesitzer)",
		sql: `UPDATE nachbuch_meldungen
			SET ausleiher_schueler_id = CASE WHEN ausleiher_schueler_id = ANY($1::uuid[]) THEN NULL ELSE ausleiher_schueler_id END,
			    vorbesitzer_schueler_id = CASE WHEN vorbesitzer_schueler_id = ANY($1::uuid[]) THEN NULL ELSE vorbesitzer_schueler_id END
			WHERE ausleiher_schueler_id = ANY($1::uuid[]) OR vorbesitzer_schueler_id = ANY($1::uuid[])`,
	},
	{
		// Schadensersatz-Bescheid: Der Brief bleibt als Beleg, über seine Referenznummer werden
		// Zahlungen zugeordnet. Was fällt, ist die Aufnahme des Empfängers (Anrede, Name,
		// Anschrift zum Datum des Briefs), die sonst im JSON stehen bliebe; den Fremdschlüssel
		// leert ON DELETE SET NULL. Wiederholbar: Ein leeres Objekt bleibt leer.
		Beschreibung: "schadensersatz_bescheide (Empfänger-Snapshot)",
		sql: `UPDATE schadensersatz_bescheide
			SET empfaenger_snapshot = '{}'::jsonb
			WHERE schueler_id = ANY($1::uuid[]) AND empfaenger_snapshot <> '{}'::jsonb`,
	},
	{
		// Datensatz-Historie: LegeLeserInPapierkorb legt Vor- und Nachname, Klasse und Barcode
		// in details ab; das ganze Objekt wird durch die Marke der Anonymisierung ersetzt.
		// Wiederholbar über die Marke.
		Beschreibung: "audit_log (Datensatz-Historie, tabelle='schueler')",
		sql: `UPDATE audit_log
			SET details = jsonb_build_object('anonymisiert', true, 'grund', $2::text)
			WHERE tabelle = 'schueler' AND datensatz_id = ANY($1::uuid[])
			  AND (details IS NULL OR NOT (details ? 'anonymisiert'))`,
		brauchtGrund: true,
	},
	{
		// Lesehistorie: dieselben Zeilen, die die Auskunft dem Leser zurechnet
		// (dsgvo_auskunft.go: tabelle = 'ausleihen' und details->>'schueler_id'). Der Eintrag
		// bleibt als Nachweis, dass ein Exemplar unterwegs war; es fallen die Schlüssel mit
		// Personenbezug, dieselben, die auch die Befristung der Lesehistorie entfernt
		// (jobs/cron_dsgvo_lesehistorie.go).
		Beschreibung: "audit_log (Lesehistorie, tabelle='ausleihen')",
		sql: `UPDATE audit_log
			SET details = details - 'schueler_id' - 'entleiher'
			WHERE tabelle = 'ausleihen' AND details->>'schueler_id' = ANY($1::text[])`,
	},
	{
		// Name und Freitext neben der Kennung des Lesers in der Datensatz-Historie: die Spur
		// einer offenen Forderung oder Vormerkung, deren Titel gelöscht wurde (schuldner,
		// beschreibung, betrifft). Die Einträge bleiben als Beleg, die Kennung ist danach ein
		// Pseudonym. Die Befristung der Lesehistorie erreicht diese Zeilen nicht, sie arbeitet
		// nur auf tabelle = 'ausleihen' (titel_loeschspur_tilgung_pg_test.go).
		Beschreibung: "audit_log (Personenbezug neben der Leserkennung)",
		sql:          tilgePersonenbezugImProtokoll("audit_log"),
	},
	{
		// Im Protokoll der Verwaltung fallen nur die Schlüssel mit Personenbezug; Aktion, Zeit
		// und schueler_id (nach der Anonymisierung ein Pseudonym) bleiben für die Rechenschaft.
		// Dazu gehören die LUSD-ID, der Barcode des Ausweises (die anonyme Zeile bekommt einen
		// neuen, damit die Karte nicht mehr aufgeht; die alte Nummer stünde sonst bis zum Ende
		// der Aufbewahrung neben der Kennung) und der Grund einer Sperre, der andere Personen
		// nennen kann. Die Schlüssel stehen in protokollSchluesselMitPersonenbezug, einer Liste
		// für beide Protokolle (api/leser/sperrgrund_tilgung_pg_test.go).
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
		// Vormerkungen: Die Notiz kann die Person nennen, und die Vormerkung eines gelöschten
		// oder anonymisierten Lesers hat keine Aufgabe mehr. Beim endgültigen Löschen räumt sie
		// auch der Fremdschlüssel (CASCADE); sie steht trotzdem hier, damit der nächtliche
		// Lauf, bei dem die Zeile anonymisiert weiterbesteht, dieselbe Liste fährt.
		//
		// Kein bloßes DELETE: Lag für den Leser schon ein Exemplar zum Abholen bereit, geht es
		// an den nächsten Wartenden; sonst bliebe das Buch im Abholregal liegen.
		Beschreibung: "vormerkungen (gelöscht, Warteschlange nachgerückt)",
		schritt:      loescheVormerkungenUndRueckeNach,
	},
}

// TilgeSchuelerSpuren entfernt die Personendaten eines Lesers aus den Nebentabellen: der
// gemeinsame Schritt der Anonymisierung eines Abgängers (AnonymisiereAbgaenger) und des
// endgültigen Löschens. Wiederholbar; es läuft vor dem DELETE des Lesers, weil audit_logs nur
// über details->>'schueler_id' am Leser hängt und nicht über einen Fremdschlüssel.
func TilgeSchuelerSpuren(ctx context.Context, ex SpurenDatenbank, schuelerID, grund string) error {
	for _, st := range spurTilgungen {
		if _, err := st.Tilge(ctx, ex, []string{schuelerID}, grund); err != nil {
			return fmt.Errorf("tilgung %s: %w", st.Beschreibung, err)
		}
	}
	return nil
}
