// Package uebernahme trägt die Teile einer Altbestands-Übernahme, die nicht von der
// Quelle abhängen: das Protokoll mit seiner Trennung von Abwertung und Ausfall, die
// Einordnung von Postgres-Fehlern, die ISBN-Prüfung, die Spaltenbreiten und der
// Savepoint je Datensatz.
//
// Sie stammen aus cmd/migrate/pg_writer.go, wo sie gegen echtes PostgreSQL gehärtet
// wurden. Herausgelöst wurden sie, als der Littera-Schreibpfad dazukam: Zwei Kopien
// dieser Logik hätten bedeutet, dass die zweite die Fehler der ersten noch einmal macht
// — und der Savepoint ist genau der Teil, dessen Fehlen jahrelang unbemerkt blieb.
package uebernahme

import (
	"bufio"
	"fmt"
	"os"
	"time"
)

// Schweregrad einer Protokollzeile.
//
// Die Trennung ist der Kern der ehrlichen Meldung: eine WARNUNG heißt „der Datensatz
// steht in der neuen Datenbank, aber abgewertet" (ISBN verworfen, Freitext gekürzt),
// ein FEHLER heißt „dieser Datensatz steht NICHT drin". Liefen beide in denselben
// Zähler, könnte niemand unterscheiden, ob 3.000 Meldungen 3.000 verlorene Titel oder
// 3.000 kosmetische Abwertungen bedeuten.
type Schweregrad string

const (
	SchweregradWarnung Schweregrad = "WARNUNG"
	SchweregradFehler  Schweregrad = "FEHLER"
)

// Protokoll schreibt die Anmerkungen eines Laufs in eine Datei und zählt sie getrennt
// nach Schweregrad.
//
// idFeld benennt den Schlüssel der Quelle („mysql_id", „littera_id"). Er steht in jeder
// Zeile, weil das Protokoll nur dann etwas wert ist, wenn man den Datensatz in der
// Quelldatei wiederfindet.
type Protokoll struct {
	f         *os.File
	w         *bufio.Writer
	idFeld    string
	warnungen int // Datensatz übernommen, aber abgewertet
	fehler    int // Datensatz NICHT übernommen
}

// NeuesProtokoll legt die Protokolldatei an (vorhandene Inhalte werden verworfen).
func NeuesProtokoll(pfad, idFeld string) (*Protokoll, error) {
	// #nosec G304 - der Pfad stammt aus dem Aufruf des Werkzeugs bzw. aus dem Test
	f, err := os.OpenFile(pfad, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return nil, fmt.Errorf("konnte die Protokolldatei %s nicht öffnen: %w", pfad, err)
	}
	return &Protokoll{f: f, w: bufio.NewWriterSize(f, 64*1024), idFeld: idFeld}, nil
}

// Warnung protokolliert eine Abwertung: der Datensatz kommt an, aber nicht unverändert.
func (p *Protokoll) Warnung(quellID, kennung, grund string) {
	p.schreibe(SchweregradWarnung, quellID, kennung, grund)
}

// Fehler protokolliert einen Ausfall: dieser Datensatz fehlt in der neuen Datenbank.
func (p *Protokoll) Fehler(quellID, kennung, grund string) {
	p.schreibe(SchweregradFehler, quellID, kennung, grund)
}

func (p *Protokoll) schreibe(sev Schweregrad, quellID, kennung, grund string) {
	switch sev {
	case SchweregradWarnung:
		p.warnungen++
	case SchweregradFehler:
		p.fehler++
	}
	ts := time.Now().Format("2006-01-02 15:04:05")
	//nolint:errcheck // ein fehlgeschlagener Protokollschreib darf die Übernahme nicht abbrechen
	_, _ = fmt.Fprintf(p.w, "[%s] %s %s=%s kennung=%q grund=%s\n", ts, sev, p.idFeld, quellID, kennung, grund)
}

// Warnungen zählt die abgewertet übernommenen Datensätze.
func (p *Protokoll) Warnungen() int { return p.warnungen }

// Fehler zählt die NICHT übernommenen Datensätze.
func (p *Protokoll) FehlerAnzahl() int { return p.fehler }

// Leeren schreibt den Puffer heraus, ohne die Datei zu schließen — für Tests und für
// Zwischenstände bei langen Läufen.
func (p *Protokoll) Leeren() error { return p.w.Flush() }

// Schliessen leert den Puffer und schließt die Datei. Ohne diesen Aufruf steht in einem
// per os.Exit beendeten Lauf nichts im Protokoll: es wird über 64 KB gepuffert.
func (p *Protokoll) Schliessen() {
	_ = p.w.Flush() //nolint:errcheck
	_ = p.f.Close() //nolint:errcheck
}
