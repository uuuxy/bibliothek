package repository

import (
	"context"
	"fmt"
	"strings"
)

// LeserListenZeile represents a student along with their current loan statistics.
type LeserListenZeile struct {
	ID            string `json:"id"`
	BarcodeID     string `json:"barcode_id"`
	Vorname       string `json:"vorname"`
	Nachname      string `json:"nachname"`
	Klasse        string `json:"klasse"`
	AbgaengerJahr int    `json:"abgaenger_jahr"`
	IstGesperrt   bool   `json:"ist_gesperrt"`
	// Art ist Schüler, Lehrkraft oder LiV. Die Leserdatei führt sie als eigene Spalte —
	// ein Kollege hat keine Klasse, und ohne die Art stünde er in der Liste wie ein
	// Schüler mit fehlender Angabe.
	Art                string `json:"art"`
	IstVonHandGesperrt bool   `json:"is_manually_blocked"`
	HatFoto            bool   `json:"-"`
	FotoURL            string `json:"foto_url"`
	AnzahlAusgeliehen  int    `json:"ausgeliehen_count"`
	AnzahlUeberfaellig int    `json:"ueberfaellig_count"`
	// AusweisGueltigBis ist das Jahr, bis zu dem der Schülerausweis gilt — aus dem
	// Bildungsgang der Klasse gerechnet, NICHT aus AbgaengerJahr (siehe
	// internal/ausweis). nil heißt: aus dieser Klassenbezeichnung nicht ableitbar,
	// dann fragt der Druckdialog nach.
	AusweisGueltigBis *int `json:"ausweis_gueltig_bis,omitempty"`
}

// LadeKlassen returns a list of all active classes.
func (a *pgLeserAbfragen) LadeKlassen(ctx context.Context) ([]string, error) {
	rows, err := a.db.Query(ctx, "SELECT DISTINCT klasse FROM schueler WHERE klasse != '' AND deleted_at IS NULL ORDER BY klasse")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var klassen []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err == nil {
			klassen = append(klassen, k)
		}
	}
	return klassen, rows.Err()
}

// leserListeMax ist die Obergrenze der ungefilterten Schülerdatei. Sie
// existiert, damit eine Schule mit 20.000 Schülern nicht bei jedem Seitenaufruf die
// komplette Kartei über die Leitung schickt. Wer jemanden dahinter sucht, findet ihn
// über suche — die Suche läuft auf dem Server und kennt keine Grenze bei 500.
const leserListeMax = 500

// LadeSchuelerListe liefert Schüler samt Ausleihzahlen, optional auf eine Klasse
// eingegrenzt und/oder auf einen Suchbegriff (Name oder Barcode).
//
// Gesucht wird über dieselben SQL-Bausteine wie in der Omnibox an der Theke
// (leserSuchCTE/-Bedingung/-Rang, siehe leser_suche.go): Tokens in beliebiger
// Reihenfolge, diakritikfrei, deutsche Ersatzschreibung in beide Richtungen. Vorher
// filterte die Schülerdatei im Browser über die ersten 500 gelieferten Zeilen — wer
// alphabetisch dahinter lag, war über die Suche nicht erreichbar, und zwar abhängig vom
// Klassennamen, also für den Benutzer ohne erkennbares Muster.
// klassen grenzt auf eine ODER MEHRERE Klassen ein (leer = alle). Mehrere, weil ein
// JAHRGANG genau das ist: die Klassen, die zu ihm gehören. Die Zuordnung steht nicht
// hier, sondern an einer Stelle in der API — sie braucht das Klassenschema der Schule
// („ET" ist Jahrgang 11), und das kann SQL nicht.
func (a *pgLeserAbfragen) LadeSchuelerListe(ctx context.Context, klassen []string, suche string, sortierung LeserSortierung) ([]LeserListenZeile, error) {
	return a.ladeListe(ctx, klassen, suche, listeAktiveSchueler, sortierung)
}

// LadeEhemaligenListe liefert die Schüler, die die Schule verlassen haben
// (ist_abgaenger = true, nicht gelöscht) — der Reiter „Ehemalige / Archiv" der
// Schülerdatei. Bis zum 05.09.2026 bettete der Reiter die Abgängerliste ein; seit die
// wieder die Abschlussklassen meint (noch an der Schule), brauchen die Weggegangenen
// eine eigene Liste. Jüngster Abgang zuerst, dann Name.
func (a *pgLeserAbfragen) LadeEhemaligenListe(ctx context.Context, suche string, sortierung LeserSortierung) ([]LeserListenZeile, error) {
	return a.ladeListe(ctx, nil, suche, listeEhemalige, sortierung)
}

