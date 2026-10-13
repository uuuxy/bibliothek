package repository

import (
	"context"
	"strings"
	"time"
)

// klassenGruppen sammelt Zeilen zu Klassen→Schüler→Medien. Bewusst index-basiert:
// Pointer in Slice-Elemente werden bei append-Reallokationen ungültig und haben
// Medien gleichnamiger Schüler still verschluckt.
type klassenGruppen struct {
	klassen     []MahnwesenKlasse
	klassenIdx  map[string]int
	schuelerIdx map[string]int
}

func newKlassenGruppen() *klassenGruppen {
	return &klassenGruppen{
		klassen:     make([]MahnwesenKlasse, 0),
		klassenIdx:  map[string]int{},
		schuelerIdx: map[string]int{},
	}
}

// merke ordnet ein Medium seiner Gruppe zu. Ehemalige bilden eine Gruppe für sich, gleich
// welche Klasse sie zuletzt trugen: Der Klassenname gehört nach der Versetzung einem
// anderen Jahrgang.
func (g *klassenGruppen) merke(klasse string, ehemalig bool, schuelerID, name string, medium UeberfaelligesMedium) {
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

// LadeUeberfaelligeNachKlasse ermittelt alle Ausleihen, deren Frist überschritten ist,
// gruppiert nach Klasse und Schüler; Ehemalige stehen als eigene Gruppe am Ende. Ein
// optionaler Filter schränkt die Abfrage auf die heutigen Schüler einer Klasse ein.
func (a *MahnwesenAbfragen) LadeUeberfaelligeNachKlasse(ctx context.Context, klassenFilter string) ([]MahnwesenKlasse, error) {
	q := `
		SELECT a.id, s.id, s.vorname || ' ' || s.nachname, s.klasse, s.ist_abgaenger,
		       t.titel, coalesce(t.autor,''), coalesce(t.isbn,''), coalesce(t.cover_url,''),
		       coalesce(e.barcode_id,''),
		       a.rueckgabe_frist,
		       GREATEST(0, EXTRACT(DAY FROM (CURRENT_TIMESTAMP - a.rueckgabe_frist))::int) AS tage_ueberfaellig,
		       a.mahnstufe, a.letztes_mahndatum
		FROM ausleihen a
		JOIN buecher_exemplare e ON a.exemplar_id = e.id
		JOIN buecher_titel t    ON e.titel_id = t.id
		-- Die Sicht schueler und nicht die Tabelle leser: Gemahnt werden Schüler, nicht
		-- das Kollegium (Entscheidung zum Lehrer-Anliegen). Der INNER JOIN lässt die
		-- Ausleihen von Kollegen aus dem Mahnlauf fallen — vor Migration 125 taten das
		-- die zwei getrennten Spalten.
		JOIN schueler s         ON a.schueler_id = s.id
		WHERE a.rueckgabe_am IS NULL
		  AND a.rueckgabe_frist < CURRENT_TIMESTAMP
		  AND s.deleted_at IS NULL
	`
	args := []any{}
	if klassenFilter != "" {
		q += " AND s.klasse = $1 AND s.ist_abgaenger = false"
		args = append(args, klassenFilter)
	}
	q += ` ORDER BY s.ist_abgaenger, CASE WHEN s.ist_abgaenger THEN '' ELSE s.klasse END,
		s.nachname, s.vorname, a.rueckgabe_frist`

	rows, err := a.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	g := newKlassenGruppen()

	for rows.Next() {
		var ausleiheID, schuelerID, name, klasse string
		var ehemalig bool
		var titel, autor, isbn, coverURL, exBarcode string
		var frist time.Time
		var tage, mahnstufe int
		var mahndatum *time.Time
		if err := rows.Scan(&ausleiheID, &schuelerID, &name, &klasse, &ehemalig,
			&titel, &autor, &isbn, &coverURL, &exBarcode,
			&frist, &tage, &mahnstufe, &mahndatum); err != nil {
			return nil, err
		}

		g.merke(klasse, ehemalig, schuelerID, name, UeberfaelligesMedium{
			AusleiheID:       ausleiheID,
			Titel:            titel,
			Autor:            autor,
			ISBN:             isbn,
			Barcode:          exBarcode,
			CoverURL:         coverURL,
			FaelligAm:        frist.Format(datumsformat),
			TageUeberfaellig: tage,
			Mahnstufe:        mahnstufe,
			LetztesMahndatum: mahndatumText(mahndatum),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	klassen := g.klassen

	a.reichereLehrerEmailsAn(ctx, klassen)
	return klassen, nil
}

// KlassenSchluessel normalisiert ein Klassenkürzel für den Vergleich zwischen
// Schülerdaten und Klassenlehrer-Mapping. Die beiden Quellen werden von Menschen
// bzw. vom LUSD-Import befüllt und stimmen in Schreibweise und Leerzeichen nicht
// zuverlässig überein.
func KlassenSchluessel(klasse string) string {
	return strings.ToLower(strings.TrimSpace(klasse))
}

// reichereLehrerEmailsAn ordnet den Klassen die Adressen der Klassenleitungen zu. Fehlt eine
// Zuordnung, bleibt LehrerEmail leer.
//
// Verglichen wird normalisiert: Bei einem genauen Vergleich träfe ein Kürzel, das als „5A" oder
// mit angehängtem Leerzeichen erfasst ist, die Klasse „5a" nicht. Die Oberfläche meldete „keine
// E-Mail", obwohl die Adresse hinterlegt ist, und der Versand überspränge die Klasse still.
func (a *MahnwesenAbfragen) reichereLehrerEmailsAn(ctx context.Context, klassen []MahnwesenKlasse) {
	if len(klassen) == 0 {
		return
	}
	mRows, err := a.db.Query(ctx, `SELECT klasse, lehrer_email FROM klassen_lehrer_mapping`)
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
