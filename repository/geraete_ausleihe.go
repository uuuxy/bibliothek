package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// LiesGeraetNachNummer liefert das Gerät mit dieser Nummer. Trägt keines sie, kommt
// pgx.ErrNoRows: Was das an der Theke heißt, sagt der Aufrufer.
func LiesGeraetNachNummer(ctx context.Context, db DBQueryer, nummer string) (Geraet, error) {
	var g Geraet
	err := db.QueryRow(ctx, `
		SELECT id, modellname, seriennummer, barcode_id, zubehoer, ist_ausleihbar, ist_ausgesondert, zustand_notiz
		FROM geraete
		WHERE barcode_id = $1
	`, nummer).Scan(&g.ID, &g.Modellname, &g.Seriennummer, &g.BarcodeID, &g.Zubehoer, &g.IstAusleihbar, &g.IstAusgesondert, &g.ZustandNotiz)
	return g, err
}

// SperreOffeneGeraeteAusleihe liefert die offene Ausleihe des Geräts und sperrt ihre Zeile bis
// zum Ende der Transaktion: Zwei Rückgaben desselben Geräts warten so aufeinander. offen ist
// false, wenn das Gerät frei ist. Sie nimmt eine Transaktion und keinen Pool: Am Pool endete
// die Sperre mit der Anweisung.
func SperreOffeneGeraeteAusleihe(ctx context.Context, tx pgx.Tx, geraetID string) (ausleihe Loan, offen bool, err error) {
	err = tx.QueryRow(ctx, `
		SELECT id, geraet_id, schueler_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am, bearbeiter_id, ist_fremdrueckgabe, ist_handapparat
		FROM ausleihen
		WHERE geraet_id = $1 AND rueckgabe_am IS NULL
		FOR UPDATE
	`, geraetID).Scan(
		&ausleihe.ID, &ausleihe.GeraetID, &ausleihe.SchuelerID,
		&ausleihe.AusgeliehenAm, &ausleihe.RueckgabeFrist, &ausleihe.RueckgabeAm,
		&ausleihe.BearbeiterID, &ausleihe.IstFremdrueckgabe, &ausleihe.IstHandapparat,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ausleihe, false, nil
	}
	if err != nil {
		return ausleihe, false, err
	}
	return ausleihe, true, nil
}

// GeraeteAusleihe trägt die Angaben einer neuen Ausleihe eines Geräts. Als Struktur, weil
// Gerät, Leser und Bearbeiter drei Kennungen vom selben Typ sind.
type GeraeteAusleihe struct {
	GeraetID       string
	LeserID        string
	BearbeiterID   string
	RueckgabeFrist time.Time
	IstDauerleihe  bool
}

// LeiheGeraetAus schreibt die Ausleihe eines Geräts und liefert ihre Kennung. Ist das Gerät
// schon verliehen, scheitert sie an uniq_ausleihen_aktiv_geraet; die Meldung dazu formuliert
// der Aufrufer.
func LeiheGeraetAus(ctx context.Context, tx DBQueryer, a GeraeteAusleihe) (string, error) {
	var id string
	err := tx.QueryRow(ctx, `
		INSERT INTO ausleihen (geraet_id, schueler_id, rueckgabe_frist, bearbeiter_id, ist_handapparat)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, a.GeraetID, a.LeserID, a.RueckgabeFrist, a.BearbeiterID, a.IstDauerleihe).Scan(&id)
	return id, err
}

// ErrGeraeteAusleiheFehlt meldet, dass es die Ausleihe nicht gibt, deren Rückgabe gebucht
// werden sollte.
var ErrGeraeteAusleiheFehlt = errors.New("ausleihe des geräts nicht gefunden")

// BucheGeraeteRueckgabe trägt an der Ausleihe eines Geräts die Rückgabe ein: jetzt, durch
// diesen Bearbeiter, und ob ein anderer als der Ausleiher das Gerät brachte.
func BucheGeraeteRueckgabe(ctx context.Context, tx DBQueryer, ausleiheID, bearbeiterID string, fremdrueckgabe bool) error {
	tag, err := tx.Exec(ctx, `
		UPDATE ausleihen
		SET rueckgabe_am = CURRENT_TIMESTAMP, rueckgabe_bearbeiter_id = $1, ist_fremdrueckgabe = $2
		WHERE id = $3
	`, bearbeiterID, fremdrueckgabe, ausleiheID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrGeraeteAusleiheFehlt
	}
	return nil
}

// LeserNameUndKlasse liefert einen Leser, der nicht gelöscht ist, mit Vorname, Nachname und
// Klasse und sonst nichts; fehlt er, kommt pgx.ErrNoRows. Gelesen wird leser und nicht die
// Sicht schueler: Der Gesuchte kann ein Kollege sein.
func LeserNameUndKlasse(ctx context.Context, db DBQueryer, leserID string) (Student, error) {
	var l Student
	err := db.QueryRow(ctx,
		"SELECT vorname, nachname, coalesce(klasse, '') FROM leser WHERE id = $1 AND deleted_at IS NULL",
		leserID).Scan(&l.Vorname, &l.Nachname, &l.Klasse)
	return l, err
}
