package repository

// sitzungen.go — die Anweisungen zur Tabelle sitzungen: eine Zeile je Anmeldung, an ihr hängt
// die Sperre nach Inaktivität. Was eine fehlende Zeile bedeutet, entscheidet auth/; die
// Lesefunktionen reichen pgx.ErrNoRows deshalb durch.

import (
	"context"
	"time"
)

// LegeSitzungAn legt die Zeile einer neuen Anmeldung an und liefert ihre Kennung.
func LegeSitzungAn(ctx context.Context, db DBQueryer, benutzerID, pruefwert string, laeuftAb time.Time) (string, error) {
	var id string
	err := db.QueryRow(ctx, `
		INSERT INTO sitzungen (benutzer_id, passwort_pruefwert, laeuft_ab)
		VALUES ($1, $2, $3)
		RETURNING id::text
	`, benutzerID, pruefwert, laeuftAb).Scan(&id)
	return id, err
}

// SitzungGesperrt sagt, ob die Anmeldung gesperrt ist.
func SitzungGesperrt(ctx context.Context, db DBQueryer, sitzungID string) (bool, error) {
	var gesperrt bool
	err := db.QueryRow(ctx, `
		SELECT gesperrt_seit IS NOT NULL FROM sitzungen WHERE id = $1
	`, sitzungID).Scan(&gesperrt)
	return gesperrt, err
}

// SperreSitzung sperrt die Anmeldung und nennt die Zahl der getroffenen Zeilen. Eine schon
// gesperrte Anmeldung behält ihren Zeitpunkt.
func SperreSitzung(ctx context.Context, db DBQueryer, sitzungID string) (int64, error) {
	tag, err := db.Exec(ctx, `
		UPDATE sitzungen SET gesperrt_seit = COALESCE(gesperrt_seit, NOW()) WHERE id = $1
	`, sitzungID)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// EntsperreSitzung hebt die Sperre auf. Keine Zeile ist kein Fehler: Dann war nichts gesperrt.
func EntsperreSitzung(ctx context.Context, db DBQueryer, sitzungID string) error {
	_, err := db.Exec(ctx, `UPDATE sitzungen SET gesperrt_seit = NULL WHERE id = $1`, sitzungID)
	return err
}

// SitzungsPruefwert liefert den Prüfwert des Passworts, mit dem die Anmeldung entstand.
func SitzungsPruefwert(ctx context.Context, db DBQueryer, sitzungID string) (string, error) {
	var pruefwert string
	err := db.QueryRow(ctx, `
		SELECT passwort_pruefwert FROM sitzungen WHERE id = $1
	`, sitzungID).Scan(&pruefwert)
	return pruefwert, err
}

// SetzeSitzungsPruefwert schreibt den Prüfwert der Anmeldung neu.
func SetzeSitzungsPruefwert(ctx context.Context, db DBQueryer, sitzungID, pruefwert string) error {
	_, err := db.Exec(ctx, `
		UPDATE sitzungen SET passwort_pruefwert = $2 WHERE id = $1
	`, sitzungID, pruefwert)
	return err
}

// VerlaengereSitzung schiebt das Ende der Zeile hinaus und nennt die Zahl der getroffenen
// Zeilen.
func VerlaengereSitzung(ctx context.Context, db DBQueryer, sitzungID string, laeuftAb time.Time) (int64, error) {
	tag, err := db.Exec(ctx, `
		UPDATE sitzungen SET laeuft_ab = $2 WHERE id = $1
	`, sitzungID, laeuftAb)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// LoescheSitzung löscht die Zeile der Anmeldung und mit ihr den Prüfwert.
func LoescheSitzung(ctx context.Context, db DBQueryer, sitzungID string) error {
	_, err := db.Exec(ctx, `DELETE FROM sitzungen WHERE id = $1`, sitzungID)
	return err
}

// LoescheAbgelaufeneSitzungen löscht die Zeilen, deren Token abgelaufen ist. Eine gesperrte
// Zeile wird nicht geschont: Ihr Token gilt nicht mehr, aufzuschließen gibt es nichts.
func LoescheAbgelaufeneSitzungen(ctx context.Context, db DBQueryer) error {
	_, err := db.Exec(ctx, `DELETE FROM sitzungen WHERE laeuft_ab < NOW()`)
	return err
}
