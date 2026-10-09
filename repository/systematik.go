package repository

import "context"

// LegeSachgruppeAn legt eine Sachgruppe an und liefert ihre Kennung. Ein vergebenes Kürzel und
// eine vergebene Bezeichnung lehnt die Datenbank ab.
func LegeSachgruppeAn(ctx context.Context, db DBQueryer, kuerzel, bezeichnung string) (string, error) {
	var id string
	err := db.QueryRow(ctx, `
			INSERT INTO systematik_kategorien (kuerzel, bezeichnung)
			VALUES ($1, $2)
			RETURNING id::text
		`, kuerzel, bezeichnung).Scan(&id)
	return id, err
}

// SachgruppenBezeichnung liest die Bezeichnung einer Sachgruppe; pgx.ErrNoRows, wenn es die
// Kennung nicht gibt.
func SachgruppenBezeichnung(ctx context.Context, db DBQueryer, id string) (string, error) {
	var bezeichnung string
	err := db.QueryRow(ctx,
		`SELECT bezeichnung FROM systematik_kategorien WHERE id = $1::uuid`, id).Scan(&bezeichnung)
	return bezeichnung, err
}

// ZaehleTitelDerSachgruppe zählt die Titel, deren Fach genau so heißt: die Zeilen, die der
// Fremdschlüssel beim Umbenennen der Sachgruppe mitzieht.
func ZaehleTitelDerSachgruppe(ctx context.Context, db DBQueryer, bezeichnung string) (int64, error) {
	var anzahl int64
	err := db.QueryRow(ctx,
		`SELECT count(*) FROM buecher_titel WHERE subject = $1`, bezeichnung).Scan(&anzahl)
	return anzahl, err
}

// ZaehleTitelMitFach zählt die Titel, deren Fach ohne Leerraum am Rand so heißt: die Prüfung,
// ob eine Sachgruppe noch an Büchern hängt.
func ZaehleTitelMitFach(ctx context.Context, db DBQueryer, bezeichnung string) (int, error) {
	var anzahl int
	err := db.QueryRow(ctx,
		`SELECT count(*) FROM buecher_titel WHERE btrim(COALESCE(subject, '')) = btrim($1)`,
		bezeichnung).Scan(&anzahl)
	return anzahl, err
}

// AendereSachgruppe setzt Kürzel und Bezeichnung einer Sachgruppe. Die Titel zieht der
// Fremdschlüssel mit, die Signaturen bleiben stehen.
func AendereSachgruppe(ctx context.Context, db DBQueryer, id, kuerzel, bezeichnung string) error {
	_, err := db.Exec(ctx, `
		UPDATE systematik_kategorien
		SET kuerzel = $2, bezeichnung = $3
		WHERE id = $1::uuid
	`, id, kuerzel, bezeichnung)
	return err
}

// BenenneFachOffenerInventurenUm stellt die laufenden Inventuren mit Filter auf ein Fach auf
// dessen neuen Namen um; der Filter ist Text und folgt keinem Fremdschlüssel. Abgeschlossene
// Inventuren bleiben stehen.
func BenenneFachOffenerInventurenUm(ctx context.Context, db DBQueryer, alt, neu string) error {
	_, err := db.Exec(ctx, `
			UPDATE inventur_sessions SET scope_subject = $2
			WHERE abgeschlossen_am IS NULL AND scope_subject = $1
		`, alt, neu)
	return err
}

// LoescheSachgruppe löscht eine Sachgruppe. Hängt ein Titel an ihr, lehnt der Fremdschlüssel
// ab.
func LoescheSachgruppe(ctx context.Context, db DBQueryer, id string) error {
	_, err := db.Exec(ctx,
		`DELETE FROM systematik_kategorien WHERE id = $1::uuid`, id)
	return err
}
