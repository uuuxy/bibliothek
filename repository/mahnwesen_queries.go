package repository

import (
	"context"
	"strings"
	"time"

	"bibliothek/pkg/schulzeit"

	"github.com/jackc/pgx/v5"
)

// klassenGrouper sammelt Zeilen zu Klassen→Schüler→Medien. Bewusst index-basiert:
// Pointer in Slice-Elemente werden bei append-Reallokationen ungültig und haben
// Medien gleichnamiger Schüler still verschluckt.
type klassenGrouper struct {
	klassen     []MahnwesenKlasse
	klassenIdx  map[string]int
	schuelerIdx map[string]int
}

func newKlassenGrouper() *klassenGrouper {
	return &klassenGrouper{
		klassen:     make([]MahnwesenKlasse, 0),
		klassenIdx:  map[string]int{},
		schuelerIdx: map[string]int{},
	}
}

// add ordnet ein Medium seiner Gruppe zu. Ehemalige bilden eine Gruppe für sich, gleich
// welche Klasse sie zuletzt trugen: Der Klassenname gehört nach der Versetzung einem
// anderen Jahrgang.
func (g *klassenGrouper) add(klasse string, ehemalig bool, schuelerID, name string, medium UeberfaelligesMedium) {
	schluessel := "klasse|" + klasse
	if ehemalig {
		klasse, schluessel = GruppeEhemalige, "ehemalige"
	}
	ki, ok := g.klassenIdx[schluessel]
	if !ok {
		g.klassen = append(g.klassen, MahnwesenKlasse{Klasse: klasse, Ehemalige: ehemalig})
		ki = len(g.klassen) - 1
		g.klassenIdx[schluessel] = ki
	}

	schuelerKey := schluessel + "|" + schuelerID
	si, ok := g.schuelerIdx[schuelerKey]
	if !ok {
		g.klassen[ki].Schueler = append(g.klassen[ki].Schueler, UeberfaelligerSchueler{
			SchuelerID: schuelerID,
			Name:       name,
			Klasse:     klasse,
		})
		si = len(g.klassen[ki].Schueler) - 1
		g.schuelerIdx[schuelerKey] = si
	}

	g.klassen[ki].Schueler[si].Medien = append(g.klassen[ki].Schueler[si].Medien, medium)
}

