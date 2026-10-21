package repository

import (
	"context"
	"time"

	"bibliothek/pkg/schulzeit"

	"github.com/jackc/pgx/v5"
)

// MahnbriefBuch ist ein Buch auf dem Mahnbrief.
type MahnbriefBuch struct {
	Titel            string
	Barcode          string
	AusgeliehenAm    time.Time
	Frist            time.Time
	TageUeberfaellig int
}

// MahnbriefEmpfaenger ist ein Schüler mit seinen Büchern über der Frist und der Anschrift
// für das Fensterkuvert.
type MahnbriefEmpfaenger struct {
	SchuelerID string
	Vorname    string
	Nachname   string
	Strasse    string
	Hausnummer string
	PLZ        string
	Ort        string
	Buecher    []MahnbriefBuch
}

// sqlMahnbriefe liest zu den gewählten Ausleihen, was auf dem Mahnbrief steht: je Schüler
// Anschrift und die offenen Bücher mit abgelaufener Frist. Die Sicht schueler lässt das
// Kollegium aus, ein Schüler im Papierkorb bekommt keinen Brief. Die Briefe einer Klasse
// liegen beieinander, die der Ehemaligen am Ende.
const sqlMahnbriefe = `
	SELECT s.id, s.vorname, s.nachname,
	       coalesce(s.strasse, ''), coalesce(s.hausnummer, ''), coalesce(s.plz, ''), coalesce(s.ort, ''),
	       t.titel, coalesce(e.barcode_id, ''), a.ausgeliehen_am, a.rueckgabe_frist,
	       GREATEST(0, EXTRACT(DAY FROM (CURRENT_TIMESTAMP - a.rueckgabe_frist))::int)
	FROM ausleihen a
	JOIN buecher_exemplare e ON a.exemplar_id = e.id
	JOIN buecher_titel t    ON e.titel_id = t.id
	JOIN schueler s         ON a.schueler_id = s.id
	WHERE a.id = ANY($1) AND a.rueckgabe_am IS NULL
	  AND a.rueckgabe_frist < CURRENT_TIMESTAMP
	  AND s.deleted_at IS NULL
	ORDER BY s.ist_abgaenger, s.klasse COLLATE deutsch, s.nachname COLLATE deutsch, s.vorname COLLATE deutsch, s.id, a.rueckgabe_frist
`

// ErhoeheMahnstufe erhöht die Mahnstufe der genannten Ausleihen und liefert, wie viele es traf.
// Gezählt wird, was auf dem Mahnbrief steht (sqlMahnbriefe): das offene Buch eines Schülers
// außerhalb des Papierkorbs, dessen Frist abgelaufen ist. Eine Ausleihe steigt höchstens
// einmal je Kalendertag der Schule, deshalb stehen beide Seiten des Vergleichs in der
// Schulzeitzone. Das UPDATE sperrt die getroffenen Zeilen bis zum Commit.
func (a *MahnwesenAbfragen) ErhoeheMahnstufe(ctx context.Context, tx pgx.Tx, ids []string) (int64, error) {
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

// LadeMahnbriefe liest die Briefe zu den genannten Ausleihen in der Transaktion, die vorher
// gezählt hat: Auf dem Blatt steht dann derselbe Stand, der gezählt wurde, auch wenn ein
// Buch zwischen dem Laden der Liste und dem Druck zurückkam.
func (a *MahnwesenAbfragen) LadeMahnbriefe(ctx context.Context, tx pgx.Tx, ids []string) ([]MahnbriefEmpfaenger, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := tx.Query(ctx, sqlMahnbriefe, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var briefe []MahnbriefEmpfaenger
	for rows.Next() {
		var e MahnbriefEmpfaenger
		var b MahnbriefBuch
		if err := rows.Scan(&e.SchuelerID, &e.Vorname, &e.Nachname,
			&e.Strasse, &e.Hausnummer, &e.PLZ, &e.Ort,
			&b.Titel, &b.Barcode, &b.AusgeliehenAm, &b.Frist, &b.TageUeberfaellig); err != nil {
			return nil, err
		}
		// Die Sortierung hält die Bücher eines Schülers beieinander.
		if n := len(briefe); n == 0 || briefe[n-1].SchuelerID != e.SchuelerID {
			briefe = append(briefe, e)
		}
		letzter := &briefe[len(briefe)-1]
		letzter.Buecher = append(letzter.Buecher, b)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return briefe, nil
}
