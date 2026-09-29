package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// kontoLoeschung beschreibt ein Zugangskonto unmittelbar vor seiner Löschung und die Tür,
// über die es fällt.
type kontoLoeschung struct {
	kontoID, vorname, nachname, email, rolle string
	leserID                                  *string
	bearbeiterID                             string // "" = der Lauf selbst (Akteur SYSTEM)
	kontext                                  *string
	// Nur Benutzer & Rechte: ob die unberührte Leserzeile mitging, und sonst warum nicht
	// (loescheUnberuehrteLeserzeile).
	ueberBenutzerverwaltung bool
	leserzeileGeloescht     bool
	leserzeileBleibtWegen   string
}

// protokolliereKontoLoeschung schreibt den Löscheintrag eines Zugangskontos in die
// Datensatz-Historie. Ein Konto fällt über zwei Türen: Benutzer & Rechte (DeleteUser) und das
// Löschen der Leserzeile eines Kollegen (DeleteStudent). Bis zum 29.09.2026 schrieb nur die
// erste einen Eintrag mit dem Konto, und keiner nannte den Leser. Nach dem Löschen führt kein
// Fremdschlüssel mehr vom Leser zum Konto (benutzer.leser_id und alle Verweise auf benutzer
// stehen auf SET NULL); dieser Eintrag ist der Weg dorthin.
//
// Die Leserkennung steht unter schueler_id wie in jedem anderen Protokolleintrag über einen
// Leser: Darüber findet die Auskunft das frühere Konto (LeseDsgvoFruehereZugangskonten), und
// darüber nimmt die Tilgung beim endgültigen Löschen des Lesers Name und Adresse heraus
// (protokollSchluesselMitPersonenbezug). Jeder Schlüssel steht wörtlich hier, damit das Gate
// in protokoll_personenbezug_test.go ihn sieht.
func (r *pgAuditRepository) protokolliereKontoLoeschung(ctx context.Context, tx pgx.Tx, l kontoLoeschung) error {
	details := map[string]any{
		"vorname":  l.vorname,
		"nachname": l.nachname,
		"email":    l.email,
		"rolle":    l.rolle,
	}
	if l.leserID != nil && *l.leserID != "" {
		details["schueler_id"] = *l.leserID
	}
	if l.ueberBenutzerverwaltung {
		details["leserzeile_geloescht"] = l.leserzeileGeloescht
		details["leserzeile_bleibt_wegen"] = l.leserzeileBleibtWegen
	}

	akteur := "SYSTEM"
	var bearbeiter *string
	if l.bearbeiterID != "" {
		akteur = "USER"
		bearbeiter = &l.bearbeiterID
	}
	if err := r.insertAuditLog(ctx, tx, auditEntry{
		Tabelle: "benutzer", Aktion: "DELETE", DatensatzID: l.kontoID,
		BearbeiterID: bearbeiter, Akteur: akteur, Kontext: l.kontext,
		Details: details,
	}); err != nil {
		return fmt.Errorf("löschung des kontos protokollieren: %w", err)
	}
	return nil
}

// loescheKontenDerLeserzeile löscht die Zugangskonten, die auf eine Leserzeile zeigen, und
// schreibt je Konto den Löscheintrag — die Tür „Leserzeile eines Kollegen in den Papierkorb"
// (DeleteStudent). Bis zum 29.09.2026 stand dort nur die Zahl der gelöschten Konten.
func (r *pgAuditRepository) loescheKontenDerLeserzeile(ctx context.Context, tx pgx.Tx, leserID, bearbeiterID string) (int64, error) {
	rows, err := tx.Query(ctx, `
		SELECT id::text, coalesce(vorname, ''), coalesce(nachname, ''), coalesce(email, ''), coalesce(rolle::text, '')
		FROM benutzer WHERE leser_id = $1 FOR UPDATE`, leserID)
	if err != nil {
		return 0, fmt.Errorf("konten der leserzeile lesen: %w", err)
	}
	var konten []kontoLoeschung
	for rows.Next() {
		var k kontoLoeschung
		if err := rows.Scan(&k.kontoID, &k.vorname, &k.nachname, &k.email, &k.rolle); err != nil {
			rows.Close()
			return 0, fmt.Errorf("konto der leserzeile lesen: %w", err)
		}
		konten = append(konten, k)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("konten der leserzeile lesen: %w", err)
	}

	kontext := "Konto mit der Leserzeile gelöscht (Papierkorb)"
	var geloescht int64
	for _, k := range konten {
		tag, err := tx.Exec(ctx, `DELETE FROM benutzer WHERE id = $1`, k.kontoID)
		if err != nil {
			return geloescht, fmt.Errorf("deleting account of reader: %w", err)
		}
		// Null Zeilen: Das Konto war zwischen Lesen und Löschen schon weg (FOR UPDATE hält das
		// in derselben Transaktion eigentlich fern). Dann gibt es auch nichts zu protokollieren.
		if tag.RowsAffected() == 0 {
			continue
		}
		geloescht++
		leser := leserID
		k.leserID, k.bearbeiterID, k.kontext = &leser, bearbeiterID, &kontext
		if err := r.protokolliereKontoLoeschung(ctx, tx, k); err != nil {
			return geloescht, err
		}
	}
	return geloescht, nil
}
