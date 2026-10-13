// Package uebernahme trägt die Teile einer Übernahme des Altbestands, die nicht von der Quelle
// abhängen: das Protokoll, das Abwertung und Ausfall trennt, die Einordnung der Fehler von
// Postgres, die Prüfung der ISBN, die Spaltenbreiten und den Savepoint je Datensatz.
//
// Sie stehen in einem eigenen Paket, weil cmd/migrate und der Schreiber für Littera dieselben
// brauchen. In zwei Fassungen fehlte der zweiten leicht, was die erste gelernt hat, und ein
// fehlender Savepoint fällt lange nicht auf.
package uebernahme

import (
	"bufio"
	"fmt"
	"os"
	"time"
)

// schweregrad einer Zeile im Protokoll.
//
// Eine WARNUNG heißt: Der Datensatz steht in der neuen Datenbank, aber abgewertet (ISBN
// verworfen, Freitext gekürzt). Ein FEHLER heißt: Der Datensatz steht nicht darin. Liefen
// beide in denselben Zähler, ließe sich nicht sagen, ob 3.000 Meldungen 3.000 verlorene Titel
// oder 3.000 gekürzte Felder bedeuten.
type schweregrad string

// Die zwei Stufen, in denen ein Lauf meldet; ihre Bedeutung steht am Typ.
const (
	schweregradWarnung schweregrad = "WARNUNG"
	schweregradFehler  schweregrad = "FEHLER"
)

// Protokoll schreibt die Anmerkungen eines Laufs in eine Datei und zählt sie getrennt nach
// Schweregrad.
//
// idFeld benennt den Schlüssel der Quelle („mysql_id", „littera_id"). Er steht in jeder Zeile,
// damit sich der Datensatz in der Quelldatei wiederfinden lässt.
type Protokoll struct {
	datei     *os.File
	schreiber *bufio.Writer
	idFeld    string
	warnungen int // Datensatz übernommen, aber abgewertet
	fehler    int // Datensatz nicht übernommen
}

// NewProtokoll legt die Protokolldatei an; was schon darin steht, wird verworfen.
func NewProtokoll(pfad, idFeld string) (*Protokoll, error) {
	// #nosec G304 - der Pfad stammt aus dem Aufruf des Werkzeugs bzw. aus dem Test
	datei, err := os.OpenFile(pfad, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return nil, fmt.Errorf("konnte die Protokolldatei %s nicht öffnen: %w", pfad, err)
	}
	return &Protokoll{datei: datei, schreiber: bufio.NewWriterSize(datei, 64*1024), idFeld: idFeld}, nil
}

// Warnung protokolliert eine Abwertung: Der Datensatz kommt an, aber nicht unverändert.
func (p *Protokoll) Warnung(quellID, kennung, grund string) {
	p.schreibe(schweregradWarnung, quellID, kennung, grund)
}

// Fehler protokolliert einen Ausfall: Dieser Datensatz fehlt in der neuen Datenbank.
func (p *Protokoll) Fehler(quellID, kennung, grund string) {
	p.schreibe(schweregradFehler, quellID, kennung, grund)
}

func (p *Protokoll) schreibe(grad schweregrad, quellID, kennung, grund string) {
	switch grad {
	case schweregradWarnung:
		p.warnungen++
	case schweregradFehler:
		p.fehler++
	}
	zeit := time.Now().Format("2006-01-02 15:04:05")
	//nolint:errcheck // ein fehlgeschlagener Protokollschreib darf die Übernahme nicht abbrechen
	_, _ = fmt.Fprintf(p.schreiber, "[%s] %s %s=%s kennung=%q grund=%s\n", zeit, grad, p.idFeld, quellID, kennung, grund)
}

// Warnungen zählt die abgewertet übernommenen Datensätze.
func (p *Protokoll) Warnungen() int { return p.warnungen }

// FehlerAnzahl zählt die nicht übernommenen Datensätze.
func (p *Protokoll) FehlerAnzahl() int { return p.fehler }

// Leere schreibt den Puffer heraus, ohne die Datei zu schließen: für Tests und für
// Zwischenstände bei langen Läufen.
func (p *Protokoll) Leere() error { return p.schreiber.Flush() }

// Schliesse leert den Puffer und schließt die Datei. Ohne diesen Aufruf steht nach einem Lauf,
// der mit os.Exit endet, nichts im Protokoll, weil 64 KB gepuffert werden.
func (p *Protokoll) Schliesse() {
	_ = p.schreiber.Flush() //nolint:errcheck
	_ = p.datei.Close()     //nolint:errcheck
}
