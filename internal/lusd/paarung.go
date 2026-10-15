package lusd

import (
	"sort"
	"strconv"
	"strings"
	"time"
)

// Umbenennung ohne Schüler-ID.
//
// Der LUSD-Export der Schule hat keine Schüler-ID und bekommt so schnell keine. Der
// Import ordnet über Name + Geburtsdatum zu (datei.go). Ändert die LUSD den Namen
// (Schreibkorrektur, Umlaut, zweiter Vorname fällt weg, Adoption) oder korrigiert sie
// das Geburtsdatum, findet der Schlüssel niemanden mehr: Der Bestandsschüler steht als
// Abgänger da, die Exportzeile als Neuzugang: zwei Datensätze für ein Kind, das seinen
// Ausweis und seine Bücher am alten hat.
//
// Hier macht das System den Vorschlag: Aus den Abgängern und den Neuzugängen einer
// Vorschau werden Paare gebildet, die nach allem, was der Export sonst noch hergibt,
// dieselbe Person sind. Der Admin bestätigt je Paar (LusdImport); bestätigt heißt,
// der bestehende Datensatz bekommt den neuen Namen bzw. das korrigierte Datum und
// behält UUID, Barcode, Ausleihen und Historie. Unbestätigt bleibt alles, wie es ist.
//
// Signale, absteigend nach Gewicht:
//   - Schuleintritt (schul_eintritt_am): übersteht jede Umbenennung.
//     Ein „Neuzugang", dessen Eintritt Jahre zurückliegt, ist fast sicher keiner.
//   - Geburtsdatum, Name (Vor- oder Nachname), Klasse (gleich oder Nachbarjahrgang),
//     Anschrift (Straße + PLZ).
//
// „sicher" (vorangekreuzt): Eintritt + Name, oder Eintritt + Geburtsdatum + Vorname.
// Ein abweichender Vorname bei gleichem Geburtsdatum ist ein Gegen-Signal (Zwillinge!):
// dann höchstens „vermutlich". Bei Gleichstand zweier Kandidaten wird kein Paar gebildet.
// „vermutlich" (angeboten, nicht angekreuzt): Geburtsdatum + ein weiteres Signal, oder
// Name + Klasse/Anschrift bei abweichendem Geburtsdatum (Datumskorrektur).
// Jeder Abgänger und jede Zeile stehen in höchstens einem Paar; bei Konkurrenz gewinnt
// das stärkere. Paare gibt es nur im Modus Name + Geburtsdatum: Im Nur-Name-Modus trägt
// ohne Datum kein Signal, im ID-Modus löst die LUSD-ID jede Umbenennung selbst.

