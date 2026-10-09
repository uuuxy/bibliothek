package repository

import "context"

// LeserVorhanden sagt, ob es die Leserzeile gibt. Gefragt wird die Tabelle leser, nicht die
// Sicht schueler: Die zeigt nur Schüler.
func LeserVorhanden(ctx context.Context, db DBQueryer, id string) (bool, error) {
	var vorhanden bool
	err := db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM leser WHERE id = $1)", id).Scan(&vorhanden)
	return vorhanden, err
}

// LeserHatOffeneAusleihen sagt, ob der Leser noch etwas entliehen hat.
func LeserHatOffeneAusleihen(ctx context.Context, db DBQueryer, id string) (bool, error) {
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

// LeserHatUnbezahlteForderungen sagt, ob am Leser ein unbezahlter Schadensfall steht.
func LeserHatUnbezahlteForderungen(ctx context.Context, db DBQueryer, id string) (bool, error) {
	var offen bool
	err := db.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM schadensfaelle WHERE schueler_id = $1 AND ist_bezahlt = false)
	`, id).Scan(&offen)
	return offen, err
}

// LusdIDDesLesers liest die LUSD-ID eines Lesers, leer wenn keine gesetzt ist; pgx.ErrNoRows,
// wenn es den Leser nicht gibt.
func LusdIDDesLesers(ctx context.Context, db DBQueryer, id string) (string, error) {
	var lusdID string
	err := db.QueryRow(ctx, "SELECT COALESCE(lusd_id, '') FROM leser WHERE id = $1", id).Scan(&lusdID)
	return lusdID, err
}

// LusdIDBeiAnderemSchueler sagt, ob ein anderer aktiver Schüler die LUSD-ID trägt. Gefragt
// wird die Sicht schueler: chk_leser_nur_schueler_werden_abgaenger lässt eine LUSD-ID nur an
// einem Schüler zu, Sicht und Tabelle liefern hier dasselbe.
func LusdIDBeiAnderemSchueler(ctx context.Context, db DBQueryer, lusdID, eigeneID string) (bool, error) {
	var belegt bool
	err := db.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM schueler WHERE lusd_id = $1 AND deleted_at IS NULL AND id <> $2)", lusdID, eigeneID).Scan(&belegt)
	return belegt, err
}
