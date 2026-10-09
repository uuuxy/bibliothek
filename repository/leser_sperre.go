package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// LeserSperrStand ist der Zustand eines Lesers, bevor die Sperre umgeschaltet wird: seine
// Art, ob er im Papierkorb liegt oder anonymisiert ist, die Sperre des Programms
// (ist_gesperrt), die von Hand und ihr Grund.
type LeserSperrStand struct {
	Art                     string
	Geloescht, Anonymisiert bool
	VomProgramm, VonHand    bool
	Grund                   string
}

// SperreLeserZeileMitStand liest den Sperrstand eines Lesers und hält seine Zeile bis zum Ende
// der Transaktion gesperrt. Gelesen wird leser, nicht die Sicht schueler: Die Akte eines
// Kollegen ruft die Tür auch. pgx.ErrNoRows, wenn es den Leser nicht gibt.
func SperreLeserZeileMitStand(ctx context.Context, tx pgx.Tx, id string) (LeserSperrStand, error) {
	var st LeserSperrStand
	err := tx.QueryRow(ctx, `
		SELECT art, deleted_at IS NOT NULL, anonymized_at IS NOT NULL,
		       ist_gesperrt, coalesce(is_manually_blocked, false), coalesce(block_reason, '')
		FROM leser WHERE id = $1 FOR UPDATE`, id).Scan(
		&st.Art, &st.Geloescht, &st.Anonymisiert, &st.VomProgramm, &st.VonHand, &st.Grund)
	return st, err
}

// LeserNachSperre ist der Leser, wie er nach dem Umschalten der Sperre in der Zeile steht.
type LeserNachSperre struct {
	ID       string
	Vorname  string
	Nachname string
	Klasse   string
	VonHand  bool
	Gesperrt bool
}

// SetzeSperreVonHand setzt die Sperre von Hand samt Grund oder nimmt beide Sperren und den
// Grund weg; chk_schueler_block_reason verlangt den Grund nur, solange eine Sperre besteht.
// Die Klasse kommt als leerer Text, wenn der Leser keine hat (ein Kollege).
func SetzeSperreVonHand(ctx context.Context, db DBQueryer, id string, sperren bool, grund string) (LeserNachSperre, error) {
	var l LeserNachSperre
	err := db.QueryRow(ctx, `
		UPDATE leser
		SET is_manually_blocked = $1,
		    ist_gesperrt = ist_gesperrt AND $1,
		    block_reason = CASE WHEN $1 THEN $3 ELSE NULL END,
		    aktualisiert_am = CURRENT_TIMESTAMP
		WHERE id = $2
		RETURNING id, vorname, nachname, coalesce(klasse, ''), is_manually_blocked, ist_gesperrt`,
		sperren, id, grund).Scan(
		&l.ID, &l.Vorname, &l.Nachname,
		&l.Klasse, &l.VonHand, &l.Gesperrt)
	return l, err
}
