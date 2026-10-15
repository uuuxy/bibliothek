package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"unicode"
)

// stilZahl zählt je Paket, was den Regeln aus docs/STIL.md noch nicht folgt.
type stilZahl struct {
	Fach    int // Namen mit einem abgelösten englischen Fachwort (STIL.md, Abschnitt 2)
	Verb    int // Funktionen, die mit einem abgelösten englischen Verb beginnen (Abschnitt 3)
	Kuerzel int // Namen mit einem gemischt geschriebenen Kürzel wie Lmf (Abschnitt 4)
	Datei   int // Dateinamen mit einem abgelösten Fachwort (Abschnitt 7)
	Datum   int // Kommentarzeilen mit einem Datum (Abschnitt 8)
}

// stilFachwoerter sind die englischen Wörter, für die STIL.md ein deutsches festlegt.
var stilFachwoerter = wortmenge("student students borrower borrowers borrowed borrow book books " +
	"title titles copy copies item items loan loans checkout due return returned reservation " +
	"reservations class classes grade grades subject subjects order orders reorder reorders " +
	"shipment shipments supplier suppliers device devices damage damages fee fees reminder " +
	"overdue user users account accounts permission permissions role roles setting settings " +
	"config configs label labels photo photos lock locked unlock deleted graduate graduates " +
	"teacher teachers inventory session sessions search login logout audit handler handlers " +
	"request response bulk repository repo standard standards default defaults service services svc")

// stilVerben sind die englischen Verben am Anfang eines Funktionsnamens, für die STIL.md ein
// deutsches festlegt.
var stilVerben = wortmenge("get list create update delete save insert upsert add remove find " +
	"fetch load lookup check validate verify generate build make send handle process resolve " +
	"parse print serve sync run execute set is has can register write read count mark extend " +
	"apply ensure calculate compute format render draw store edit start stop sanitize clean " +
	"convert encode decode guard close require post put patch seed query " +
	// Der Erzeuger eines Typs heißt New oder new, in jedem Paket dasselbe Wort.
	"neuer neue neues")

// stilFremdeNamen gibt eine Schnittstelle von Go oder einer Bibliothek vor oder das Protokoll
// HTTP mit dem Namen eines Status (apierrors.BadRequest wie http.StatusBadRequest); sie bleiben.
var stilFremdeNamen = wortmenge("ServeHTTP RoundTrip MarshalJSON UnmarshalJSON WriteHeader " +
	"CopyFrom QueryRow SendBatch GetStringWidth GetCellMargin BadRequest")

var (
	stilKuerzel = regexp.MustCompile(`(Lmf|Dsgvo|Lusd|Dnb|Opac|Isbn|Pdf|Csv|Xlsx|Json|Sql|Http|` +
		`Smtp|Imap|Url|Uuid|Sse|Csrf|Jwt|Ean|Id|Ip|Db)([A-Z0-9]|$)`)
	stilDatum = regexp.MustCompile(`\b\d{1,2}\.\d{1,2}\.20\d{2}\b|\b20\d{2}-\d{2}-\d{2}\b`)
	// stilQuelle trifft ein Datum, das zu einer Quelle gehört: Es benennt einen Erlass oder sagt,
	// wann eine Seite gelesen wurde. Marke und Datum stehen in einer Zeile.
	stilQuelle = regexp.MustCompile(`(Erlass|Beschluss|Gesetz|Verordnung) vom \d{1,2}\.\d{1,2}\.20\d{2}|` +
		`abgerufen am \d{1,2}\.\d{1,2}\.20\d{2}`)
)

// Der Bestand an Namen und Kommentaren, die docs/STIL.md noch nicht folgen, wächst nicht, und
// ein Paket ohne Eintrag bleibt ohne Fund. Die Zahlen stehen in stil_bestand_test.go.
//
// Blindheit: Die Oberfläche liest die Prüfung nicht. In Testdateien zählt sie Namen und
// Dateinamen, aber kein Datum: Dort ist ein Datum meist ein Testwert (eine feste Uhr, ein
// Ferientag). Sie sieht nur Wörter der drei Listen oben; ob ein deutscher Name die richtige Sache trifft, ob ein Name öffentlich
// sein muss und ob ein Kommentar das Warum nennt, liest ein Mensch. Lokale Variablen und
// Parameter zählt sie nicht. Das Datum einer Quelle (stilQuelle) zählt sie nicht als Datum.
func TestStil_BestandWaechstNicht(t *testing.T) {
	ist, funde, namen, gelesen := stilMessen(t)
	if namen < 5000 || gelesen < 60 {
		t.Fatalf("nur %d Namen in %d Paketen gelesen: Der Detektor greift ins Leere", namen, gelesen)
	}
	pakete := map[string]bool{}
	for p := range ist {
		pakete[p] = true
	}
	for p := range stilBestand {
		pakete[p] = true
	}
	for _, p := range sortiert(pakete) {
		soll, gezaehlt := stilBestand[p], ist[p]
		if soll == gezaehlt {
			continue
		}
		zeile := fmt.Sprintf("%q: {Fach: %d, Verb: %d, Kuerzel: %d, Datei: %d, Datum: %d},",
			p, gezaehlt.Fach, gezaehlt.Verb, gezaehlt.Kuerzel, gezaehlt.Datei, gezaehlt.Datum)
		if gezaehlt == (stilZahl{}) {
			zeile = "die Zeile des Pakets entfernen"
		}
		neu := gezaehlt.Fach > soll.Fach || gezaehlt.Verb > soll.Verb || gezaehlt.Kuerzel > soll.Kuerzel ||
			gezaehlt.Datei > soll.Datei || gezaehlt.Datum > soll.Datum
		if !neu {
			t.Errorf("%s: weniger Funde als im Bestand. In stil_bestand_test.go nachziehen: %s", p, zeile)
			continue
		}
		t.Errorf("%s: mehr Funde als im Bestand (%+v, erlaubt %+v). Ein neuer Name folgt docs/STIL.md; "+
			"gefunden sind:\n  %s", p, gezaehlt, soll, strings.Join(funde[p], "\n  "))
	}
}

