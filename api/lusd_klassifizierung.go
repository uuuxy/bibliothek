package api

import (
	"fmt"
	"regexp"
	"strings"
)

// lusdZuordnung ist das Ergebnis der Klassifizierung — je CSV-Zeile das Ziel. Die
// Vorschau (LusdPreviewResult) ist dieselbe Entscheidung, nur für Menschen erzählt.
type lusdZuordnung struct {
	zielID        map[int]string // Zeilenindex → schueler.id: Bestand aktualisieren
	ueberspringen map[int]bool   // Zeilenindex → nicht anfassen (ohne ID, mehrdeutig)
	adoptionen    []AdoptionDiff // ID-Modus: LUSD-ID anheften; Name+Geb-Modus: Geburtsdatum nachtragen
	abgaengerIDs  []string
	// neuZeilen sind die Zeilenindizes, die als Neuzugang enden würden — die eine Seite der
	// Umbenennungs-Paarung (lusd_paarung.go). geburtsdatumSetzen markiert bestätigte
	// Paare: Nur dort schreibt der Bestands-Batch das Geburtsdatum aus dem Export (eine
	// Datumskorrektur der LUSD), sonst bliebe der Schlüssel beim nächsten Lauf falsch.
	neuZeilen          []int
	geburtsdatumSetzen map[int]bool
	// datumNachgetragen: Bestandsschüler ohne Geburtsdatum, die im Name+Geb-Modus über
	// den Namen zugeordnet wurden — sie gehören nicht mehr in „nicht abgleichbar".
	datumNachgetragen map[string]bool
}

var fuehrendeNullen = regexp.MustCompile(`^0+(\d)`)

// klassenNormkey spiegelt die SQL-Funktion klassen_normkey aus Migration 079 (klein,
// ohne Leerzeichen, ohne führende Nullen vor Ziffern). Der BEFORE-Trigger kanonisiert
// jede geschriebene Klasse auf die registrierte Schreibweise — "05A" aus der LUSD wird
// zu "5a", wenn die Schule so schreibt. Vergliche der Import roh, stünde in jeder
// Vorschau ein Klassenwechsel "5a → 05A", der keiner ist. Die Parität zur SQL-Fassung
// hält lusd_klassen_normkey_pg_test.go.
func klassenNormkey(klasse string) string {
	s := strings.ToLower(strings.ReplaceAll(strings.Trim(klasse, " "), " ", ""))
	return fuehrendeNullen.ReplaceAllString(s, "$1")
}

func klassenGleich(a, b string) bool { return klassenNormkey(a) == klassenNormkey(b) }

// diffZeile baut den Vorschau-Eintrag einer CSV-Zeile. id ist der Listenschlüssel fürs
// Frontend: die LUSD-ID (ID-Modus) oder die schueler-UUID — bei Neuzugängen im
// Namensmodus die Zeilennummer, denn sonst gäbe es nichts Stabiles.
func diffZeile(id string, rec parsedStudentRow, alteKlasse, neueKlasse string) StudentDiff {
	return StudentDiff{ID: id, Vorname: rec.Vorname, Nachname: rec.Nachname, AlteKlasse: alteKlasse, NeueKlasse: neueKlasse}
}

func diffBestand(s *lusdBestandsSchueler, id string) StudentDiff {
	return StudentDiff{ID: id, Vorname: s.Vorname, Nachname: s.Nachname, AlteKlasse: s.Klasse}
}

// lusdKlassifizierer bündelt den Kontext für die Klassifizierung, um Parameterlisten kurz zu halten.
type lusdKlassifizierer struct {
	modus   lusdModus
	idx     lusdIndex
	res     *LusdPreviewResult
	z       *lusdZuordnung
	gesehen map[string]bool
}

// klassifiziereLusd ordnet die CSV-Zeilen (rein klassifizierend, ohne Schreibzugriff)
// ein und füllt Vorschau und Zuordnung in einem Durchgang.
func klassifiziereLusd(datei lusdDatei, idx lusdIndex, res *LusdPreviewResult) lusdZuordnung {
	z := lusdZuordnung{zielID: map[int]string{}, ueberspringen: map[int]bool{}, datumNachgetragen: map[string]bool{}, geburtsdatumSetzen: map[int]bool{}}
	gesehen := map[string]bool{}
	namenInDatei := zaehleNamenInDatei(datei)
	k := lusdKlassifizierer{modus: datei.Modus, idx: idx, res: res, z: &z, gesehen: gesehen}
	for i, rec := range datei.Zeilen {
		switch datei.Modus {
		case lusdModusID:
			k.klassifiziereZeileID(i, rec)
		case lusdModusName:
			k.klassifiziereZeileName(i, rec)
		default:
			// Nur-Name: Derselbe Name zweimal in der Datei sind zwei Menschen, die sich
			// nicht auseinanderhalten lassen — beide melden, keinen anfassen.
			if namenInDatei[rec.namensschluessel()] > 1 {
				res.Mehrdeutig = append(res.Mehrdeutig, diffZeile(fmt.Sprintf("zeile-%d", rec.LineNum), rec, "", rec.Klasse))
				z.ueberspringen[i] = true
				// Der Name STEHT im Export — ein bestätigter Bestandsschüler dieses Namens
				// ist also nicht „nicht im Export". Ohne diese Zeile machte sammleAbgaenger
				// ihn zum Abgänger und Apply anonymisierte ihn (Prüfung 22.08.2026, A1).
				gesehen[rec.namensschluessel()] = true
				continue
			}
			k.klassifiziereZeileName(i, rec)
		}
	}
	k.sammleAbgaenger()
	return z
}

