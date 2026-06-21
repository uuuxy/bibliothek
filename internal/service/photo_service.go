package service

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"bibliothek/db"
	"bibliothek/internal/crypto"
	"bibliothek/pkg/imageutil"

	"github.com/jackc/pgx/v5"
)

// UploadStudentPhoto verarbeitet den Base64-String eines Fotos, konvertiert ihn zu WebP,
// verschlüsselt ihn per AES und speichert ihn in der Datenbank ab.
func UploadStudentPhoto(ctx context.Context, dbPool db.PgxPoolIface, studentID string, base64DataStr string) (string, error) {
	// 1. Resolve student's barcode ID from database
	var barcodeID string
	err := dbPool.QueryRow(ctx, "SELECT barcode_id FROM schueler WHERE id = $1", studentID).Scan(&barcodeID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", errors.New("schüler nicht gefunden")
		}
		return "", err
	}

	// 2. Decode base64 image data
	base64Data := strings.TrimPrefix(base64DataStr, "data:image/jpeg;base64,")
	base64Data = strings.TrimPrefix(base64Data, "data:image/png;base64,")
	
	imgBytes, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return "", fmt.Errorf("ungültiges base64-format: %w", err)
	}

	// 3. Konvertierung nach WebP
	webpBytes, err := imageutil.ConvertToWebP(imgBytes, 80)
	if err != nil {
		return "", err
	}

	// 4. Verschlüsseln der WebP-Bytes
	encryptedData, err := crypto.Encrypt(webpBytes)
	if err != nil {
		return "", fmt.Errorf("fehler bei der fotostrukturierung: %w", err)
	}

	// 5. In der Datenbank abspeichern (Upsert in schueler_fotos)
	query := `
		INSERT INTO schueler_fotos (schueler_id, foto_encrypted)
		VALUES ($1, $2)
		ON CONFLICT (schueler_id) DO UPDATE SET 
			foto_encrypted = EXCLUDED.foto_encrypted,
			aktualisiert_am = CURRENT_TIMESTAMP
	`
	_, err = dbPool.Exec(ctx, query, studentID, encryptedData)
	if err != nil {
		return "", fmt.Errorf("fehler beim speichern des fotos in der db: %w", err)
	}

	photoURL := fmt.Sprintf("/api/schueler/%s/photo", barcodeID)
	return photoURL, nil
}
