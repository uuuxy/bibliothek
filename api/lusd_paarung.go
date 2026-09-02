package api

import (
	"sort"
	"strconv"
	"strings"
	"time"
)

// Umbenennung ohne Schüler-ID.
//
// Der LUSD-Export der Schule hat keine Schüler-ID und bekommt so schnell keine. Der
// Import ordnet über Name + Geburtsdatum zu (lusd_parser.go). Ändert die LUSD den Namen
// (Schreibkorrektur, Umlaut, zweiter Vorname fällt weg, Adoption) oder korrigiert sie
// das Geburtsdatum, findet der Schlüssel niemanden mehr: Der Bestandsschüler steht als
// Abgänger da, die Exportzeile als Neuzugang — zwei Datensätze für ein Kind, das seinen
// Ausweis und seine Bücher am alten hat. Littera hatte dasselbe Problem und löste es mit
// einer gedruckten Abgängerliste zum Abhaken.
//
// Hier macht das System den Vorschlag: Aus den Abgängern und den Neuzugängen EINER
// Vorschau werden Paare gebildet, die nach allem, was der Export sonst noch hergibt,
// dieselbe Person sind. Der Admin bestätigt je Paar (LusdImportView); bestätigt heißt,
// der bestehende Datensatz bekommt den neuen Namen bzw. das korrigierte Datum und
// behält UUID, Barcode, Ausleihen und Historie. Unbestätigt bleibt alles wie bisher.
//
// Signale, absteigend nach Gewicht:
//   - Schuleintritt (schul_eintritt_am, Migration 094) — übersteht jede Umbenennung.
//     Ein „Neuzugang", dessen Eintritt Jahre zurückliegt, ist fast sicher keiner.
//   - Geburtsdatum, Name (Vor- oder Nachname), Klasse (gleich oder Nachbarjahrgang),
//     Anschrift (Straße + PLZ).
//
// „sicher" (vorangekreuzt): Eintritt + Geburtsdatum, oder Eintritt + Name.
// „vermutlich" (angeboten, nicht angekreuzt): Geburtsdatum + ein weiteres Signal, oder
// Name + Klasse/Anschrift bei abweichendem Geburtsdatum (Datumskorrektur).
// Jeder Abgänger und jede Zeile stehen in höchstens einem Paar; bei Konkurrenz gewinnt
// das stärkere. Im Nur-Name-Modus gibt es keine Paare — ohne Datum trägt kein Signal.

// UmbenennungDiff ist ein vorgeschlagenes Paar in der Vorschau. Zeile ist die Zeilen-
// nummer der Datei (stabil über Vorschau und Import), SchuelerID der Bestandsdatensatz.
type UmbenennungDiff struct {
	Zeile           int    `json:"zeile"`
	SchuelerID      string `json:"schueler_id"`
	AltVorname      string `json:"alt_vorname"`
	AltNachname     string `json:"alt_nachname"`
	AltKlasse       string `json:"alt_klasse"`
	AltGeburtsdatum string `json:"alt_geburtsdatum,omitempty"`
	NeuVorname      string `json:"neu_vorname"`
	NeuNachname     string `json:"neu_nachname"`
	NeuKlasse       string `json:"neu_klasse"`
	NeuGeburtsdatum string `json:"neu_geburtsdatum,omitempty"`
	Grund           string `json:"grund"`
	Sicher          bool   `json:"sicher"`
	// WarAbgaenger: der Bestandsdatensatz ist schon aus einem früheren Lauf Abgänger
	// (gesperrt, noch nicht anonymisiert) — die Karenzzeit hält ihn dafür offen.
	WarAbgaenger bool `json:"war_abgaenger"`
	// Bestaetigt wird erst beim Import gesetzt: dieses Paar hat der Admin gewählt.
	Bestaetigt bool `json:"bestaetigt"`
}

// umbenennungWahl ist, was der Admin zurückschickt: Zeile → Bestandsdatensatz.
type umbenennungWahl struct {
	Zeile      int    `json:"zeile"`
	SchuelerID string `json:"schueler_id"`
}

type paarKandidat struct {
	zeile   int // Index in datei.Zeilen
	s       *lusdBestandsSchueler
	signale int
	sicher  bool
	grund   string
}

