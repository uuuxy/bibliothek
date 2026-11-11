package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v5"
)

// Spalten der Zulauf-Abfrage: die sieben alten plus bestellung_id, lieferant_name,
// bestelldatum (NULL bei Altbestand ohne Bestellung) und der Stand der Bestellung.
var zulaufSpalten = []string{"id", "titel_id", "erstellt_am", "zustand_notiz", "titel", "isbn", "cover_url",
	"bestellung_id", "lieferant_name", "bestelldatum",
	"mit_bestaetigung", "bestaetigt_am", "bestaetigt_durch", "mail_gescheitert_am"}

const zulaufAbfrage = `LEFT JOIN bestellungen_verlauf b ON b\.id = e\.bestellung_id\s*WHERE e\.ist_ausleihbar = false\s*AND e\.bestellstatus IS NOT NULL\s*AND e\.ist_ausgesondert = false\s*ORDER BY e\.erstellt_am DESC`

func TestLadeZulauf_GruppiertNachBestellung(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	// 22:30 UTC am 29.09. ist in Friedrichsdorf schon der 30.09. — der Kalendertag der
	// Schule steht in der Gruppe, nicht der des Servers.
	bestellt := time.Date(2026, 9, 29, 22, 30, 0, 0, time.UTC)
	alt := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	haendler := "Buchhandlung Probe"
	bestaetigt := time.Date(2026, 10, 12, 8, 0, 0, 0, time.UTC)
	gescheitert := time.Date(2026, 9, 29, 22, 31, 0, 0, time.UTC)

	rows := pgxmock.NewRows(zulaufSpalten).
		// Zwei Exemplare derselben Bestellung und desselben Titels: eine Gruppe, eine Position, Menge 2.
		AddRow("ex1", "t1", bestellt, "Im Zulauf - "+haendler, "Titel 1", "123", "cover1", ptr("B1"), &haendler, &bestellt, true, &bestaetigt, ptr("lieferant"), nil).
		AddRow("ex2", "t1", bestellt, "Im Zulauf - "+haendler, "Titel 1", "123", "cover1", ptr("B1"), &haendler, &bestellt, true, &bestaetigt, ptr("lieferant"), nil).
		// Zweiter Topf am selben Tag beim selben Händler: eigene Gruppe.
		AddRow("ex3", "t2", bestellt, "Bestellt (ohne Vorab-Barcode) - "+haendler, "Titel 2", "456", "cover2", ptr("B2"), &haendler, &bestellt, true, nil, nil, &gescheitert).
		// Altbestand ohne Bestellung (vor Migration 063): der Name kommt aus der Notiz.
		AddRow("ex4", "t3", alt, "Im Zulauf - Lieferant A", "Titel 3", "", "", nil, nil, nil, false, nil, nil, nil).
		AddRow("ex5", "t4", alt, "andere notiz", "Titel 4", "", "", nil, nil, nil, false, nil, nil, nil)

	mock.ExpectQuery(zulaufAbfrage).WillReturnRows(rows)

	groups, err := LadeZulauf(context.Background(), mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Die Erwartung steht als Tabelle da, damit auch eine fehlende Gruppe auffällt —
	// eine Schleife mit if/else prüft nur die Gruppen, die tatsächlich kamen.
	erwartet := map[string]struct {
		datum, lieferant string
		posten           int // Titel-Positionen in der Gruppe
		menge            int // Exemplare der ersten Position
	}{
		"B1":                               {"30.09.2026", haendler, 1, 2},
		"B2":                               {"30.09.2026", haendler, 1, 1},
		"01.01.2024|Lieferant A":           {"01.01.2024", "Lieferant A", 1, 1},
		"01.01.2024|Unbekannter Lieferant": {"01.01.2024", "Unbekannter Lieferant", 1, 1},
	}
	if len(groups) != len(erwartet) {
		t.Fatalf("LadeZulauf(…) = %d Gruppen, erwartet %d", len(groups), len(erwartet))
	}
	gesehen := map[string]bool{}
	for _, g := range groups {
		soll, bekannt := erwartet[g.ID]
		if !bekannt {
			t.Errorf("unerwartete Gruppe %q (%s / %s)", g.ID, g.Datum, g.LieferantName)
			continue
		}
		gesehen[g.ID] = true
		if g.Datum != soll.datum || g.LieferantName != soll.lieferant {
			t.Errorf("%s: %s / %s, erwartet %s / %s", g.ID, g.Datum, g.LieferantName, soll.datum, soll.lieferant)
		}
		if len(g.Posten) != soll.posten {
			t.Errorf("%s: %d Positionen, erwartet %d", g.ID, len(g.Posten), soll.posten)
			continue
		}
		if g.Posten[0].Menge != soll.menge {
			t.Errorf("%s: Menge %d, erwartet %d", g.ID, g.Posten[0].Menge, soll.menge)
		}
	}
	for id := range erwartet {
		if !gesehen[id] {
			t.Errorf("Gruppe fehlt vollständig: %s", id)
		}
	}
	// Die Position nennt Titel, ISBN und Cover aus ihren Spalten und ihre Exemplare, das
	// zuerst gelesene zuerst.
	for _, g := range groups {
		if g.ID != "B1" || len(g.Posten) != 1 {
			continue
		}
		p := g.Posten[0]
		if p.TitelID != "t1" || p.Titel != "Titel 1" || p.ISBN != "123" || p.CoverURL != "cover1" ||
			len(p.ExemplarIDs) != 2 || p.ExemplarIDs[0] != "ex1" || p.ExemplarIDs[1] != "ex2" {
			t.Errorf("Position der Bestellung B1: %+v", *p)
		}
	}
	// Die Gruppe trägt den Stand ihrer Bestellung, der Altbestand keinen.
	for _, g := range groups {
		switch g.ID {
		case "B1":
			if !g.MitBestaetigung || g.BestaetigtAm == nil || !g.BestaetigtAm.Equal(bestaetigt) ||
				g.BestaetigtDurch == nil || *g.BestaetigtDurch != "lieferant" || g.MailGescheitertAm != nil {
				t.Errorf("Stand von B1: %+v", *g)
			}
		case "B2":
			if !g.MitBestaetigung || g.BestaetigtAm != nil || g.BestaetigtDurch != nil ||
				g.MailGescheitertAm == nil || !g.MailGescheitertAm.Equal(gescheitert) {
				t.Errorf("Stand von B2: %+v", *g)
			}
		default:
			if g.MitBestaetigung || g.BestaetigtAm != nil || g.BestaetigtDurch != nil || g.MailGescheitertAm != nil {
				t.Errorf("der Altbestand %q trägt einen Stand: %+v", g.ID, *g)
			}
		}
	}
	// Neueste Bestellung zuerst.
	if groups[0].ID != "B1" && groups[0].ID != "B2" {
		t.Errorf("neueste Gruppe zuerst erwartet, oben steht %q", groups[0].ID)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// Der Altbestand kann denselben Zeitpunkt tragen. Ohne feste Ordnung entschiede die Map, und die
// Gruppen sprängen bei jedem Nachladen, während jemand die Liste liest.
func TestLadeZulauf_GleicherZeitpunktBleibtInDerselbenReihenfolge(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()
	alt := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	for lauf := 0; lauf < 20; lauf++ {
		rows := pgxmock.NewRows(zulaufSpalten)
		for _, lieferant := range []string{"E", "A", "D", "B", "C"} {
			rows.AddRow("ex"+lieferant, "t"+lieferant, alt, "Im Zulauf - "+lieferant, "Titel "+lieferant, "", "", nil, nil, nil, false, nil, nil, nil)
		}
		mock.ExpectQuery(zulaufAbfrage).WillReturnRows(rows)

		gruppen, err := LadeZulauf(context.Background(), mock)
		if err != nil {
			t.Fatalf("LadeZulauf: %v", err)
		}
		namen := make([]string, 0, len(gruppen))
		for _, g := range gruppen {
			namen = append(namen, g.LieferantName)
		}
		if got := strings.Join(namen, ""); got != "ABCDE" {
			t.Fatalf("Lauf %d: Reihenfolge = %s, erwartet ABCDE", lauf, got)
		}
	}
}

func TestLadeZulauf_AbfragefehlerGehtWeiter(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	expectedErr := errors.New("db error")
	mock.ExpectQuery(zulaufAbfrage).WillReturnError(expectedErr)

	_, err = LadeZulauf(context.Background(), mock)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("LadeZulauf(…) = %v, erwartet %v", err, expectedErr)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// Zwei Titel mit demselben Wortlaut in einer Bestellung, etwa Schülerbuch und Arbeitsheft,
// sind zwei Posten: Jeder trägt seine ISBN, sein Cover und seine Exemplare und lässt sich im
// Wareneingang allein ankreuzen.
func TestOrdneExemplarZu_GleicherTitelEigenerPosten(t *testing.T) {
	gruppe := &Zulaufgruppe{}
	ordneExemplarZu(gruppe, exemplarZurGruppe{TitelID: "t1", Titel: "Green Line 3", ISBN: "111", CoverURL: "c1", ExemplarID: "e1"})
	ordneExemplarZu(gruppe, exemplarZurGruppe{TitelID: "t2", Titel: "Green Line 3", ISBN: "222", CoverURL: "c2", ExemplarID: "e2"})
	ordneExemplarZu(gruppe, exemplarZurGruppe{TitelID: "t1", Titel: "Green Line 3", ISBN: "111", CoverURL: "c1", ExemplarID: "e3"})

	if len(gruppe.Posten) != 2 {
		t.Fatalf("%d Posten, erwartet 2: %+v", len(gruppe.Posten), gruppe.Posten)
	}
	erst, zweit := gruppe.Posten[0], gruppe.Posten[1]
	if erst.TitelID != "t1" || erst.ISBN != "111" || erst.Menge != 2 || strings.Join(erst.ExemplarIDs, ",") != "e1,e3" {
		t.Errorf("erster Posten: %+v", *erst)
	}
	if zweit.TitelID != "t2" || zweit.ISBN != "222" || zweit.CoverURL != "c2" || zweit.Menge != 1 ||
		strings.Join(zweit.ExemplarIDs, ",") != "e2" {
		t.Errorf("zweiter Posten: %+v", *zweit)
	}
}