// zaehleNamenInDatei zählt im Nur-Name-Modus, wie oft jeder Name in der Datei steht.
func zaehleNamenInDatei(datei lusdDatei) map[string]int {
	n := map[string]int{}
	if datei.Modus != lusdModusNurName {
		return n
	}
	for _, rec := range datei.Zeilen {
		n[rec.namensschluessel()]++
	}
	return n
}

// klassifiziereZeileID: Schlüssel LUSD-ID. Reihenfolge: aktiver Bestand → Rückkehrer
// (Abgänger mit dieser ID) → Adoption (ID-loser Schüler gleichen Namens+Geburtsdatums)
// → Neuzugang.
func (k *lusdKlassifizierer) klassifiziereZeileID(i int, rec parsedStudentRow) {
	if rec.LusdID == "" {
		k.res.SkippedNoID++ // ohne LUSD-ID gibt es keinen stabilen Schlüssel — sichtbar zählen
		k.z.ueberspringen[i] = true
		return
	}
	k.gesehen[rec.LusdID] = true
	if s := k.idx.aktiv[rec.LusdID]; s != nil {
		k.z.zielID[i] = s.ID
		if !klassenGleich(s.Klasse, rec.Klasse) {
			k.res.ClassChanges = append(k.res.ClassChanges, diffZeile(rec.LusdID, rec, s.Klasse, rec.Klasse))
		}
		return
	}
	if s := k.idx.abgaenger[rec.LusdID]; s != nil {
		k.z.zielID[i] = s.ID
		k.res.Rueckkehrer = append(k.res.Rueckkehrer, diffZeile(rec.LusdID, rec, s.Klasse, rec.Klasse))
		return
	}
	key := rec.schluessel()
	if w := k.idx.waisen[key]; key != "" && w != nil {
		k.z.adoptionen = append(k.z.adoptionen, AdoptionDiff{
			SchuelerID: w.ID, LusdID: rec.LusdID, Vorname: rec.Vorname, Nachname: rec.Nachname,
			Geburtsdatum: rec.GebDatum.Format("2006-01-02"), AlteKlasse: w.Klasse, NeueKlasse: rec.Klasse,
		})
		delete(k.idx.waisen, key) // konsumiert — zwei CSV-Zeilen beanspruchen nie denselben Waisen
		return
	}
	k.res.NewStudents = append(k.res.NewStudents, diffZeile(rec.LusdID, rec, "", rec.Klasse))
	k.z.neuZeilen = append(k.z.neuZeilen, i)
}

