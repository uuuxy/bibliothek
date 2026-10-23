package repository

// sitzungen.go — die Anweisungen zur Tabelle sitzungen: eine Zeile je Anmeldung, an ihr hängen
// die Sperre nach Inaktivität und die Zeit der letzten Passworteingabe. Was eine fehlende Zeile
// bedeutet, entscheidet auth/; die
// Lesefunktionen reichen pgx.ErrNoRows deshalb durch.

import (
	"context"
	"time"
)

// LegeSitzungAn legt die Zeile einer neuen Anmeldung an und liefert ihre Kennung.
func LegeSitzungAn(ctx context.Context, db Datenbank, benutzerID, pruefwert string, laeuftAb time.Time) (string, error) {
	var id string
	err := db.QueryRow(ctx, `
		INSERT INTO sitzungen (benutzer_id, passwort_pruefwert, laeuft_ab)
		VALUES ($1, $2, $3)
		RETURNING id::text
	`, benutzerID, pruefwert, laeuftAb).Scan(&id)
	return id, err
}

// LadeSitzungsZustand liefert, ob die Anmeldung nach Inaktivität gesperrt ist und wann ihr
// Passwort zuletzt geprüft wurde. Wie lange das her sein darf, entscheidet auth/.
func LadeSitzungsZustand(ctx context.Context, db Datenbank, sitzungID string) (gesperrt bool, passwortGeprueftAm time.Time, err error) {
	err = db.QueryRow(ctx, `
		SELECT gesperrt_seit IS NOT NULL, passwort_geprueft_am FROM sitzungen WHERE id = $1
	`, sitzungID).Scan(&gesperrt, &passwortGeprueftAm)
	return gesperrt, passwortGeprueftAm, err
}

// SperreSitzung sperrt die Anmeldung und nennt die Zahl der getroffenen Zeilen. Eine schon
// gesperrte Anmeldung behält ihren Zeitpunkt.
func SperreSitzung(ctx context.Context, db Datenbank, sitzungID string) (int64, error) {
	tag, err := db.Exec(ctx, `
		UPDATE sitzungen SET gesperrt_seit = COALESCE(gesperrt_seit, NOW()) WHERE id = $1
	`, sitzungID)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// EntsperreSitzung hebt die Sperre auf und vermerkt das eben geprüfte Passwort. Keine Zeile ist
// kein Fehler: Dann war nichts gesperrt.
func EntsperreSitzung(ctx context.Context, db Datenbank, sitzungID string) error {
	_, err := db.Exec(ctx, `
		UPDATE sitzungen SET gesperrt_seit = NULL, passwort_geprueft_am = NOW() WHERE id = $1
	`, sitzungID)
	return err
}

// LadeSitzungsPruefwert liefert den Prüfwert des Passworts, mit dem die Anmeldung entstand.
func LadeSitzungsPruefwert(ctx context.Context, db Datenbank, sitzungID string) (string, error) {
	var pruefwert string
	err := db.QueryRow(ctx, `
		SELECT passwort_pruefwert FROM sitzungen WHERE id = $1
	`, sitzungID).Scan(&pruefwert)
	return pruefwert, err
}

// SetzeSitzungsPruefwert schreibt den Prüfwert der Anmeldung neu.
func SetzeSitzungsPruefwert(ctx context.Context, db Datenbank, sitzungID, pruefwert string) error {
	_, err := db.Exec(ctx, `
		UPDATE sitzungen SET passwort_pruefwert = $2 WHERE id = $1
	`, sitzungID, pruefwert)
	return err
}

// VerlaengereSitzung schiebt das Ende der Zeile hinaus und nennt die Zahl der getroffenen
// Zeilen.
func VerlaengereSitzung(ctx context.Context, db Datenbank, sitzungID string, laeuftAb time.Time) (int64, error) {
	tag, err := db.Exec(ctx, `
		UPDATE sitzungen SET laeuft_ab = $2 WHERE id = $1
	`, sitzungID, laeuftAb)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// LoescheSitzung löscht die Zeile der Anmeldung und mit ihr den Prüfwert.
func LoescheSitzung(ctx context.Context, db Datenbank, sitzungID string) error {
	_, err := db.Exec(ctx, `DELETE FROM sitzungen WHERE id = $1`, sitzungID)
	return err
}

// LoescheAbgelaufeneSitzungen löscht die Zeilen, deren Token abgelaufen ist. Eine gesperrte
// Zeile wird nicht geschont: Ihr Token gilt nicht mehr, aufzuschließen gibt es nichts.
func LoescheAbgelaufeneSitzungen(ctx context.Context, db Datenbank) error {
	_, err := db.Exec(ctx, `DELETE FROM sitzungen WHERE laeuft_ab < NOW()`)
	return err
}
