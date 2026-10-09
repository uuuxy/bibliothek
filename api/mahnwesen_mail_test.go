package api

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"bibliothek/pdf"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"
)

// Die Mahnliste entsteht aus den Gruppen der Abfrage. Jedes Feld ihrer Eingabe kommt aus
// seiner Quelle, die Werte sind je Feld verschieden, auch die Klasse des Schülers und der Name
// seiner Gruppe; ein Feld, das die Eingabe später dazubekommt und das niemand füllt, fällt an
// leereFelder auf.
func TestMahnlisteSeiten_FuelltJedesFeldDerEingabe(t *testing.T) {
	klassen := []repository.MahnwesenKlasse{
		{Klasse: "05F", LehrerEmail: "leitung5f@schule.invalid", Schueler: []repository.UeberfaelligerSchueler{
			{SchuelerID: "s-1", Name: "Anna Apfel", Klasse: "05F neu", Medien: []repository.UeberfaelligesMedium{
				{AusleiheID: "a-1", Titel: "Titel 1", Autor: "Autor 1", ISBN: "isbn-1", Barcode: "B-1",
					CoverURL: "/uploads/c1.webp", FaelligAm: "01.06.2026", TageUeberfaellig: 9, Mahnstufe: 1, LetztesMahndatum: "2026-06-05"},
				{AusleiheID: "a-2", Titel: "Titel 2", Autor: "Autor 2", ISBN: "isbn-2", Barcode: "B-2",
					CoverURL: "/uploads/c2.webp", FaelligAm: "02.06.2026", TageUeberfaellig: 8, Mahnstufe: 2, LetztesMahndatum: "2026-06-06"},
			}},
		}},
		{Klasse: "06G"},
		{Klasse: repository.GruppeEhemalige, Ehemalige: true, Schueler: []repository.UeberfaelligerSchueler{
			{SchuelerID: "s-2", Name: "Ben Birne", Klasse: "13T zuletzt", Medien: []repository.UeberfaelligesMedium{
				{AusleiheID: "a-3", Titel: "Titel 3", Autor: "Autor 3", ISBN: "isbn-3", Barcode: "B-3",
					CoverURL: "/uploads/c3.webp", FaelligAm: "03.06.2026", TageUeberfaellig: 7},
			}},
		}},
	}

	seiten := mahnlisteSeiten(klassen)
	want := []pdf.MahnlisteSchueler{
		{Name: "Anna Apfel", Klasse: "05F neu", Medien: []pdf.MahnlisteMedium{
			{Titel: "Titel 1", Autor: "Autor 1", Barcode: "B-1", CoverURL: "/uploads/c1.webp", FaelligAm: "01.06.2026", TageUeberfaellig: 9},
			{Titel: "Titel 2", Autor: "Autor 2", Barcode: "B-2", CoverURL: "/uploads/c2.webp", FaelligAm: "02.06.2026", TageUeberfaellig: 8},
		}},
		{Name: "Ben Birne", Klasse: "13T zuletzt", Medien: []pdf.MahnlisteMedium{
			{Titel: "Titel 3", Autor: "Autor 3", Barcode: "B-3", CoverURL: "/uploads/c3.webp", FaelligAm: "03.06.2026", TageUeberfaellig: 7},
		}},
	}
	if !reflect.DeepEqual(seiten, want) {
		t.Errorf("Eingabe der Mahnliste =\n%+v\nerwartet\n%+v", seiten, want)
	}
	if leer := leereFelder(reflect.ValueOf(seiten), "[]pdf.MahnlisteSchueler"); len(leer) > 0 {
		t.Errorf("mahnlisteSeiten füllt diese Felder nicht: %v", leer)
	}
}

// Ohne Schüler gibt es keine Seite: Der Erzeuger druckt dann den Satz, dass nichts überfällig ist.
func TestMahnlisteSeiten_OhneSchuelerKeineSeite(t *testing.T) {
	for name, klassen := range map[string][]repository.MahnwesenKlasse{
		"keine Gruppe":           nil,
		"Gruppen ohne Schüler":   {{Klasse: "05F"}, {Klasse: "06G"}},
		"leere Liste der Gruppe": {{Klasse: "05F", Schueler: []repository.UeberfaelligerSchueler{}}},
	} {
		if seiten := mahnlisteSeiten(klassen); len(seiten) != 0 {
			t.Errorf("%s: %d Seiten, erwartet keine", name, len(seiten))
		}
	}
}

// Die Mail an die Klassenleitung nennt Klasse, Tag und die zwei Zahlen und trägt die Liste als
// Anhang. Der Tag ist der der Schule: im Betreff und im Text in der deutschen Form, im
// Dateinamen in der sortierbaren.
func TestBaueMahnMailRequest_NenntKlasseTagUndZahlenUndTraegtDieListe(t *testing.T) {
	liste := []byte("%PDF-Probe")
	vorher := schulzeit.Jetzt()
	mail := baueMahnMailRequest(mahnwesenSendenRequest{Klasse: "07B", Email: "leitung7b@schule.invalid"}, liste, 3, 5)
	nachher := schulzeit.Jetzt()

	// Läuft der Test über Mitternacht, gilt der Tag davor oder danach.
	einerDerTage := func(text, muster, form string) bool {
		return text == fmt.Sprintf(muster, vorher.Format(form)) || text == fmt.Sprintf(muster, nachher.Format(form))
	}

	if mail.To != "leitung7b@schule.invalid" {
		t.Errorf("Empfänger = %q, erwartet die Adresse der Klassenleitung", mail.To)
	}
	if !einerDerTage(mail.Subject, "Mahnliste Schulbibliothek – Klasse 07B – %s", "02.01.2006") {
		t.Errorf("Betreff = %q, erwartet Klasse und heutigen Tag in der Form TT.MM.JJJJ", mail.Subject)
	}
	kopf, rest, _ := strings.Cut(mail.Body, "\n\nBetroffene")
	if !einerDerTage(kopf, "Sehr geehrte Damen und Herren,\n\nanbei erhalten Sie die aktuelle Mahnliste der "+
		"Schulbibliothek für die Klasse 07B (Stand: %s).", "02.01.2006") {
		t.Errorf("Anfang des Texts = %q, erwartet Klasse und heutigen Tag", kopf)
	}
	if !strings.HasPrefix(rest, " Schüler/innen: 3\nÜberfällige Medien gesamt: 5\n") {
		t.Errorf("Zahlen im Text = %q, erwartet 3 Schüler/innen und 5 Medien", rest)
	}

	if len(mail.Attachments) != 1 {
		t.Fatalf("%d Anhänge, erwartet einen", len(mail.Attachments))
	}
	anhang := mail.Attachments[0]
	if !einerDerTage(anhang.Name, "mahnliste_07B_%s.pdf", "2006-01-02") {
		t.Errorf("Name des Anhangs = %q, erwartet Klasse und heutigen Tag in der Form JJJJ-MM-TT", anhang.Name)
	}
	if anhang.ContentType != contentTypePDF || !bytes.Equal(anhang.Data, liste) {
		t.Errorf("Anhang: Typ %q mit %d Bytes, erwartet die übergebene Liste als PDF", anhang.ContentType, len(anhang.Data))
	}
}