// LadeLeserListe liefert ALLE Leser — Schüler und Kollegium — für die Leserdatei.
//
// Dieselben Zeilen und dieselbe Suche wie die Schülerliste, nur über die Tabelle `leser`
// statt über die Sicht `schueler`. Bis zum 16.09.2026 stand ein Kollege in keiner Liste;
// man sah an der Theke, dass er ein Buch bekommen hatte, aber nirgends, welche er hat.
func (a *pgLeserAbfragen) LadeLeserListe(ctx context.Context, klassen []string, suche string, sortierung LeserSortierung) ([]LeserListenZeile, error) {
	return a.ladeListe(ctx, klassen, suche, listeAlleLeser, sortierung)
}

// listenArt sagt, WELCHE Zeilen eine Liste zeigt. Die drei Fälle sind die drei Reiter
// bzw. Ansichten und schließen einander aus; als zwei bool-Schalter nebeneinander wäre
// die vierte, sinnlose Kombination („ehemalige UND alle Leser") jederzeit tippbar.
type listenArt int

const (
	// listeAktiveSchueler: die Schülerdatei, wie sie war — die Sicht `schueler`.
	listeAktiveSchueler listenArt = iota
	// listeEhemalige: der Reiter „Ehemalige / Archiv", ebenfalls nur Schüler. Kollegium
	// hat kein Abgängerjahr und wird vom LUSD-Abgleich nicht angefasst.
	listeEhemalige
	// listeAlleLeser: die Leserdatei über die Tabelle `leser`.
	listeAlleLeser
)