// findeUmbenennungen bildet die Paare. Kandidaten auf der Bestandsseite sind die
// Abgänger dieses Laufs (aktive, die im Export fehlen) und die noch nicht anonymisierten
// Abgänger früherer Läufe; auf der Dateiseite die Zeilen, die neu angelegt würden.
func findeUmbenennungen(datei lusdDatei, bestand []lusdBestandsSchueler, idx lusdIndex, z lusdZuordnung) []UmbenennungDiff {
	if datei.Modus == lusdModusNurName || len(z.neuZeilen) == 0 {
		return nil
	}
	nachID := make(map[string]*lusdBestandsSchueler, len(bestand))
	for i := range bestand {
		nachID[bestand[i].ID] = &bestand[i]
	}
	var seite []*lusdBestandsSchueler
	for _, id := range z.abgaengerIDs {
		if s := nachID[id]; s != nil {
			seite = append(seite, s)
		}
	}
	for _, s := range idx.abgaenger {
		if s != nil && !s.Anonymisiert {
			seite = append(seite, s)
		}
	}

	var kandidaten []paarKandidat
	for _, i := range z.neuZeilen {
		for _, s := range seite {
			if k, ok := bewertePaar(i, datei.Zeilen[i], s); ok {
				kandidaten = append(kandidaten, k)
			}
		}
	}
	return waehlePaare(datei, kandidaten)
}

// bewertePaar sammelt die Signale zwischen einer Exportzeile und einem Bestandsschüler.
func bewertePaar(i int, rec parsedStudentRow, s *lusdBestandsSchueler) (paarKandidat, bool) {
	geb := datumGleich(rec.GebDatum, s.Geburtsdatum)
	eintritt := datumGleich(rec.EintrittAm, s.EintrittAm)
	nachname := namensteilGleich(rec.Nachname, s.Nachname)
	vorname := namensteilGleich(rec.Vorname, s.Vorname)
	name := nachname && vorname
	klasse := klassenNachbar(s.Klasse, rec.Klasse)
	adresse := rec.PLZ != "" && rec.Strasse != "" &&
		normName(rec.PLZ) == normName(s.PLZ) && normName(rec.Strasse) == normName(s.Strasse)

	var gruende []string
	add := func(ok bool, text string) {
		if ok {
			gruende = append(gruende, text)
		}
	}
	add(eintritt, "gleicher Schuleintritt")
	add(geb, "gleiches Geburtsdatum")
	add(name, "gleicher Name")
	add(!name && nachname, "gleicher Nachname")
	add(!name && vorname, "gleicher Vorname")
	add(klasse, "gleiche oder benachbarte Klasse")
	add(adresse, "gleiche Anschrift")

	k := paarKandidat{zeile: i, s: s, signale: len(gruende)}
	switch {
	case eintritt && (geb || name):
		k.sicher = true
	case geb && (nachname || vorname || klasse || adresse):
	case name && (klasse || adresse):
		gruende = append(gruende, "Geburtsdatum abweichend (Korrektur?)")
	default:
		return paarKandidat{}, false
	}
	k.grund = strings.Join(gruende, ", ")
	return k, true
}

// waehlePaare löst Konkurrenz auf: stärkstes Paar zuerst, jeder Abgänger und jede
// Zeile nur einmal. Die Reihenfolge ist deterministisch (Signale, Sicherheit, Zeile).
func waehlePaare(datei lusdDatei, kandidaten []paarKandidat) []UmbenennungDiff {
	sort.SliceStable(kandidaten, func(a, b int) bool {
		ka, kb := kandidaten[a], kandidaten[b]
		if ka.sicher != kb.sicher {
			return ka.sicher
		}
		if ka.signale != kb.signale {
			return ka.signale > kb.signale
		}
		return ka.zeile < kb.zeile
	})
	zeileBelegt, schuelerBelegt := map[int]bool{}, map[string]bool{}
	var paare []UmbenennungDiff
	for _, k := range kandidaten {
		if zeileBelegt[k.zeile] || schuelerBelegt[k.s.ID] {
			continue
		}
		zeileBelegt[k.zeile], schuelerBelegt[k.s.ID] = true, true
		rec := datei.Zeilen[k.zeile]
		paare = append(paare, UmbenennungDiff{
			Zeile: rec.LineNum, SchuelerID: k.s.ID,
			AltVorname: k.s.Vorname, AltNachname: k.s.Nachname, AltKlasse: k.s.Klasse,
			AltGeburtsdatum: datumText(k.s.Geburtsdatum),
			NeuVorname:      rec.Vorname, NeuNachname: rec.Nachname, NeuKlasse: rec.Klasse,
			NeuGeburtsdatum: datumText(rec.GebDatum),
			Grund:           k.grund, Sicher: k.sicher, WarAbgaenger: k.s.IstAbgaenger,
		})
	}
	sort.Slice(paare, func(a, b int) bool { return paare[a].Zeile < paare[b].Zeile })
	return paare
}

