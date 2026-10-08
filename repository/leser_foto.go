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
