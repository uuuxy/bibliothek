package repository

import "context"

// IstLeserVorhanden sagt, ob es die Leserzeile gibt. Gefragt wird die Tabelle leser, nicht die
// Sicht schueler: Die zeigt nur Schüler.
func IstLeserVorhanden(ctx context.Context, db Datenbank, id string) (bool, error) {
	var vorhanden bool
	err := db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM leser WHERE id = $1)", id).Scan(&vorhanden)
	return vorhanden, err
}

// HatLeserOffeneAusleihen sagt, ob der Leser noch etwas entliehen hat.
func HatLeserOffeneAusleihen(ctx context.Context, db Datenbank, id string) (bool, error) {
	var offen bool
	err := db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM ausleihen
			WHERE schueler_id = $1 AND rueckgabe_am IS NULL
		)
	`, id).Scan(&offen)
	return offen, err
}

// HatLeserUnbezahlteForderungen sagt, ob am Leser ein unbezahlter Schadensfall steht.
func HatLeserUnbezahlteForderungen(ctx context.Context, db Datenbank, id string) (bool, error) {
	var offen bool
	err := db.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM schadensfaelle WHERE schueler_id = $1 AND ist_bezahlt = false)
	`, id).Scan(&offen)
	return offen, err
}

// LadeLUSDIDDesLesers liest die LUSD-ID eines Lesers, leer wenn keine gesetzt ist; pgx.ErrNoRows,
// wenn es den Leser nicht gibt.
func LadeLUSDIDDesLesers(ctx context.Context, db Datenbank, id string) (string, error) {
	var lusdID string
	err := db.QueryRow(ctx, "SELECT COALESCE(lusd_id, '') FROM leser WHERE id = $1", id).Scan(&lusdID)
	return lusdID, err
}

// HatAndererSchuelerLUSDID sagt, ob ein anderer aktiver Schüler die LUSD-ID trägt. Gefragt
// wird die Sicht schueler: chk_leser_nur_schueler_werden_abgaenger lässt eine LUSD-ID nur an
// einem Schüler zu, Sicht und Tabelle liefern hier dasselbe.
func HatAndererSchuelerLUSDID(ctx context.Context, db Datenbank, lusdID, eigeneID string) (bool, error) {
	var belegt bool
	err := db.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM schueler WHERE lusd_id = $1 AND deleted_at IS NULL AND id <> $2)", lusdID, eigeneID).Scan(&belegt)
	return belegt, err
}