// klassifiziereZeileName: Schlüssel Name+Geburtsdatum oder nur Name (key kommt vom
// Aufrufer, vom Parser garantiert nicht leer). Mehrdeutige Treffer werden übersprungen
// und gemeldet — ein Neuzugang an ihrer Stelle liefe ohnehin am Unique-Index
// unique_schueler_name_gebdatum auf oder wäre im Nur-Name-Modus ein Ratespiel.
//
// nurName: Im Nur-Name-Modus ist ein Abgänger mit demselben Namen KEIN sicherer
// Rückkehrer — ebenso gut ein neuer Fünftklässler, der sonst auf dem Datensatz (Sperre,
// Schulden, Lesehistorie) des Abgegangenen landete (Prüfung 22.08.2026, A2). Er wird als
// mehrdeutig gemeldet und nicht angefasst; das Sekretariat entscheidet von Hand.
func (k *lusdKlassifizierer) klassifiziereZeileName(i int, rec parsedStudentRow) {
	// Schlüssel AUS dem Modus ableiten (rec.schluesselFuer), nicht vom Aufrufer entgegennehmen:
	// Die Gegenseite (bestandsSchluessel in lusd_bestand.go) tut dasselbe. Kämen beide aus
	// verschiedenen Händen, könnte ein Aufrufer den Namensschlüssel gegen den Name+Datum-Index
	// schlagen — das matcht dann still niemanden. Nebenbei fällt der achte Parameter weg und
	// aus dem nichtssagenden `…, gesehen, true)` am Aufrufer wird der benannte Modus.
	nurName := k.modus == lusdModusNurName
	key := rec.schluesselFuer(k.modus)
	zeilenID := fmt.Sprintf("zeile-%d", rec.LineNum)
	if s, ok := k.idx.aktiv[key]; ok {
		if s == nil {
			k.res.Mehrdeutig = append(k.res.Mehrdeutig, diffZeile(zeilenID, rec, "", rec.Klasse))
			k.z.ueberspringen[i] = true
			return
		}
		k.gesehen[key] = true
		k.z.zielID[i] = s.ID
		if !klassenGleich(s.Klasse, rec.Klasse) {
			k.res.ClassChanges = append(k.res.ClassChanges, diffZeile(s.ID, rec, s.Klasse, rec.Klasse))
		}
		return
	}
	if s, ok := k.idx.abgaenger[key]; ok {
		if s == nil || nurName {
			alteKlasse := ""
			if s != nil {
				alteKlasse = s.Klasse
			}
			k.res.Mehrdeutig = append(k.res.Mehrdeutig, diffZeile(zeilenID, rec, alteKlasse, rec.Klasse))
			k.z.ueberspringen[i] = true
			return
		}
		k.z.zielID[i] = s.ID
		k.res.Rueckkehrer = append(k.res.Rueckkehrer, diffZeile(s.ID, rec, s.Klasse, rec.Klasse))
		return
	}
	// Rückfallstufe (nur Name+Geb-Modus): Bestandsschüler OHNE Geburtsdatum über den
	// Namen — eindeutig → zuordnen und das Datum aus dem Export nachtragen; mehrdeutig
	// → melden, nichts anlegen. Ohne diese Stufe wurde beim Modus-Wechsel (erst LANIS-
	// Liste, später Export mit Datum) jeder Bestandsschüler als „neu" dupliziert.
	if !nurName && rec.GebDatum != nil {
		nameKey := rec.namensschluessel()
		if w, ok := k.idx.ohneDatumNachName[nameKey]; ok {
			if w == nil {
				k.res.Mehrdeutig = append(k.res.Mehrdeutig, diffZeile(zeilenID, rec, "", rec.Klasse))
				k.z.ueberspringen[i] = true
				return
			}
			k.z.zielID[i] = w.ID
			k.z.datumNachgetragen[w.ID] = true
			k.z.adoptionen = append(k.z.adoptionen, AdoptionDiff{
				SchuelerID: w.ID, Vorname: rec.Vorname, Nachname: rec.Nachname,
				Geburtsdatum: rec.GebDatum.Format("2006-01-02"), AlteKlasse: w.Klasse, NeueKlasse: rec.Klasse,
			})
			delete(k.idx.ohneDatumNachName, nameKey) // konsumiert — zwei Zeilen beanspruchen nie denselben
			return
		}
	}
	k.res.NewStudents = append(k.res.NewStudents, diffZeile(zeilenID, rec, "", rec.Klasse))
	k.z.neuZeilen = append(k.z.neuZeilen, i)
}

// sammleAbgaenger: Wer aktiv ist und nicht im Export steht, geht ab — im ID-Modus jeder
// mit echter LUSD-ID, in den Namensmodi nur, wer schon einmal von einem Export BESTÄTIGT
// wurde (lusd_bestaetigt_am). Nie bestätigte Handanlagen bleiben stehen und werden als
// „nicht im Export" gemeldet; Schüler ohne Geburtsdatum sind nicht abgleichbar.
func (k *lusdKlassifizierer) sammleAbgaenger() {
	for key, s := range k.idx.aktiv {
		if s == nil || k.gesehen[key] {
			continue
		}
		if k.modus != lusdModusID && !s.LusdBestaetigt {
			k.res.NichtImExport = append(k.res.NichtImExport, diffBestand(s, s.ID))
			continue
		}
		listenID := key // ID-Modus: die LUSD-ID
		if k.modus != lusdModusID {
			listenID = s.ID
		}
		k.res.Graduates = append(k.res.Graduates, diffBestand(s, listenID))
		k.z.abgaengerIDs = append(k.z.abgaengerIDs, s.ID)
	}
	for i := range k.idx.ohneSchluessel {
		if k.z.datumNachgetragen[k.idx.ohneSchluessel[i].ID] {
			continue // über den Namen zugeordnet, Datum wird nachgetragen
		}
		k.res.NichtAbgleichbar = append(k.res.NichtAbgleichbar, diffBestand(&k.idx.ohneSchluessel[i], k.idx.ohneSchluessel[i].ID))
	}
	for i := range k.idx.mehrdeutigAktiv {
		k.res.Mehrdeutig = append(k.res.Mehrdeutig, diffBestand(&k.idx.mehrdeutigAktiv[i], k.idx.mehrdeutigAktiv[i].ID))
	}
}