// uebernimmUmbenennungen wendet die Wahl des Admins auf Zuordnung und Vorschau an.
// Nur Paare, die diese Vorschau selbst vorgeschlagen hat, zählen — eine fremde
// Kombination (verändertes Formular, veraltete Vorschau) wird abgewiesen, nicht geraten.
func uebernimmUmbenennungen(datei lusdDatei, wahl []umbenennungWahl, paare []UmbenennungDiff, z *lusdZuordnung, res *LusdPreviewResult) error {
	if len(wahl) == 0 {
		return nil
	}
	zeilenIndex := make(map[int]int, len(datei.Zeilen))
	for i, rec := range datei.Zeilen {
		zeilenIndex[rec.LineNum] = i
	}
	vorgeschlagen := make(map[int]int, len(paare)) // Zeile → Index in paare
	for i, p := range paare {
		vorgeschlagen[p.Zeile] = i
	}
	for _, w := range wahl {
		pi, ok := vorgeschlagen[w.Zeile]
		if !ok || paare[pi].SchuelerID != w.SchuelerID {
			return &errUmbenennungUngueltig{Zeile: w.Zeile}
		}
		p := &paare[pi]
		if p.Bestaetigt {
			continue
		}
		p.Bestaetigt = true
		i := zeilenIndex[w.Zeile]
		z.zielID[i] = p.SchuelerID
		z.geburtsdatumSetzen[i] = true
		z.abgaengerIDs = entferneID(z.abgaengerIDs, p.SchuelerID)
		res.Graduates = entferneDiff(res.Graduates, p.SchuelerID)
		res.NewStudents = entferneDiff(res.NewStudents, "zeile-"+strconv.Itoa(w.Zeile))
	}
	return nil
}

type errUmbenennungUngueltig struct{ Zeile int }

func (e *errUmbenennungUngueltig) Error() string {
	return "Zuordnung für Zeile " + strconv.Itoa(e.Zeile) + " ist nicht mehr gültig — Vorschau neu laden."
}

func entferneID(ids []string, id string) []string {
	out := ids[:0]
	for _, x := range ids {
		if x != id {
			out = append(out, x)
		}
	}
	return out
}

func entferneDiff(liste []StudentDiff, id string) []StudentDiff {
	out := liste[:0]
	for _, d := range liste {
		if d.ID != id {
			out = append(out, d)
		}
	}
	return out
}

// ── Vergleichshelfer ─────────────────────────────────────────────────────────

func datumGleich(a, b *time.Time) bool {
	return a != nil && b != nil && a.Format("2006-01-02") == b.Format("2006-01-02")
}

func datumText(d *time.Time) string {
	if d == nil {
		return ""
	}
	return d.Format("2006-01-02")
}

// normName ebnet ein, was eine Umbenennung typischerweise ändert: Groß/Klein, Umlaut-
// Schreibung, Bindestrich, Leerzeichen. „Al-Sayed" und „Al Sayed", „Müller" und
// „Mueller" gelten als gleich.
func normName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.NewReplacer("ä", "ae", "ö", "oe", "ü", "ue", "ß", "ss", "-", "", " ", "", ".", "").Replace(s)
	return s
}

// namensteilGleich: gleich nach Normierung, oder der eine ist Anfang des anderen (ein
// zweiter Vorname kommt hinzu oder fällt weg: „Ayman Sharafudin" ↔ „Ayman").
func namensteilGleich(a, b string) bool {
	na, nb := normName(a), normName(b)
	if na == "" || nb == "" {
		return false
	}
	return na == nb || strings.HasPrefix(na, nb) || strings.HasPrefix(nb, na)
}

// klassenNachbar: gleiche Klasse (Normkey) oder derselbe Zug im Nachbarjahrgang —
// „05F1" und „06F1" sind beim Schuljahreswechsel dieselbe Klasse ein Jahr später.
func klassenNachbar(a, b string) bool {
	ka, kb := klassenNormkey(a), klassenNormkey(b)
	if ka == kb {
		return true
	}
	ja, ra := jahrgangUndRest(ka)
	jb, rb := jahrgangUndRest(kb)
	if ja < 0 || jb < 0 || ra != rb {
		return false
	}
	return ja-jb == 1 || jb-ja == 1
}

func jahrgangUndRest(normkey string) (int, string) {
	i := 0
	for i < len(normkey) && normkey[i] >= '0' && normkey[i] <= '9' {
		i++
	}
	if i == 0 {
		return -1, normkey
	}
	j, err := strconv.Atoi(normkey[:i])
	if err != nil {
		return -1, normkey
	}
	return j, normkey[i:]
}