// QueryUeberfaelligeNachKlasse ermittelt alle Ausleihen, deren Frist überschritten ist,
// gruppiert nach Klasse und Schüler; Ehemalige stehen als eigene Gruppe am Ende. Ein
// optionaler Filter schränkt die Abfrage auf die heutigen Schüler einer Klasse ein.
func (repo *MahnwesenRepository) QueryUeberfaelligeNachKlasse(ctx context.Context, klasseFilter string) ([]MahnwesenKlasse, error) {
	q := `
		SELECT a.id, s.id, s.vorname || ' ' || s.nachname, s.klasse, s.ist_abgaenger,
		       t.titel, coalesce(t.autor,''), coalesce(t.isbn,''), coalesce(t.cover_url,''),
		       coalesce(e.barcode_id,''),
		       a.rueckgabe_frist,
		       GREATEST(0, EXTRACT(DAY FROM (CURRENT_TIMESTAMP - a.rueckgabe_frist))::int) AS tage_ueberfaellig
		FROM ausleihen a
		JOIN buecher_exemplare e ON a.exemplar_id = e.id
		JOIN buecher_titel t    ON e.titel_id = t.id
		-- Die SICHT schueler und nicht die Tabelle leser: Gemahnt werden Schüler, nicht
		-- das Kollegium (Entscheidung zum Lehrer-Anliegen). Der INNER JOIN lässt die
		-- Ausleihen von Kollegen aus dem Mahnlauf fallen — vor Migration 125 taten das
		-- die zwei getrennten Spalten.
		JOIN schueler s         ON a.schueler_id = s.id
		WHERE a.rueckgabe_am IS NULL
		  AND a.rueckgabe_frist < CURRENT_TIMESTAMP
		  AND s.deleted_at IS NULL
	`
	args := []any{}
	if klasseFilter != "" {
		q += " AND s.klasse = $1 AND s.ist_abgaenger = false"
		args = append(args, klasseFilter)
	}
	q += ` ORDER BY s.ist_abgaenger, CASE WHEN s.ist_abgaenger THEN '' ELSE s.klasse END,
		s.nachname, s.vorname, a.rueckgabe_frist`

	rows, err := repo.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	g := newKlassenGrouper()

	for rows.Next() {
		var ausleiheID, schuelerID, name, klasse string
		var ehemalig bool
		var titel, autor, isbn, coverURL, exBarcode string
		var frist time.Time
		var tage int
		if err := rows.Scan(&ausleiheID, &schuelerID, &name, &klasse, &ehemalig,
			&titel, &autor, &isbn, &coverURL, &exBarcode,
			&frist, &tage); err != nil {
			return nil, err
		}

		g.add(klasse, ehemalig, schuelerID, name, UeberfaelligesMedium{
			AusleiheID:       ausleiheID,
			Titel:            titel,
			Autor:            autor,
			ISBN:             isbn,
			Barcode:          exBarcode,
			CoverURL:         coverURL,
			FaelligAm:        frist.Format("02.01.2006"),
			TageUeberfaellig: tage,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	klassen := g.klassen

	repo.reichereLehrerEmails(ctx, klassen)
	return klassen, nil
}

// KlassenSchluessel normalisiert ein Klassenkürzel für den Vergleich zwischen
// Schülerdaten und Klassenlehrer-Mapping. Die beiden Quellen werden von Menschen
// bzw. vom LUSD-Import befüllt und stimmen in Schreibweise und Leerzeichen nicht
// zuverlässig überein.
func KlassenSchluessel(klasse string) string {
	return strings.ToLower(strings.TrimSpace(klasse))
}

// reichereLehrerEmails ordnet den Klassen die Lehrer-E-Mails aus dem Mapping zu.
// Best-effort: bei fehlendem/teilweisem Mapping bleibt LehrerEmail leer.
//
// Verglichen wird NORMALISIERT. Vorher war es ein exakter Schlüsselvergleich: Ein im
// Mapping als „5A" oder mit angehängtem Leerzeichen erfasstes Kürzel traf die Klasse
// „5a" nicht — die Oberfläche meldete „keine E-Mail", obwohl die Adresse hinterlegt
// war, und der Versand übersprang die Klasse still.
func (repo *MahnwesenRepository) reichereLehrerEmails(ctx context.Context, klassen []MahnwesenKlasse) {
	if len(klassen) == 0 {
		return
	}
	mRows, err := repo.db.Query(ctx, `SELECT klasse, lehrer_email FROM klassen_lehrer_mapping`)
	if err != nil {
		return
	}
	defer mRows.Close()

	emailMap := map[string]string{}
	for mRows.Next() {
		var k, e string
		if err := mRows.Scan(&k, &e); err == nil {
			emailMap[KlassenSchluessel(k)] = e
		}
	}
	if err := mRows.Err(); err != nil {
		emailMap = map[string]string{} // Teil-Mapping verwerfen (best-effort-Anreicherung)
	}
	for i := range klassen {
		if klassen[i].Ehemalige {
			continue
		}
		klassen[i].LehrerEmail = emailMap[KlassenSchluessel(klassen[i].Klasse)]
	}
}

// sqlUeberfaelligeByAusleiheIDs sammelt die PDF-Daten zu konkreten Ausleih-IDs.
// rueckgabe_am IS NULL: bereits zurückgegebene Bücher gehören nicht in eine Mahnung.
// rueckgabe_frist < CURRENT_TIMESTAMP: ein Buch, dessen Frist noch läuft, ebenso wenig.
// s.deleted_at IS NULL: für einen in den Papierkorb gelöschten Schüler dürfen keine
// Mahn-Daten mehr verarbeitet werden (DSGVO — sonst gingen "Zombie-Mahnungen" an den
// ehemaligen Klassenlehrer).
const sqlUeberfaelligeByAusleiheIDs = `
	SELECT a.id, s.id, s.vorname || ' ' || s.nachname, s.klasse,
	       t.titel, coalesce(t.autor,''), coalesce(t.isbn,''), coalesce(t.cover_url,''),
	       coalesce(e.barcode_id,''),
	       a.rueckgabe_frist,
	       GREATEST(0, EXTRACT(DAY FROM (CURRENT_TIMESTAMP - a.rueckgabe_frist))::int) AS tage_ueberfaellig
	FROM ausleihen a
	JOIN buecher_exemplare e ON a.exemplar_id = e.id
	JOIN buecher_titel t    ON e.titel_id = t.id
	JOIN schueler s         ON a.schueler_id = s.id
	WHERE a.id = ANY($1) AND a.rueckgabe_am IS NULL
	  AND a.rueckgabe_frist < CURRENT_TIMESTAMP
	  AND s.deleted_at IS NULL
	ORDER BY s.klasse, s.nachname, s.vorname, a.rueckgabe_frist
`

// scanUeberfaelligeKlassen liest die Spaltenreihenfolge von sqlUeberfaelligeByAusleiheIDs
// in die gruppierte Klassen→Schüler→Medien-Struktur.
func scanUeberfaelligeKlassen(rows pgx.Rows) ([]MahnwesenKlasse, error) {
	g := newKlassenGrouper()
	for rows.Next() {
		var ausleiheID, schuelerID, name, klasse string
		var titel, autor, isbn, coverURL, exBarcode string
		var frist time.Time
		var tage int
		if err := rows.Scan(&ausleiheID, &schuelerID, &name, &klasse,
			&titel, &autor, &isbn, &coverURL, &exBarcode,
			&frist, &tage); err != nil {
			return nil, err
		}
		// Das Blatt nennt die Klasse, die das Kind zuletzt trug; gruppiert wird nur für
		// die Reihenfolge der Seiten.
		g.add(klasse, false, schuelerID, name, UeberfaelligesMedium{
			AusleiheID:       ausleiheID,
			Titel:            titel,
			Autor:            autor,
			ISBN:             isbn,
			Barcode:          exBarcode,
			CoverURL:         coverURL,
			FaelligAm:        frist.Format("02.01.2006"),
			TageUeberfaellig: tage,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return g.klassen, nil
}

// ZaehleMahnungTx erhöht die Mahnstufe der genannten Ausleihen und liefert, wie viele es traf.
// Gezählt wird, was auf dem Blatt steht (sqlUeberfaelligeByAusleiheIDs): das offene Buch eines
// Schülers außerhalb des Papierkorbs, dessen Frist abgelaufen ist. Eine Ausleihe steigt
// höchstens einmal je Kalendertag der Schule, deshalb stehen beide Seiten des Vergleichs in
// der Schulzeitzone. Das UPDATE sperrt die getroffenen Zeilen bis zum Commit.
func (repo *MahnwesenRepository) ZaehleMahnungTx(ctx context.Context, tx pgx.Tx, ids []string) (int64, error) {
	tag, err := tx.Exec(ctx, `
		UPDATE ausleihen a
		SET mahnstufe = a.mahnstufe + 1,
		    letztes_mahndatum = CURRENT_TIMESTAMP
		FROM schueler s
		WHERE a.id = ANY($1)
		  AND s.id = a.schueler_id
		  AND s.deleted_at IS NULL
		  AND a.rueckgabe_am IS NULL
		  AND a.rueckgabe_frist < CURRENT_TIMESTAMP
		  AND (a.letztes_mahndatum IS NULL
		       OR (a.letztes_mahndatum AT TIME ZONE '`+schulzeit.ZonenName+`')::date < `+schulzeit.SQLHeute+`)
	`, ids)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// QueryUeberfaelligeByAusleiheIDsTx liest dieselben Daten INNERHALB einer Transaktion.
// Der Bulk-Mahnlauf ruft dies NACH dem Mahnstufen-UPDATE in derselben Tx auf, damit das
// PDF exakt den festgeschriebenen Zustand widerspiegelt (Papier == DB). Läge das Auslesen
// vor der Tx, könnte ein zwischenzeitlich zurückgegebenes Buch aufs Papier geraten, ohne
// dass seine Mahnstufe steigt (TOCTOU).
func (repo *MahnwesenRepository) QueryUeberfaelligeByAusleiheIDsTx(ctx context.Context, tx pgx.Tx, ids []string) ([]MahnwesenKlasse, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := tx.Query(ctx, sqlUeberfaelligeByAusleiheIDs, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanUeberfaelligeKlassen(rows)
}
