package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// LadeTitel liest einen Titel samt Metadaten. Fehlt er, kommt ErrTitelNichtGefunden zurück, ein
// Bedienfehler, kein Fehler des Servers. Jeder andere Fehler (Verbindung, Scan einer Spalte mit
// NULL in einen String) geht weiter: Hieße auch er „buch nicht gefunden", wäre ein
// Auseinanderlaufen des Schemas als 404 verkleidet.
func (a *TitelAbfragen) LadeTitel(ctx context.Context, id string) (*KatalogTitel, error) {
	abfrage := `
		SELECT id, COALESCE(isbn, '') AS isbn, titel AS title, COALESCE(autor, '') AS author, COALESCE(signatur, '') AS signatur, COALESCE(cover_url, '') AS cover_url, COALESCE(subject, '') AS subject, COALESCE(track, '') AS track,
		       (SELECT COUNT(*)::int FROM buecher_exemplare e WHERE e.titel_id = buecher_titel.id AND e.ist_ausgesondert = false) AS stock,
		       TO_CHAR(last_counted, 'YYYY-MM-DD') as last_counted, sort_order, COALESCE(medientyp, 'Buch') AS medientyp, COALESCE(jahrgang_von, 0) AS jahrgang_von, COALESCE(jahrgang_bis, 0) AS jahrgang_bis, erweiterte_eigenschaften, COALESCE(auflage, '') AS auflage
		FROM buecher_titel
		WHERE id = $1::uuid`

	var eintrag KatalogTitel
	err := a.pool.QueryRow(ctx, abfrage, id).Scan(
		&eintrag.ID,
		&eintrag.ISBN,
		&eintrag.Titel,
		&eintrag.Autor,
		&eintrag.Signatur,
		&eintrag.CoverURL,
		&eintrag.Fach,
		&eintrag.Zweig,
		&eintrag.Bestand,
		&eintrag.ZuletztGezaehlt,
		&eintrag.Reihenfolge,
		&eintrag.Medientyp,
		&eintrag.JahrgangVon,
		&eintrag.JahrgangBis,
		&eintrag.ErweiterteEigenschaften,
		&eintrag.Auflage,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTitelNichtGefunden
	}
	if err != nil {
		return nil, fmt.Errorf("buch laden: %w", err)
	}
	return &eintrag, nil
}

// AendereTitelMetadaten schreibt die bibliografischen Angaben eines Titels — der Weg, über
// den ein DNB-Treffer im Katalog landet.
func (a *TitelAbfragen) AendereTitelMetadaten(ctx context.Context, id string, titelText, autor, coverURL string) error {
	abfrage := `
		UPDATE buecher_titel
		SET titel = COALESCE(NULLIF($1, ''), titel),
		    autor = COALESCE(NULLIF($2, ''), autor),
		    cover_url = COALESCE(NULLIF($3, ''), cover_url),
		    -- Wer ein Cover setzt (Hand-Upload, DNB-Übernahme), hat es gefunden. Ohne das
		    -- blieb ein neuer Titel auf 'PENDING', und der Cover-Sync überschrieb das von
		    -- Hand hochgeladene Cover beim nächsten Lauf (Bestands-Durchgang 10.09.2026).
		    cover_status = CASE WHEN NULLIF($3, '') IS NOT NULL THEN 'FOUND' ELSE cover_status END
		WHERE id = $4::uuid`

	ergebnis, err := a.pool.Exec(ctx, abfrage, titelText, autor, coverURL, id)
	if err != nil {
		return fmt.Errorf("metadaten konnten nicht aktualisiert werden: %w", err)
	}
	if ergebnis.RowsAffected() == 0 {
		return ErrTitelNichtGefunden
	}
	return nil
}
