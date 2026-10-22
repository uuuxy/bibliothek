package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

// Die Felder beim Zusammenführen zweier Titel: Was am Ziel steht, bleibt; was dort leer ist,
// kommt von der Quelle. Leer heißt hier „unbekannt". Ein Feld, in dem leer etwas bedeutet,
// füllt die Quelle nicht auf: Ein Titel ohne Schulzweig gilt für alle Zweige.

// TitelFeld ist ein Feld, das das Ziel von der Quelle übernimmt, mit der Bezeichnung der Maske.
type TitelFeld struct {
	Bezeichnung string `json:"bezeichnung"`
	Wert        string `json:"wert"`
}

// TitelAbweichung ist ein Feld, in dem beide Titel verschieden sind; das Ziel behält seinen Wert.
type TitelAbweichung struct {
	Bezeichnung string `json:"bezeichnung"`
	Hier        string `json:"hier"`
	Dort        string `json:"dort"`
}

// titelStand sind die Felder eines Titels, die beim Zusammenführen zählen, als Text gelesen.
type titelStand struct {
	id, titel, medientyp, erweitert                                         string
	istLernmittel                                                           bool
	isbn, untertitel, autor, verlag, erscheinungsjahr, auflage, listenpreis *string
	signatur, fach, zweig, coverURL, coverStatus, zaehldatum                *string
	jahrgangVon, jahrgangBis                                                *string
}

// auffuellbar sind die Felder, die die Quelle am Ziel auffüllt, in der Reihenfolge der Maske.
var auffuellbar = []struct {
	bezeichnung string
	feld        func(*titelStand) **string
}{
	{"ISBN", func(t *titelStand) **string { return &t.isbn }},
	{"Untertitel", func(t *titelStand) **string { return &t.untertitel }},
	{"Autor", func(t *titelStand) **string { return &t.autor }},
	{"Verlag", func(t *titelStand) **string { return &t.verlag }},
	{"Erscheinungsjahr", func(t *titelStand) **string { return &t.erscheinungsjahr }},
	{"Auflage", func(t *titelStand) **string { return &t.auflage }},
	{"Listenpreis", func(t *titelStand) **string { return &t.listenpreis }},
	{"Signatur", func(t *titelStand) **string { return &t.signatur }},
	{"Fach", func(t *titelStand) **string { return &t.fach }},
	{"Zähldatum", func(t *titelStand) **string { return &t.zaehldatum }},
}

func leer(s *string) bool { return s == nil || strings.TrimSpace(*s) == "" }

