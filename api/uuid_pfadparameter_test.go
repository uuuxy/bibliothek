package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"bibliothek/db"
	"bibliothek/internal/tuertest"
	"bibliothek/sse"
)

// uuidPfadParameter ist eine Liste von Hand. Zwei Dinge können dort still falsch werden: Ein
// neuer Pfad-Parameter, dessen Spalte UUID ist, fehlt in der Liste (dann geht sein Wert
// ungeprüft an Postgres, 22P02 → 500). Oder ein Name steht drin, den kein Pfad mehr trägt (dann
// behauptet die Liste einen Schutz, den es nicht gibt). Deshalb ist jeder Platzhalter aus den
// Anmeldungen entweder in uuidPfadParameter oder hier mit dem Grund eingeordnet, warum er keine
// UUID ist, und jeder Eintrag der Liste kommt in den Pfaden vor.
var keineUUIDPfadParameter = map[string]string{
	"token":      "256-Bit-Bestätigungs-Token der Lieferantenmail (nur als Hash in der DB)",
	"art":        "Art des LMF-Plans (ausgabe/rueckgabe), CHECK-Constraint",
	"klasse":     "Klassenname (klassen.name, VARCHAR)",
	"groesse":    "Etikettengröße klein/gross",
	"barcode_id": "Barcode (VARCHAR(100)) — genau die Verwechslung, die das Foto-GET lahmgelegt hatte",
	"isbn":       "ISBN (VARCHAR)",
}

var pfadPlatzhalter = regexp.MustCompile(`\{([a-z_]+)\}`)

func TestUUIDPfadParameter_JederPlatzhalterIstEingeordnet(t *testing.T) {
	dateien, err := filepath.Glob("routen_*.go")
	if err != nil {
		t.Fatal(err)
	}
	dateien = append(dateien, "router.go")

	vorkommen := map[string]int{}
	for _, datei := range dateien {
		inhalt, err := os.ReadFile(datei) // #nosec G304 -- Repo-Dateien
		if err != nil {
			t.Fatalf("%s lesen: %v", datei, err)
		}
		for _, m := range pfadPlatzhalter.FindAllStringSubmatch(string(inhalt), -1) {
			vorkommen[m[1]]++
		}
	}
	// Nicht-leer-Garantie: Findet der Scanner kaum Platzhalter, misst er nichts.
	if len(vorkommen) < 5 {
		t.Fatalf("nur %d verschiedene Platzhalter gefunden — der Scanner greift nicht mehr: %v", len(vorkommen), vorkommen)
	}

	uuidNamen := map[string]bool{}
	for _, name := range uuidPfadParameter {
		uuidNamen[name] = true
	}
	for name := range vorkommen {
		_, keineUUID := keineUUIDPfadParameter[name]
		switch {
		case uuidNamen[name] && keineUUID:
			t.Errorf("{%s} steht in uuidPfadParameter UND in keineUUIDPfadParameter — eines von beiden lügt", name)
		case !uuidNamen[name] && !keineUUID:
			t.Errorf("{%s} ist ein neuer Pfad-Parameter ohne Einordnung: Ist die Spalte dahinter UUID, gehört er in "+
				"uuidPfadParameter (api/middleware.go); sonst mit Grund in keineUUIDPfadParameter.", name)
		}
	}
	for _, name := range uuidPfadParameter {
		if vorkommen[name] == 0 {
			t.Errorf("uuidPfadParameter nennt %q, aber kein Pfad trägt {%s} — der Eintrag behauptet einen Schutz, den es nicht gibt", name, name)
		}
	}
	for name := range keineUUIDPfadParameter {
		if vorkommen[name] == 0 {
			t.Errorf("keineUUIDPfadParameter nennt %q, aber kein Pfad trägt {%s} — Eintrag austragen", name, name)
		}
	}
}

