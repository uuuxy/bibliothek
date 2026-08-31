package jobs

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Ratsche: Kein Löschjob formuliert seine Bedingung selbst.
//
// Anlass (31.08.2026): cron_dsgvo_anliegen.go schrieb sein WHERE als Literal, während
// der Rückstands-Wächter (repository/loeschrueckstand.go) dieselbe Frage über
// PredikatAnliegen stellte. Beide stimmten nur zufällig überein — genau die Bauform,
// gegen die der Kopf von repository/loeschfristen.go geschrieben ist: „Zwei
// Wahrheitsquellen für dieselbe Frage sind hier kein Schönheitsfehler, sondern der
// schlimmste denkbare Fehler: Der Wächter beruhigt, während der Job schläft." Eine
// Ausnahme im Prädikat hätte der Job ignoriert, und der Wächter hätte weiter „0
// Rückstand" gemeldet.
//
// Regel: Eine Datei mit einem löschenden/anonymisierenden Statement muss das Prädikat
// aus repository/ beziehen (repository.Predikat…). Neue Löschroutinen werden sonst rot.
var (
	loeschStatement = regexp.MustCompile(`(?i)\b(DELETE\s+FROM|UPDATE)\s+[a-z_]+`)
	pruefeAlsQuelle = regexp.MustCompile(`repository\.Predikat`)
)

func TestLoeschjobsTeilenIhrPraedikat(t *testing.T) {
	eintraege, err := filepath.Glob("cron_*.go")
	if err != nil {
		t.Fatalf("jobs/ nicht lesbar: %v", err)
	}
	// Gegenprobe am Detektor: Ohne Fundstellen prüfte dieser Test ins Leere.
	var mitStatement []string
	for _, pfad := range eintraege {
		if strings.HasSuffix(pfad, "_test.go") {
			continue
		}
		roh, err := os.ReadFile(filepath.Clean(pfad))
		if err != nil {
			t.Fatalf("%s nicht lesbar: %v", pfad, err)
		}
		quelle := ohneKommentarzeilen(string(roh))
		if !loeschStatement.MatchString(quelle) {
			continue
		}
		mitStatement = append(mitStatement, pfad)
		if !pruefeAlsQuelle.MatchString(quelle) {
			t.Errorf("jobs/%s löscht oder anonymisiert mit einer EIGENEN Bedingung.\n"+
				"Das Prädikat gehört nach repository/loeschfristen.go, damit Job und\n"+
				"Rückstands-Wächter denselben Satz benutzen (sonst beruhigt der Wächter,\n"+
				"während der Job schläft).", pfad)
		}
	}
	if len(mitStatement) < 3 {
		t.Fatalf("nur %d Löschjobs gefunden (%v) — der Detektor misst offenbar nichts mehr",
			len(mitStatement), mitStatement)
	}
}

// ohneKommentarzeilen blendet //-Kommentare aus: Erklärtexte nennen „DELETE FROM"
// häufig, ohne dass dort ein Statement steht.
func ohneKommentarzeilen(quelle string) string {
	var b strings.Builder
	for zeile := range strings.Lines(quelle) {
		if i := strings.Index(zeile, "//"); i >= 0 {
			zeile = zeile[:i]
		}
		b.WriteString(zeile)
		b.WriteString("\n")
	}
	return b.String()
}