func (a *pgLeserAbfragen) ladeListe(ctx context.Context, klassen []string, suche string, art listenArt, sortierung LeserSortierung) ([]LeserListenZeile, error) {
	// Die Bedingungen werden zusammengesetzt, weil jede für sich optional ist. Die
	// Platzhalternummern stehen fest ($1/$2 Suche, $3 Klasse) und die zugehörigen
	// Argumente werden nur dann angehängt, wenn ihre Bedingung auch im SQL landet —
	// Postgres lehnt sonst überzählige Bind-Parameter ab.
	// Weggegangene (ist_abgaenger) gehören nicht in „Aktive Schüler" (Fund 31.08.2026:
	// sie standen als „Gesperrt" ohne erkennbaren Grund zwischen den Aktiven; nach
	// einem Schuljahreswechsel wäre das ein ganzer Jahrgang Karteileichen) — sie haben
	// den Reiter „Ehemalige / Archiv", der über dieselbe Abfrage mit umgekehrtem
	// Vorzeichen läuft.
	statusBedingung := "COALESCE(s.ist_abgaenger, false) = false"
	if art == listeEhemalige {
		statusBedingung = "s.ist_abgaenger = true"
	}
	// Die Leserdatei liest die TABELLE, die beiden Schüler-Listen die Sicht. Das ist der
	// einzige Unterschied im Rumpf — alles andere (Suche, Ausleihzahlen, Foto) gilt für
	// einen Kollegen genauso.
	quelle := "schueler"
	if art == listeAlleLeser {
		quelle = "leser"
	}
	bedingungen := []string{"s.deleted_at IS NULL", statusBedingung}
	args := []any{}
	praefix := ""
	limit := ""

	if tokens := suchTokens(suche); len(tokens) > 0 {
		praefix = leserSuchCTE
		// mitKlasse: Die Schülerdatei durchsuchte schon immer auch die Klasse ("10a"
		// listet die Klasse). Das darf die Serversuche nicht stillschweigend verlieren.
		bedingungen = append(bedingungen, leserSuchBedingung(true))
		args = append(args, tokens, tokens[0])
	} else {
		// Nur die reine Liste wird gekappt. Eine Suche darf das nicht — genau daran
		// ist sie vorher gescheitert.
		limit = fmt.Sprintf(" LIMIT %d", leserListeMax)
	}

	// = ANY statt =, seit der Filter auch einen ganzen Jahrgang meinen kann (17.09.2026,
	// OFFEN.md 9.5). Eine Klasse ist der Sonderfall mit einem Element; die leere Liste
	// filtert nicht — sonst zeigte ein Jahrgang ohne Klassen versehentlich ALLE Leser
	// statt keinem. Genau umgekehrt wäre es aber richtig, deshalb die Trennung in der
	// API: Ein Jahrgang, den es nicht gibt, kommt dort gar nicht erst hierher.
	if len(klassen) > 0 {
		args = append(args, klassen)
		bedingungen = append(bedingungen, fmt.Sprintf("s.klasse = ANY($%d)", len(args)))
	}

	// Die Reihenfolge — Vorgabe, Suchrang oder die vom Benutzer gewählte Spalte. Die
	// Regeln stehen in leser_sortierung.go, samt der Begründung, warum das Kollegium
	// auch beim Sortieren oben bleibt.
	suchRang := ""
	if praefix != "" {
		suchRang = leserSuchRang
	}
	orderBy := sortierung.sqlSortierung(art, suchRang)

	// coalesce auf Ausweis, Klasse und Abgängerjahr: Seit Migration 123 sind die drei
	// nullbar (ein Kollege hat keine Klasse und oft keinen gedruckten Ausweis). Ohne das
	// zerbricht der Scan an der ersten Lehrkraft — dort, wo eben noch eine Liste stand,
	// stünde ein 500.
	rows, err := a.db.Query(ctx, praefix+`
		SELECT s.id, COALESCE(s.barcode_id, ''), s.vorname, s.nachname, COALESCE(s.klasse, ''),
			COALESCE(s.abgaenger_jahr, 0), s.ist_gesperrt, s.art,
			COALESCE(s.is_manually_blocked, false) as is_manually_blocked,
			COALESCE(l.ausgeliehen_anzahl, 0) as ausgeliehen_anzahl,
			COALESCE(l.ueberfaellig_anzahl, 0) as ueberfaellig_anzahl,
			EXISTS(SELECT 1 FROM schueler_fotos sf WHERE sf.schueler_id = s.id) as has_foto
		FROM `+quelle+` s
		LEFT JOIN LATERAL (
			SELECT
				COUNT(*) as ausgeliehen_anzahl,
				-- Dauerleihen werden NICHT überfällig. Das ist keine neue Regel, sondern
				-- dieselbe, nach der die Sperr-Automatik seit jeher zählt
				-- (ZaehleUeberfaelligeBuecher: ist_handapparat = false). Hier fehlte sie:
				-- Ein Kollege stand nach einem Jahr mit roter Zahl in der Leserdatei,
				-- während die Theke ihn anstandslos bediente — zwei Wahrheiten über
				-- dieselbe Ausleihe. Entschieden am 16.09.2026: „kollegen haben keine
				-- frist bzw werden einfach nie gesperrt!"
				COUNT(*) FILTER (WHERE a.rueckgabe_frist < CURRENT_TIMESTAMP
				                   AND a.ist_handapparat = false) as ueberfaellig_anzahl
			FROM ausleihen a
			WHERE a.schueler_id = s.id AND a.rueckgabe_am IS NULL
		) l ON true
		WHERE `+strings.Join(bedingungen, " AND ")+`
		ORDER BY `+orderBy+limit, args...)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []LeserListenZeile
	for rows.Next() {
		var s LeserListenZeile
		if err := rows.Scan(&s.ID, &s.BarcodeID, &s.Vorname, &s.Nachname, &s.Klasse, &s.AbgaengerJahr, &s.IstGesperrt, &s.Art, &s.IstVonHandGesperrt, &s.AnzahlAusgeliehen, &s.AnzahlUeberfaellig, &s.HatFoto); err != nil {
			return nil, err
		}
		if s.BarcodeID != "" && s.HatFoto {
			s.FotoURL = fmt.Sprintf("/api/schueler/%s/photo", s.BarcodeID)
		}
		// Wie in leseLeserMitZusatz: Die Ausweisgültigkeit wird beim Lesen gerechnet,
		// nicht gespeichert. Der Klassensatz-Druck im Ausweis-Designer liest über diese
		// Liste — ohne die Zeile trüge dort jede Karte "Gültig bis: –".
		s.AusweisGueltigBis = ausweisGueltigBis(s.Klasse)
		stats = append(stats, s)
	}
	return stats, rows.Err()
}
