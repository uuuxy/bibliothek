package repository

import "context"

// LadeFotoZurAusweisnummer liefert das verschlüsselte Passbild des Lesers mit dieser
// Ausweisnummer, oder pgx.ErrNoRows, wenn es keins gibt.
//
// Gelesen wird über die Tabelle leser, nicht über die Sicht schueler: Das Passbild hängt an
// der Kennung des Lesers, und ein Kollege steht nicht in der Sicht. Ein Leser im Papierkorb
// gibt sein Bild nicht mehr heraus, sonst würde aus dem Papierkorb ein Archiv; die Zeile in
// schueler_fotos bleibt, damit das Bild mit ihm zurückkommt.
func LadeFotoZurAusweisnummer(ctx context.Context, db DBQueryer, ausweisnummer string) ([]byte, error) {
	query := `
			SELECT sf.foto_encrypted 
			FROM schueler_fotos sf
			JOIN leser s ON s.id = sf.schueler_id
			WHERE s.barcode_id = $1 AND s.deleted_at IS NULL
		`

	var ciphertext []byte
	err := db.QueryRow(ctx, query, ausweisnummer).Scan(&ciphertext)
	return ciphertext, err
}

// AusweisnummerDesLesers liefert die Ausweisnummer zur Kennung eines Lesers, leer, wenn er
// noch keine hat, oder pgx.ErrNoRows, wenn es den Leser nicht gibt. Gelesen wird die Tabelle
// leser und nicht die Sicht schueler: Ein Passbild kann jeder Leser bekommen.
func AusweisnummerDesLesers(ctx context.Context, db DBQueryer, leserID string) (string, error) {
	var ausweisnummer string
	err := db.QueryRow(ctx, "SELECT COALESCE(barcode_id, '') FROM leser WHERE id = $1", leserID).Scan(&ausweisnummer)
	return ausweisnummer, err
}

// SpeichereFoto legt das verschlüsselte Passbild eines Lesers ab oder ersetzt das vorhandene.
func SpeichereFoto(ctx context.Context, db DBQueryer, leserID string, verschluesselt []byte) error {
	query := `
		INSERT INTO schueler_fotos (schueler_id, foto_encrypted)
		VALUES ($1, $2)
		ON CONFLICT (schueler_id) DO UPDATE SET 
			foto_encrypted = EXCLUDED.foto_encrypted,
			aktualisiert_am = CURRENT_TIMESTAMP
	`
	_, err := db.Exec(ctx, query, leserID, verschluesselt)
	return err
}