// stilMessen liest Programmcode und Tests und zählt je Paket die Funde, mit ihren Stellen; dazu
// die Zahl der gelesenen Namen und Pakete.
func stilMessen(t *testing.T) (map[string]stilZahl, map[string][]string, int, int) {
	t.Helper()
	fset := token.NewFileSet()
	ist := map[string]stilZahl{}
	funde := map[string][]string{}
	gelesen := map[string]bool{}
	namen := 0
	err := filepath.WalkDir(".", func(pfad string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if name := d.Name(); name == "node_modules" || name == "frontend" || name == "testdata" ||
				(strings.HasPrefix(name, ".") && pfad != ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(pfad, ".go") || filepath.ToSlash(pfad) == "docs/docs.go" {
			return nil
		}
		test := strings.HasSuffix(pfad, "_test.go")
		datei, err := parser.ParseFile(fset, pfad, nil, parser.ParseComments|parser.SkipObjectResolution)
		if err != nil {
			return err
		}
		paket := filepath.ToSlash(filepath.Dir(pfad))
		gelesen[paket] = true
		z := ist[paket]
		merke := func(art, was string) { funde[paket] = append(funde[paket], art+": "+was+" ("+pfad+")") }
		if hatWort(woerter(strings.TrimSuffix(strings.TrimSuffix(filepath.Base(pfad), ".go"), "_test")), stilFachwoerter) {
			z.Datei++
			merke("Datei", filepath.Base(pfad))
		}
		if !test {
			for _, gruppe := range datei.Comments {
				for _, k := range gruppe.List {
					for _, zeile := range strings.Split(k.Text, "\n") {
						if stilDatum.MatchString(stilQuelle.ReplaceAllString(zeile, "")) {
							z.Datum++
							merke("Datum", strings.TrimSpace(zeile))
						}
					}
				}
			}
		}
		for _, name := range erklaerteNamen(datei) {
			namen++
			if stilFremdeNamen[name.text] {
				continue
			}
			ws := woerter(name.text)
			if hatWort(ws, stilFachwoerter) {
				z.Fach++
				merke("Fachwort", name.text)
			}
			if name.funktion && len(ws) > 1 && stilVerben[ws[0]] {
				z.Verb++
				merke("Verb", name.text)
			}
			if stilKuerzel.MatchString(name.text) {
				z.Kuerzel++
				merke("Kürzel", name.text)
			}
		}
		if z != (stilZahl{}) {
			ist[paket] = z
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Baum lesen: %v", err)
	}
	return ist, funde, namen, len(gelesen)
}

type erklaerterName struct {
	text     string
	funktion bool
}

// erklaerteNamen liefert, was eine Datei auf der Ebene des Pakets benennt: Typen mit ihren
// Feldern, Funktionen, Methoden, Konstanten und Variablen.
func erklaerteNamen(datei *ast.File) []erklaerterName {
	var namen []erklaerterName
	for _, dekl := range datei.Decls {
		switch d := dekl.(type) {
		case *ast.FuncDecl:
			namen = append(namen, erklaerterName{d.Name.Name, true})
		case *ast.GenDecl:
			for _, spez := range d.Specs {
				switch s := spez.(type) {
				case *ast.TypeSpec:
					namen = append(namen, erklaerterName{s.Name.Name, false})
					if st, ok := s.Type.(*ast.StructType); ok {
						for _, feld := range st.Fields.List {
							for _, n := range feld.Names {
								namen = append(namen, erklaerterName{n.Name, false})
							}
						}
					}
				case *ast.ValueSpec:
					for _, n := range s.Names {
						namen = append(namen, erklaerterName{n.Name, false})
					}
				}
			}
		}
	}
	return namen
}

// woerter teilt einen Namen an Großbuchstaben, Ziffern und Unterstrichen und schreibt die
// Teile klein; ein Kürzel in Großbuchstaben bleibt ein Wort (LMFPlan: lmf, plan).
func woerter(name string) []string {
	var teile []string
	r := []rune(name)
	anfang := 0
	schneide := func(ende int) {
		if ende > anfang {
			teile = append(teile, strings.ToLower(string(r[anfang:ende])))
		}
		anfang = ende
	}
	for i := 1; i < len(r); i++ {
		switch {
		case r[i] == '_':
			schneide(i)
			anfang = i + 1
		case unicode.IsUpper(r[i]) && !unicode.IsUpper(r[i-1]):
			schneide(i)
		case unicode.IsUpper(r[i]) && i+1 < len(r) && unicode.IsLower(r[i+1]):
			schneide(i)
		case unicode.IsDigit(r[i]) != unicode.IsDigit(r[i-1]) && r[i-1] != '_':
			schneide(i)
		}
	}
	schneide(len(r))
	return teile
}

func hatWort(ws []string, menge map[string]bool) bool {
	for _, w := range ws {
		if menge[w] {
			return true
		}
	}
	return false
}

func wortmenge(liste string) map[string]bool {
	menge := map[string]bool{}
	for _, w := range strings.Fields(liste) {
		menge[w] = true
	}
	return menge
}

func sortiert(menge map[string]bool) []string {
	schluessel := make([]string, 0, len(menge))
	for s := range menge {
		schluessel = append(schluessel, s)
	}
	sort.Strings(schluessel)
	return schluessel
}