// Jeder Eintrag der Liste wird auch wirklich geprüft, und zwar so streng wie Postgres:
// `urn:uuid:…` nimmt uuid.Parse an, die Datenbank nicht (an einer lokalen Anlage nachgestellt,
// 500). Die Klammerform nimmt Postgres an, hier ist sie trotzdem keine Kennung des Hauses.
func TestUUIDPfadParameter_JederEintragWirdStrengGeprueft(t *testing.T) {
	ziel := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	const gueltig = "3f2504e0-4f89-11d3-9a0c-0305e82c3301"
	for _, name := range uuidPfadParameter {
		mux := http.NewServeMux()
		mux.Handle("GET /probe/{"+name+"}", uuidPfadMiddleware(ziel))
		for wert, erwartet := range map[string]int{
			gueltig:               http.StatusOK,
			"kein-uuid":           http.StatusBadRequest,
			"urn:uuid:" + gueltig: http.StatusBadRequest,
			"{" + gueltig + "}":   http.StatusBadRequest,
		} {
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/probe/"+wert, nil))
			if rec.Code != erwartet {
				t.Errorf("{%s} = %q: Status = %d, erwartet %d", name, wert, rec.Code, erwartet)
			}
		}
	}
}

// Die Prüfung sitzt in verlangeRecht und nicht um den Mux. Eine Route, die den Zugriff anders
// prüft, etwa mit VerlangeAnmeldung, reichte eine falsche Kennung an Postgres weiter (22P02,
// 500). Deshalb geht an jede angemeldete Route mit einem Platzhalter aus der Liste eine Anfrage
// durch den ganzen Router, ohne Sitzung und mit „kein-uuid" an dieser Stelle; sie muss so
// antworten wie uuidPfadMiddleware selbst.
func TestUUIDPfadParameter_JedeRoutePrueftDieKennung(t *testing.T) {
	routen := registrierteRouten(t)

	soll := httptest.NewRecorder()
	probe := http.NewServeMux()
	probe.Handle("GET /probe/{id}", uuidPfadMiddleware(http.NotFoundHandler()))
	probe.ServeHTTP(soll, httptest.NewRequest(http.MethodGet, "/probe/kein-uuid", nil))
	if soll.Code != http.StatusBadRequest {
		t.Fatalf("uuidPfadMiddleware mit kein-uuid: Status %d, erwartet 400", soll.Code)
	}
	sollMeldung := tuertest.Fehlermeldung(t, soll)

	// Alle Anfragen kommen von derselben Adresse; die Bremse zählt hier nicht mit.
	t.Setenv("RATE_LIMIT", "100000")
	// Routes legt das Verzeichnis uploads an. Die Routendateien sind da schon gelesen.
	t.Chdir(t.TempDir())
	router := NewServer(&db.Verbindung{}, nil, sse.NewBroker(), ServerKonfig{}).Routes()

	const gueltig = "3f2504e0-4f89-11d3-9a0c-0305e82c3301"
	geprueft := 0
	for muster := range routen {
		methode, pfad, mitMethode := strings.Cut(muster, " ")
		if !mitMethode {
			methode, pfad = http.MethodGet, muster
		}
		for _, name := range uuidPfadParameter {
			if !strings.Contains(pfad, "{"+name+"}") {
				continue
			}
			adresse := pfadPlatzhalter.ReplaceAllStringFunc(pfad, func(platzhalter string) string {
				anderer := strings.Trim(platzhalter, "{}")
				if anderer == name {
					return "kein-uuid"
				}
				if slices.Contains(uuidPfadParameter, anderer) {
					return gueltig
				}
				return "x"
			})
			if strings.ContainsAny(adresse, "{}") {
				t.Errorf("%s: Platzhalter in einer Form, die der Test nicht einsetzt (%s)", muster, adresse)
				continue
			}
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, tuertest.MitCSRF(httptest.NewRequest(methode, adresse, nil)))
			geprueft++
			if rec.Code != soll.Code {
				t.Errorf("%s %s: Status %d, erwartet %d wie uuidPfadMiddleware.\n"+
					"→ Die Route reicht die Kennung ungeprüft an die Tür weiter. Mit verlangeRecht anmelden; "+
					"genügt ihr die Sitzung: uuidPfadMiddleware(s.VerlangeAnmeldung()(…)).", methode, adresse, rec.Code, soll.Code)
				continue
			}
			if got := tuertest.Fehlermeldung(t, rec); got != sollMeldung {
				t.Errorf("%s %s: Meldung %q, erwartet %q wie uuidPfadMiddleware", methode, adresse, got, sollMeldung)
			}
		}
	}
	if geprueft < 50 {
		t.Fatalf("nur %d Routen mit Kennung im Pfad geprüft, erwartet über 60 — registrierteRouten oder das Einsetzen greift nicht mehr", geprueft)
	}
}
