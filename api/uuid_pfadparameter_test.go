package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// uuidPfadParameter ist eine Liste von Hand — und hatte keinen Test (OFFEN.md 5.12).
// Zwei Dinge können dort still falsch werden: Ein neuer Pfad-Parameter, dessen Spalte
// UUID ist, fehlt in der Liste (dann geht sein Wert ungeprüft an Postgres, 22P02 → 500).
// Oder ein Name steht drin, den kein Pfad mehr trägt (dann behauptet die Liste einen
// Schutz, den es nicht gibt). Deshalb: Jeder Platzhalter aus den Registrierungen ist
// entweder in uuidPfadParameter oder hier mit dem Grund eingeordnet, warum er KEINE UUID
// ist — und jeder Eintrag der Liste kommt in den Pfaden vor.
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
	dateien, err := filepath.Glob("routes_*.go")
	if err != nil {
		t.Fatal(err)
	}
	dateien = append(dateien, "router.go", filepath.Join("..", "inventur", "api_routen.go"))

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

// Jeder Eintrag der Liste wird auch wirklich geprüft — und zwar so streng wie Postgres:
// `urn:uuid:…` nimmt uuid.Parse an, die Datenbank nicht (13.09.2026 am Stack nachgestellt,
// 500). Die Klammerform nimmt Postgres an, hier ist sie trotzdem keine Kennung des Hauses.
func TestUUIDPfadParameter_JederEintragWirdStrengGeprueft(t *testing.T) {
	ziel := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	const gueltig = "3f2504e0-4f89-11d3-9a0c-0305e82c3301"
	for _, name := range uuidPfadParameter {
		mux := http.NewServeMux()
		mux.Handle("GET /probe/{"+name+"}", ValidateUUIDParamsMiddleware(ziel))
		for wert, erwartet := range map[string]int{
			gueltig:               http.StatusOK,
			"kein-uuid":           http.StatusBadRequest,
			"urn:uuid:" + gueltig: http.StatusBadRequest,
			"{" + gueltig + "}":   http.StatusBadRequest,
		} {
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/probe/"+wert, nil))
			if rec.Code != erwartet {
				t.Errorf("{%s} = %q: erwartet %d, war %d", name, wert, erwartet, rec.Code)
			}
		}
	}
}
