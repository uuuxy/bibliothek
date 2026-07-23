package repository

import (
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
)

// medienAnzahl liefert die Anzahl der Medien des Schülers mit der gegebenen ID
// (0, wenn nicht in der Liste).
func medienAnzahl(schueler []UeberfaelligerSchueler, id string) int {
	for _, sch := range schueler {
		if sch.SchuelerID == id {
			return len(sch.Medien)
		}
	}
	return 0
}

// T4 (Fahrplan): Die Gruppierung der Mahnliste ist geldrelevant — verlorene
// Medien bedeuten nicht gemahnte Bücher. Der Test deckt insbesondere den Fall
// zweier gleichnamiger Schüler ab, deren Zeilen durch die Sortierung
// (nachname, vorname, frist) verzahnt eintreffen.
func TestQueryUeberfaelligeNachKlasse_GruppiertKorrekt(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewMahnwesenRepository(mock)
	frist := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)

	rows := pgxmock.NewRows([]string{
		"id", "s_id", "name", "klasse",
		"titel", "autor", "isbn", "cover_url", "barcode",
		"rueckgabe_frist", "tage_ueberfaellig",
	}).
		// Klasse 7A: zwei VERSCHIEDENE Schülerinnen mit identischem Namen,
		// deren Ausleihen nach Frist verzahnt sortiert sind.
		AddRow("a1", "s1", "Anna Müller", "7A", "Faust", "Goethe", "978-1", "", "B-1", frist, 17).
		AddRow("a2", "s2", "Anna Müller", "7A", "Die Räuber", "Schiller", "978-2", "", "B-2", frist.AddDate(0, 0, 1), 16).
		AddRow("a3", "s1", "Anna Müller", "7A", "Woyzeck", "Büchner", "978-3", "", "B-3", frist.AddDate(0, 0, 2), 15).
		// Zweite Klasse — löst die Reallokation des klassen-Slices aus.
		AddRow("a4", "s3", "Ben Yilmaz", "8B", "Effi Briest", "Fontane", "978-4", "", "B-4", frist, 17)

	mock.ExpectQuery(`SELECT a\.id, s\.id, s\.vorname \|\| ' ' \|\| s\.nachname, s\.klasse`).
		WillReturnRows(rows)

	mock.ExpectQuery(`SELECT klasse, lehrer_email FROM klassen_lehrer_mapping`).
		WillReturnRows(pgxmock.NewRows([]string{"klasse", "lehrer_email"}).
			AddRow("7A", "lehrer7a@schule.de"))

	klassen, err := repo.QueryUeberfaelligeNachKlasse(t.Context(), "")
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}

	if len(klassen) != 2 {
		t.Fatalf("erwartet 2 Klassen, bekam %d", len(klassen))
	}

	k7a := klassen[0]
	if k7a.Klasse != "7A" || len(k7a.Schueler) != 2 {
		t.Fatalf("7A: erwartet 2 Schüler, bekam %+v", k7a)
	}
	if k7a.LehrerEmail != "lehrer7a@schule.de" {
		t.Errorf("LehrerEmail nicht zugeordnet: %q", k7a.LehrerEmail)
	}

	// s1 hat ZWEI Medien — das zweite kam nach dem Einschub von s2.
	// Verliert die Gruppierung es, wird das Buch nie angemahnt.
	if n := medienAnzahl(k7a.Schueler, "s1"); n != 2 {
		t.Errorf("s1: erwartet 2 Medien, bekam %d (Medium bei Slice-Reallokation verloren?)", n)
	}
	if n := medienAnzahl(k7a.Schueler, "s2"); n != 1 {
		t.Errorf("s2: erwartet 1 Medium, bekam %d", n)
	}

	k8b := klassen[1]
	if k8b.Klasse != "8B" || len(k8b.Schueler) != 1 || len(k8b.Schueler[0].Medien) != 1 {
		t.Fatalf("8B falsch gruppiert: %+v", k8b)
	}
	if k8b.LehrerEmail != "" {
		t.Errorf("8B hat kein Mapping, LehrerEmail muss leer sein: %q", k8b.LehrerEmail)
	}

	// Formatierung ist Teil des Mahnschreibens (deutsches Datumsformat).
	m := k8b.Schueler[0].Medien[0]
	if m.FaelligAm != "20.06.2026" || m.TageUeberfaellig != 17 {
		t.Errorf("Medium falsch gemappt: %+v", m)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene Erwartungen: %v", err)
	}
}

func TestQueryUeberfaelligeNachKlasse_MitKlassenfilter(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewMahnwesenRepository(mock)

	mock.ExpectQuery(`AND s\.klasse = \$1`).
		WithArgs("7A").
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "s_id", "name", "klasse",
			"titel", "autor", "isbn", "cover_url", "barcode",
			"rueckgabe_frist", "tage_ueberfaellig",
		}))

	klassen, err := repo.QueryUeberfaelligeNachKlasse(t.Context(), "7A")
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if len(klassen) != 0 {
		t.Errorf("erwartet leere Liste, bekam %d Klassen", len(klassen))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene Erwartungen: %v", err)
	}
}
