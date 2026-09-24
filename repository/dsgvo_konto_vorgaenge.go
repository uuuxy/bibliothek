package repository

// dsgvo_konto_vorgaenge.go — die Vorgänge, die eine Person mit ihrem Zugangskonto selbst
// bearbeitet hat: gebuchte Ausleihen und Rückgaben, Stornierungen, Bescheide, Inventuren,
// bearbeitete Meldungen, Protokoll- und Verwaltungseinträge. Das VVT zählt sie zu ihren Daten
// (Tätigkeit 3); entschieden am 24.09.2026: in die Auskunft, OHNE die Daten Dritter
// (OFFEN.md 5.19, Art. 15 Abs. 4 DSGVO). Deshalb liest jede Teilabfrage nur Zeitpunkt,
// Handlung und — beim Verwaltungsprotokoll — die IP-Adresse des eigenen Arbeitsplatzes. Wer
// betroffen war, welches Buch, welcher Betrag, welche Protokoll-Details: nichts davon.
//
// Jede Spalte, die auf benutzer zeigt, steht hier oder bei den Anfragen (dsgvo_konto.go);
// die Ratsche api/dsgvo_konto_quellen_pg_test.go hält die Liste gegen das Schema.
//
// Der Zeitpunkt ist ein Zeiger: Bearbeiter und Zeitpunkt setzen die Schreiber zusammen, die
// Datenbank verlangt es aber nicht. Eine Zeile mit Bearbeiter und ohne Zeitpunkt soll in der
// Auskunft stehen, statt sie mit einem Scan-Fehler abzubrechen.

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// DsgvoEigenerVorgang ist ein Vorgang, den die Person selbst bearbeitet hat.
type DsgvoEigenerVorgang struct {
	Zeitpunkt *time.Time `json:"zeitpunkt"`
	Handlung  string     `json:"handlung"`
	// Nur im Verwaltungsprotokoll gespeichert (audit_logs.ip_adresse): der Arbeitsplatz,
	// an dem die Person gearbeitet hat.
	IPAdresse *string `json:"ip_adresse"`
}

const dsgvoEigeneVorgaengeSQL = `
	SELECT zeitpunkt, handlung, ip FROM (
		SELECT a.ausgeliehen_am AS zeitpunkt, 'Ausleihe gebucht' AS handlung, NULL::text AS ip
		FROM ausleihen a WHERE a.bearbeiter_id = $1
		UNION ALL
		SELECT a.rueckgabe_am, 'Rückgabe gebucht', NULL
		FROM ausleihen a WHERE a.rueckgabe_bearbeiter_id = $1
		UNION ALL
		SELECT sf.storniert_am, 'Schadensfall storniert', NULL
		FROM schadensfaelle sf WHERE sf.storniert_von = $1
		UNION ALL
		SELECT b.erstellt_am, 'Schadensersatz-Bescheid erstellt', NULL
		FROM schadensersatz_bescheide b WHERE b.erstellt_von = $1
		UNION ALL
		SELECT i.gestartet_am,
		       'Inventur begonnen' || CASE WHEN btrim(i.scope_label) <> '' THEN ': ' || i.scope_label ELSE '' END,
		       NULL
		FROM inventur_sessions i WHERE i.gestartet_von = $1
		UNION ALL
		SELECT n.quittiert_am, 'Meldung nach Netzausfall bearbeitet', NULL
		FROM nachbuch_meldungen n WHERE n.quittiert_von = $1
		UNION ALL
		SELECT l.timestamp, 'Protokolleintrag: ' || l.aktion || ' (' || l.tabelle || ')', NULL
		FROM audit_log l WHERE l.bearbeiter_id = $1
		UNION ALL
		SELECT g.zeitstempel, 'Verwaltungseingriff: ' || g.aktion, g.ip_adresse
		FROM audit_logs g WHERE g.admin_id = $1
	) v
	ORDER BY zeitpunkt DESC NULLS LAST`

// leseDsgvoEigeneVorgaenge liest die selbst bearbeiteten Vorgänge eines Kontos. Die Liste
// ist nicht begrenzt: Eine Auskunft ist vollständig oder falsch. Bei einer Bibliothekskraft
// sind es über die Aufbewahrung der Protokolle hinweg Tausende Zeilen.
func leseDsgvoEigeneVorgaenge(ctx context.Context, q DBQueryer, kontoID string) ([]DsgvoEigenerVorgang, error) {
	vorgaenge, err := sammle(ctx, q, dsgvoEigeneVorgaengeSQL, func(r pgx.Rows) (DsgvoEigenerVorgang, error) {
		var v DsgvoEigenerVorgang
		return v, r.Scan(&v.Zeitpunkt, &v.Handlung, &v.IPAdresse)
	}, kontoID)
	if err != nil {
		return nil, fmt.Errorf("selbst bearbeitete vorgänge: %w", err)
	}
	return vorgaenge, nil
}