// VorschauUmbenennung ist ein vorgeschlagenes Paar in der Vorschau. Zeile ist die Zeilen-
// nummer der Datei (stabil über Vorschau und Import), SchuelerID der Bestandsdatensatz.
type VorschauUmbenennung struct {
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

// UmbenennungWahl ist, was der Admin zurückschickt: Zeile → Bestandsdatensatz.
type UmbenennungWahl struct {
	Zeile      int    `json:"zeile"`
	SchuelerID string `json:"schueler_id"`
}

type paarKandidat struct {
	zeile   int // Index in datei.Zeilen
	s       *bestandsSchueler
	signale int
	sicher  bool
	grund   string
}

// findeUmbenennungen bildet die Paare. Kandidaten auf der Bestandsseite sind die
// Abgänger dieses Laufs (aktive, die im Export fehlen) und die noch nicht anonymisierten
// Abgänger früherer Läufe; auf der Dateiseite die Zeilen, die neu angelegt würden.
//
// Nur im Modus Name + Geburtsdatum: Im ID-Modus findet die LUSD-ID eine Umbenennung selbst;
// ein Paar behielte dort die alte lusd_id und erschiene beim nächsten Lauf erneut als
// Abgänger und Neuzugang.
func findeUmbenennungen(datei Datei, bestand []bestandsSchueler, index bestandsIndex, z Zuordnung) []VorschauUmbenennung {
	if datei.Modus != ModusName || len(z.neuZeilen) == 0 {
		return nil
	}
	seite := umbenennungsSeite(bestand, index, z)

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

// umbenennungsSeite sammelt die Bestandsschüler, die für ein Paar in Frage kommen, nach ID
// sortiert. Wen dieser Lauf schon per Schlüssel zuordnet (Rückkehrer in z.zielID), der
// bleibt draußen: Sonst zeigten zwei Zeilen auf dieselbe ID.
func umbenennungsSeite(bestand []bestandsSchueler, index bestandsIndex, z Zuordnung) []*bestandsSchueler {
	nachID := make(map[string]*bestandsSchueler, len(bestand))
	for i := range bestand {
		nachID[bestand[i].ID] = &bestand[i]
	}
	schonZugeordnet := make(map[string]bool, len(z.zielID))
	for _, id := range z.zielID {
		schonZugeordnet[id] = true
	}
	var seite []*bestandsSchueler
	for _, id := range z.abgaengerIDs {
		if s := nachID[id]; s != nil && !schonZugeordnet[id] {
			seite = append(seite, s)
		}
	}
	for _, s := range index.abgaenger {
		if s != nil && !s.Anonymisiert && !schonZugeordnet[s.ID] {
			seite = append(seite, s)
		}
	}
	sort.Slice(seite, func(a, b int) bool { return seite[a].ID < seite[b].ID })
	return seite
}

// paarSignale hält, worin eine Exportzeile und ein Bestandsschüler übereinstimmen.
type paarSignale struct {
	eintritt, geb, nachname, vorname, klasse, adresse bool
}

func lesePaarSignale(zeile Zeile, s *bestandsSchueler) paarSignale {
	return paarSignale{
		eintritt: istDatumGleich(zeile.EintrittAm, s.EintrittAm),
		geb:      istDatumGleich(zeile.Geburtsdatum, s.Geburtsdatum),
		nachname: istNamensteilGleich(zeile.Nachname, s.Nachname),
		vorname:  istNamensteilGleich(zeile.Vorname, s.Vorname),
		klasse:   istKlassenNachbar(s.Klasse, zeile.Klasse),
		adresse: zeile.PLZ != "" && zeile.Strasse != "" &&
			normierterName(zeile.PLZ) == normierterName(s.PLZ) && normierterName(zeile.Strasse) == normierterName(s.Strasse),
	}
}

func (p paarSignale) istNameGleich() bool { return p.nachname && p.vorname }

// gruende nennt die Übereinstimmungen, absteigend nach Gewicht. Ein voller Name steht für
// seine beiden Teile.
func (p paarSignale) gruende() []string {
	var gruende []string
	for _, g := range []struct {
		gilt bool
		text string
	}{
		{p.eintritt, "gleicher Schuleintritt"},
		{p.geb, "gleiches Geburtsdatum"},
		{p.istNameGleich(), "gleicher Name"},
		{!p.istNameGleich() && p.nachname, "gleicher Nachname"},
		{!p.istNameGleich() && p.vorname, "gleicher Vorname"},
		{p.klasse, "gleiche oder benachbarte Klasse"},
		{p.adresse, "gleiche Anschrift"},
	} {
		if g.gilt {
			gruende = append(gruende, g.text)
		}
	}
	return gruende
}

// bewertePaar sammelt die Signale zwischen einer Exportzeile und einem Bestandsschüler.
func bewertePaar(i int, zeile Zeile, s *bestandsSchueler) (paarKandidat, bool) {
	p := lesePaarSignale(zeile, s)
	gruende := p.gruende()
	k := paarKandidat{zeile: i, s: s, signale: len(gruende)}
	switch {
	case p.eintritt && (p.istNameGleich() || (p.geb && p.vorname)):
		k.sicher = true
	case p.geb && (p.nachname || p.vorname || p.klasse || p.adresse || p.eintritt):
	case p.istNameGleich() && (p.klasse || p.adresse):
		gruende = append(gruende, "Geburtsdatum abweichend (Korrektur?)")
	default:
		return paarKandidat{}, false
	}
	// Gegen-Signal: gleiches Geburtsdatum, aber ein Vorname, der weder gleich noch Anfang des
	// anderen ist. So sehen Zwillinge aus und ein Tippfehler im Vornamen; das Paar wird
	// angeboten, aber nie vorangekreuzt.
	if p.geb && !p.vorname && normierterName(zeile.Vorname) != "" && normierterName(s.Vorname) != "" {
		gruende = append(gruende, "Vorname abweichend (Zwilling?)")
	}
	k.grund = strings.Join(gruende, ", ")
	return k, true
}

// waehlePaare löst Konkurrenz auf: stärkstes Paar zuerst, jeder Abgänger und jede
// Zeile nur einmal. Die Reihenfolge ist deterministisch (Sicherheit, Signale, Zeile,
// Schüler-ID) — sie darf nicht von der Map-Reihenfolge des Bestands abhängen, sonst
// fiele ein Paar mal auf den einen, mal auf den anderen Zwilling.
//
// Gleichstand: Stehen für eine Zeile zwei noch freie Bestandsschüler mit exakt gleich
// starken Signalen (oder ein Bestandsschüler für zwei gleich starke Zeilen), wird nicht
// gewürfelt, sondern kein Paar gebildet — das Kind bleibt Abgänger + Neuzugang, und der
// Admin führt bei Bedarf über die Akte zusammen.
func waehlePaare(datei Datei, kandidaten []paarKandidat) []VorschauUmbenennung {
	sort.SliceStable(kandidaten, func(a, b int) bool {
		ka, kb := kandidaten[a], kandidaten[b]
		if ka.sicher != kb.sicher {
			return ka.sicher
		}
		if ka.signale != kb.signale {
			return ka.signale > kb.signale
		}
		if ka.zeile != kb.zeile {
			return ka.zeile < kb.zeile
		}
		return ka.s.ID < kb.s.ID
	})
	zeileBelegt, schuelerBelegt := map[int]bool{}, map[string]bool{}
	var paare []VorschauUmbenennung
	for i, k := range kandidaten {
		if zeileBelegt[k.zeile] || schuelerBelegt[k.s.ID] {
			continue
		}
		if hatGleichstand(kandidaten[i+1:], k, zeileBelegt, schuelerBelegt) {
			zeileBelegt[k.zeile], schuelerBelegt[k.s.ID] = true, true
			continue
		}
		zeileBelegt[k.zeile], schuelerBelegt[k.s.ID] = true, true
		zeile := datei.Zeilen[k.zeile]
		paare = append(paare, VorschauUmbenennung{
			Zeile: zeile.Nr, SchuelerID: k.s.ID,
			AltVorname: k.s.Vorname, AltNachname: k.s.Nachname, AltKlasse: k.s.Klasse,
			AltGeburtsdatum: datumText(k.s.Geburtsdatum),
			NeuVorname:      zeile.Vorname, NeuNachname: zeile.Nachname, NeuKlasse: zeile.Klasse,
			NeuGeburtsdatum: datumText(zeile.Geburtsdatum),
			Grund:           k.grund, Sicher: k.sicher, WarAbgaenger: k.s.IstAbgaenger,
		})
	}
	sort.Slice(paare, func(a, b int) bool { return paare[a].Zeile < paare[b].Zeile })
	return paare
}

// hatGleichstand: gibt es unter den schwächeren oder gleich starken Kandidaten einen
// noch freien mit derselben Zeile oder demselben Schüler und exakt gleicher Stärke?
func hatGleichstand(rest []paarKandidat, k paarKandidat, zeileBelegt map[int]bool, schuelerBelegt map[string]bool) bool {
	for _, o := range rest {
		if o.sicher != k.sicher || o.signale != k.signale {
			return false // sortiert: ab hier nur noch schwächere
		}
		selbeZeile := o.zeile == k.zeile && o.s.ID != k.s.ID && !schuelerBelegt[o.s.ID]
		selberSchueler := o.s.ID == k.s.ID && o.zeile != k.zeile && !zeileBelegt[o.zeile]
		if selbeZeile || selberSchueler {
			return true
		}
	}
	return false
}

// uebernimmUmbenennungen wendet die Wahl des Admins auf Zuordnung und Vorschau an.
// Nur Paare, die diese Vorschau selbst vorgeschlagen hat, zählen — eine fremde
// Kombination (verändertes Formular, veraltete Vorschau) wird abgewiesen, nicht geraten.
func uebernimmUmbenennungen(datei Datei, wahl []UmbenennungWahl, paare []VorschauUmbenennung, z *Zuordnung, vorschau *Vorschau) error {
	if len(wahl) == 0 {
		return nil
	}
	zeilenIndex := make(map[int]int, len(datei.Zeilen))
	for i, zeile := range datei.Zeilen {
		zeilenIndex[zeile.Nr] = i
	}
	vorgeschlagen := make(map[int]int, len(paare)) // Zeile → Index in paare
	for i, p := range paare {
		vorgeschlagen[p.Zeile] = i
	}
	for _, w := range wahl {
		pi, ok := vorgeschlagen[w.Zeile]
		if !ok || paare[pi].SchuelerID != w.SchuelerID {
			return &UmbenennungUngueltigFehler{Zeile: w.Zeile}
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
		vorschau.Abgaenger = entferneAusVorschau(vorschau.Abgaenger, p.SchuelerID)
		vorschau.Neuzugaenge = entferneAusVorschau(vorschau.Neuzugaenge, "zeile-"+strconv.Itoa(w.Zeile))
	}
	return nil
}

// UmbenennungUngueltigFehler meldet eine gewählte Umbenennung, die es im Lauf nicht mehr
// gibt: Der Bestand hat sich seit der Vorschau geändert.
type UmbenennungUngueltigFehler struct{ Zeile int }

func (e *UmbenennungUngueltigFehler) Error() string {
	return "zuordnung für zeile " + strconv.Itoa(e.Zeile) + " nicht mehr gültig"
}

func entferneID(ids []string, id string) []string {
	rest := ids[:0]
	for _, x := range ids {
		if x != id {
			rest = append(rest, x)
		}
	}
	return rest
}

func entferneAusVorschau(liste []VorschauSchueler, id string) []VorschauSchueler {
	rest := liste[:0]
	for _, d := range liste {
		if d.ID != id {
			rest = append(rest, d)
		}
	}
	return rest
}

// ── Vergleichshelfer ─────────────────────────────────────────────────────────

func istDatumGleich(a, b *time.Time) bool {
	return a != nil && b != nil && a.Format(datumsformatISO) == b.Format(datumsformatISO)
}

func datumText(d *time.Time) string {
	if d == nil {
		return ""
	}
	return d.Format(datumsformatISO)
}

// normierterName ebnet ein, was eine Umbenennung typischerweise ändert: Groß/Klein, Umlaut-
// Schreibung, Bindestrich, Leerzeichen. „Al-Sayed" und „Al Sayed", „Müller" und
// „Mueller" gelten als gleich.
func normierterName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.NewReplacer("ä", "ae", "ö", "oe", "ü", "ue", "ß", "ss", "-", "", " ", "", ".", "").Replace(s)
	return s
}

// istNamensteilGleich: gleich nach Normierung, oder der eine ist Anfang des anderen (ein
// zweiter Vorname kommt hinzu oder fällt weg: „Ayman Sharafudin" ↔ „Ayman").
func istNamensteilGleich(a, b string) bool {
	na, nb := normierterName(a), normierterName(b)
	if na == "" || nb == "" {
		return false
	}
	return na == nb || strings.HasPrefix(na, nb) || strings.HasPrefix(nb, na)
}

// istKlassenNachbar: gleiche Klasse (Normkey) oder derselbe Zug im Nachbarjahrgang —
// „05F1" und „06F1" sind beim Schuljahreswechsel dieselbe Klasse ein Jahr später.
func istKlassenNachbar(a, b string) bool {
	ka, kb := KlassenSchluessel(a), KlassenSchluessel(b)
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