func wert(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// fuegeTitelZusammen liefert die Felder des Ziels nach dem Zusammenführen, dazu, was es von der
// Quelle übernimmt und wo beide verschieden sind. Tragen beide eine ISBN, lehnt es ab.
func fuegeTitelZusammen(ziel, quelle titelStand) (titelStand, []TitelFeld, []TitelAbweichung, error) {
	if !leer(ziel.isbn) && !leer(quelle.isbn) {
		return ziel, nil, nil, ErrTitelZusammenfuehrenZweiISBN
	}
	nachher := ziel
	uebernommen := []TitelFeld{}
	abweichend := []TitelAbweichung{}
	for _, f := range auffuellbar {
		hier, dort := f.feld(&nachher), f.feld(&quelle)
		switch {
		case leer(*hier) && !leer(*dort):
			*hier = *dort
			uebernommen = append(uebernommen, TitelFeld{f.bezeichnung, anzeige(f.bezeichnung, *dort)})
		case !leer(*hier) && !leer(*dort) && strings.TrimSpace(**hier) != strings.TrimSpace(**dort):
			abweichend = append(abweichend, TitelAbweichung{f.bezeichnung, anzeige(f.bezeichnung, *hier), anzeige(f.bezeichnung, *dort)})
		}
	}
	// Die Jahrgänge kommen nur zusammen (chk_jahrgang_spanne).
	switch hier, dort := jahrgang(ziel), jahrgang(quelle); {
	case hier == "" && dort != "":
		nachher.jahrgangVon, nachher.jahrgangBis = quelle.jahrgangVon, quelle.jahrgangBis
		uebernommen = append(uebernommen, TitelFeld{"Jahrgang", dort})
	case hier != "" && dort != "" && hier != dort:
		abweichend = append(abweichend, TitelAbweichung{"Jahrgang", hier, dort})
	}
	if leer(ziel.coverURL) && !leer(quelle.coverURL) {
		nachher.coverURL, nachher.coverStatus = quelle.coverURL, quelle.coverStatus
		uebernommen = append(uebernommen, TitelFeld{"Cover", ""})
	}
	// Was die Quelle sonst noch weiß, kommt dazu; ein Schlüssel des Ziels bleibt
	// (schreibeTitelStand: quelle || ziel).
	nachher.erweitert = quelle.erweitert

	// Felder, die die Quelle nie auffüllt, stehen vorn: Titel und Art bestimmen, was das Ziel
	// danach ist.
	kopf := []TitelAbweichung{}
	for _, a := range []TitelAbweichung{
		{"Titel", ziel.titel, quelle.titel},
		{"Art des Mediums", ziel.medientyp, quelle.medientyp},
		{"Schulzweig", leerAlsAlle(ziel.zweig), leerAlsAlle(quelle.zweig)},
		{"Lernmittel", jaNein(ziel.istLernmittel), jaNein(quelle.istLernmittel)},
	} {
		if strings.TrimSpace(a.Hier) != strings.TrimSpace(a.Dort) {
			kopf = append(kopf, a)
		}
	}
	return nachher, uebernommen, append(kopf, abweichend...), nil
}

// anzeige bringt einen gelesenen Wert in die Form, die die Maske zeigt.
func anzeige(bezeichnung string, s *string) string {
	v := strings.TrimSpace(wert(s))
	switch bezeichnung {
	case "Listenpreis":
		return strings.Replace(v, ".", ",", 1) + " €"
	case "Zähldatum":
		if teile := strings.Split(v, "-"); len(teile) == 3 {
			return teile[2] + "." + teile[1] + "." + teile[0]
		}
	}
	return v
}

func jahrgang(t titelStand) string {
	if leer(t.jahrgangVon) || leer(t.jahrgangBis) {
		return ""
	}
	return *t.jahrgangVon + "–" + *t.jahrgangBis
}

func leerAlsAlle(zweig *string) string {
	if leer(zweig) {
		return "alle"
	}
	return *zweig
}

func jaNein(b bool) string {
	if b {
		return "ja"
	}
	return "nein"
}

// alsProtokoll hält die Felder eines Titels für den Eintrag im Protokoll fest, nur die belegten.
func (t titelStand) alsProtokoll() map[string]any {
	felder := map[string]any{"titel": t.titel, "medientyp": t.medientyp, "ist_lernmittel": t.istLernmittel}
	for _, f := range []struct {
		schluessel string
		wert       *string
	}{
		{"isbn", t.isbn}, {"untertitel", t.untertitel}, {"autor", t.autor}, {"verlag", t.verlag},
		{"erscheinungsjahr", t.erscheinungsjahr}, {"auflage", t.auflage}, {"listenpreis", t.listenpreis},
		{"signatur", t.signatur}, {"subject", t.fach}, {"track", t.zweig}, {"cover_url", t.coverURL},
		{"last_counted", t.zaehldatum}, {"jahrgang_von", t.jahrgangVon}, {"jahrgang_bis", t.jahrgangBis},
	} {
		if !leer(f.wert) {
			felder[f.schluessel] = *f.wert
		}
	}
	return felder
}

// ladeTitelStand liest einen Titel, den die Transaktion schon gesperrt hat (sperreAuflagen).
func ladeTitelStand(ctx context.Context, tx pgx.Tx, id string) (titelStand, error) {
	t := titelStand{id: id}
	err := tx.QueryRow(ctx, `
		SELECT titel, medientyp, erweiterte_eigenschaften::text, ist_lernmittel,
		       isbn, untertitel, autor, verlag, erscheinungsjahr::text, auflage, listenpreis::text,
		       signatur, subject, track, cover_url, cover_status, last_counted::text,
		       jahrgang_von::text, jahrgang_bis::text
		FROM buecher_titel WHERE id = $1`, id).Scan(
		&t.titel, &t.medientyp, &t.erweitert, &t.istLernmittel,
		&t.isbn, &t.untertitel, &t.autor, &t.verlag, &t.erscheinungsjahr, &t.auflage, &t.listenpreis,
		&t.signatur, &t.fach, &t.zweig, &t.coverURL, &t.coverStatus, &t.zaehldatum,
		&t.jahrgangVon, &t.jahrgangBis)
	if errors.Is(err, pgx.ErrNoRows) {
		return t, ErrTitelNichtGefunden
	}
	if err != nil {
		return t, fmt.Errorf("titel lesen: %w", err)
	}
	return t, nil
}

// schreibeTitelStand schreibt die aufgefüllten Felder an das Ziel. Die Felder, die das Ziel
// behält, stehen mit seinem eigenen Wert darin; die Zeile ist gesperrt, kein anderer hat sie
// seit dem Lesen geändert.
func schreibeTitelStand(ctx context.Context, tx pgx.Tx, t titelStand) error {
	tag, err := tx.Exec(ctx, `
		UPDATE buecher_titel SET
			isbn = $2, untertitel = $3, autor = $4, verlag = $5, erscheinungsjahr = $6::integer,
			auflage = $7, listenpreis = $8::numeric, signatur = $9, subject = $10,
			cover_url = $11, cover_status = $12, last_counted = $13::date,
			jahrgang_von = $14::integer, jahrgang_bis = $15::integer,
			erweiterte_eigenschaften = $16::jsonb || erweiterte_eigenschaften
		WHERE id = $1`,
		t.id, t.isbn, t.untertitel, t.autor, t.verlag, t.erscheinungsjahr,
		t.auflage, t.listenpreis, t.signatur, t.fach,
		t.coverURL, t.coverStatus, t.zaehldatum,
		t.jahrgangVon, t.jahrgangBis, t.erweitert)
	if err != nil {
		return fmt.Errorf("titel zusammenführen: ziel schreiben: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("titel zusammenführen: ziel schreiben: %d zeilen statt 1", tag.RowsAffected())
	}
	return nil
}
