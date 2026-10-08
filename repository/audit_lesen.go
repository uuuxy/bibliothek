package repository

import (
	"context"
	"strconv"
	"time"
)

// AuditLogZeile ist ein Eintrag im Protokoll der Vorgänge mit dem Namen des Bearbeiters.
// Akteur unterscheidet USER von SYSTEM: Ein Eintrag ohne Bearbeiter wäre sonst nicht von
// einem Datenfehler zu unterscheiden.
type AuditLogZeile struct {
	ID                 string
	Tabelle            string
	Aktion             string
	DatensatzID        string
	Timestamp          time.Time
	BearbeiterID       string
	BearbeiterVorname  string
	BearbeiterNachname string
	Akteur             string
}

// ListeAuditLog liefert die jüngsten Einträge des Protokolls der Vorgänge, höchstens limit.
//
// LEFT JOIN, weil Vorgänge des Programms selbst (Bereinigung, Sicherung, automatische Sperre)
// keinen Bearbeiter tragen und trotzdem im Protokoll stehen müssen. COALESCE, weil Bearbeiter
// und Name damit leer sein können und die Felder der Zeile es nicht sind.
func ListeAuditLog(ctx context.Context, db DBQueryer, limit int) ([]AuditLogZeile, error) {
	query := `
			SELECT l.id, l.tabelle, l.aktion, l.datensatz_id, l.timestamp,
			       COALESCE(l.bearbeiter_id::text, ''),
			       COALESCE(b.vorname, ''), COALESCE(b.nachname, ''),
			       l.akteur
			FROM audit_log l
			LEFT JOIN benutzer b ON l.bearbeiter_id = b.id
			ORDER BY l.timestamp DESC
			LIMIT ` + strconv.Itoa(limit)
	rows, err := db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var zeilen []AuditLogZeile
	for rows.Next() {
		var l AuditLogZeile
		err := rows.Scan(&l.ID, &l.Tabelle, &l.Aktion, &l.DatensatzID, &l.Timestamp,
			&l.BearbeiterID, &l.BearbeiterVorname, &l.BearbeiterNachname, &l.Akteur)
		if err != nil {
			return nil, err
		}
		zeilen = append(zeilen, l)
	}
	return zeilen, rows.Err()
}

// AdminProtokollZeile ist ein Eintrag im Protokoll der Verwaltung. AdminID ist leer bei
// Einträgen ohne Bearbeiter (Nachtlauf, Migration); sie stehen trotzdem im Protokoll.
type AdminProtokollZeile struct {
	ID          string
	AdminID     *string
	AdminName   string
	Aktion      string
	Details     any
	IpAdresse   string
	Zeitstempel time.Time
}

// ListeAdminProtokoll liefert die jüngsten 1000 Einträge des Protokolls der Verwaltung. Eine
// Zeile, die sich nicht lesen lässt, wird ausgelassen.
func ListeAdminProtokoll(ctx context.Context, db DBQueryer) ([]AdminProtokollZeile, error) {
	query := `
			SELECT 
				a.id, a.admin_id, coalesce(b.vorname || ' ' || b.nachname, 'System/Unbekannt'),
				a.aktion, a.details, coalesce(a.ip_adresse, ''), a.zeitstempel
			FROM audit_logs a
			LEFT JOIN benutzer b ON a.admin_id = b.id
			ORDER BY a.zeitstempel DESC
			LIMIT 1000
		`
	rows, err := db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var zeilen []AdminProtokollZeile
	for rows.Next() {
		var l AdminProtokollZeile
		if err := rows.Scan(&l.ID, &l.AdminID, &l.AdminName, &l.Aktion, &l.Details, &l.IpAdresse, &l.Zeitstempel); err != nil {
			continue
		}
		zeilen = append(zeilen, l)
	}
	return zeilen, rows.Err()
}
