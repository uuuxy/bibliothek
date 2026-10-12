package testdaten

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"bibliothek/pkg/lmf"
)

// TitelMitMeldebestand legt einen Titel mit Meldebestand an. Trägt der Titel die Kennung der
// Lernmittel („LMF-Mathe 7“), entsteht er als Lernmittel.
func TitelMitMeldebestand(t *testing.T, pool *pgxpool.Pool, titel string, meldebestand int) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO buecher_titel (titel, meldebestand, ist_lernmittel) VALUES ($1, $2, $3) RETURNING id`,
		titel, meldebestand, lmf.HatKennung(titel)).Scan(&id); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	return id
}

// TitelMitSignatur legt einen Titel mit Signatur an; Lernmittel, wenn Titel oder Signatur die
// Kennung der Lernmittel tragen.
func TitelMitSignatur(t *testing.T, pool *pgxpool.Pool, titel, signatur string, meldebestand int) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO buecher_titel (titel, signatur, meldebestand, ist_lernmittel) VALUES ($1, $2, $3, $4) RETURNING id`,
		titel, signatur, meldebestand, lmf.HatKennung(titel) || lmf.HatKennung(signatur)).Scan(&id); err != nil {
		t.Fatalf("Titel mit Signatur anlegen: %v", err)
	}
	return id
}

// SeedMonitorTitel legt einen Titel mit Signatur an, wahlweise mit Cover, angelegt vor alterTage
// Tagen: Das Datum steuert „Neu eingetroffen“ am Monitor, das Cover seine Cover-Pflicht.
func SeedMonitorTitel(t *testing.T, pool *pgxpool.Pool, titel, signatur string, mitCover bool, alterTage int) string {
	t.Helper()
	var cover *string
	if mitCover {
		url := "https://cover.invalid/" + titel
		cover = &url
	}
	var id string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO buecher_titel (titel, signatur, cover_url, erstellt_am, ist_lernmittel)
		VALUES ($1, $2, $3, NOW() - make_interval(days => $4), $5)
		RETURNING id`, titel, signatur, cover, alterTage, lmf.HatKennung(signatur)).Scan(&id); err != nil {
		t.Fatalf("Titel %q anlegen: %v", titel, err)
	}
	return id
}

// SeedSignaturMitExemplar legt einen Titel unter der Signatur mit einem ausleihbaren Exemplar an
// und liefert die Kennung des Exemplars.
func SeedSignaturMitExemplar(t *testing.T, pool *pgxpool.Pool, signatur, barcode string) string {
	t.Helper()
	var titelID string
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO buecher_titel (titel, signatur) VALUES ($1, $2) RETURNING id`,
		signatur+"-Buch", signatur).Scan(&titelID); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	return Exemplar(t, pool, titelID, barcode, true, "")
}

// Exemplar legt ein Exemplar mit Barcode, Verleihstatus und Notiz an.
func Exemplar(t *testing.T, pool *pgxpool.Pool, titelID, barcode string, ausleihbar bool, notiz string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, zustand_notiz)
		 VALUES ($1, $2, $3, $4) RETURNING id`,
		titelID, barcode, ausleihbar, notiz).Scan(&id); err != nil {
		t.Fatalf("Exemplar %q anlegen: %v", barcode, err)
	}
	return id
}

// ZulaufExemplar legt ein Exemplar an, wie es das Bestellwesen anlegt: nicht ausleihbar, mit
// Bestellstatus und dem heutigen Tag als erworben_am (Vorgabe der Spalte).
func ZulaufExemplar(t *testing.T, pool *pgxpool.Pool, titelID, barcode string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, bestellstatus)
		VALUES ($1, $2, false, 'im_zulauf') RETURNING id`, titelID, barcode).Scan(&id); err != nil {
		t.Fatalf("Zulauf-Exemplar %q: %v", barcode, err)
	}
	return id
}

// AusleihbaresExemplar legt Titel und Exemplar an, das sofort ausgeliehen werden kann.
func AusleihbaresExemplar(t *testing.T, pool *pgxpool.Pool, titel, barcode string) {
	t.Helper()
	titelID := TitelMitMeldebestand(t, pool, titel, 0)
	Exemplar(t, pool, titelID, barcode, true, "")
}
