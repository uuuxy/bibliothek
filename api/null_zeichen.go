package api

import (
	"bytes"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"

	"bibliothek/apierrors"
)

// meldungNullZeichen sagt, wie die Person den Fehler vermeidet (M3, Text fields, Error text):
// Das Zeichen kommt nur über Einfügen oder einen Scanner in ein Feld.
const meldungNullZeichen = "Die Eingabe enthält ein unsichtbares Zeichen. Bitte sie neu eintippen."

// nullZeichenVorlauf ist, wie viel eines JSON-Rumpfs der Eingang vor der Tür liest. Ein Rumpf
// bis zu dieser Größe wird abgewiesen, bevor eine Tür läuft; bei einem größeren scheitert erst
// das Lesen in der Tür. Mehr vorab zu lesen hieße, vor der Anmeldung zu puffern.
const nullZeichenVorlauf = 64 << 10

// errNullZeichen meldet ein Null-Zeichen im JSON-Rumpf.
var errNullZeichen = errors.New("null-zeichen im rumpf")

// nullZeichenMiddleware weist eine Anfrage mit Null-Zeichen (U+0000) in Pfad, Abfrage oder
// JSON-Rumpf mit 400 ab. PostgreSQL nimmt das Zeichen in keinem Text an (SQLSTATE 22021, in jsonb
// 22P05), und jede Tür, die es durchreichte, antwortete 500. Ein Rumpf anderer Art bleibt
// ungelesen: Eine hochgeladene Datei darf Null-Bytes tragen, ihre Zeilen prüft der Import.
func nullZeichenMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.ContainsRune(r.URL.Path, 0) || hatNullZeichen(r.URL.RawQuery) {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, meldungNullZeichen, nil)
			return
		}
		if r.Body != nil && r.Body != http.NoBody && istJSONRumpf(r.Header.Get("Content-Type")) {
			leser := &nullZeichenLeser{quelle: r.Body}
			vorlauf, err := io.ReadAll(io.LimitReader(leser, nullZeichenVorlauf))
			if errors.Is(err, errNullZeichen) {
				apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, meldungNullZeichen, err)
				return
			}
			// Jeder andere Lesefehler bleibt im Leser stehen und erreicht die Tür hinter dem
			// Vorlauf, wie ohne diese Prüfung.
			r.Body = gelesenerRumpf{Reader: io.MultiReader(bytes.NewReader(vorlauf), leser), Closer: r.Body}
		}
		next.ServeHTTP(w, r)
	})
}

// hatNullZeichen sieht in der rohen Abfrage nach, damit auch ein Wert zählt, den url.ParseQuery
// wegen eines anderen Fehlers nicht mehr liefert.
func hatNullZeichen(roh string) bool {
	return strings.Contains(roh, "%00") || strings.ContainsRune(roh, 0)
}

// istJSONRumpf gilt für JSON und für einen Rumpf ohne Angabe der Art: Eine Tür mit eigenem
// Decoder liest ihn als JSON, gleich was die Kopfzeile sagt.
func istJSONRumpf(art string) bool {
	if art == "" {
		return true
	}
	medientyp, _, err := mime.ParseMediaType(art)
	return err == nil && (medientyp == "application/json" || strings.HasSuffix(medientyp, "+json"))
}

// gelesenerRumpf ist der Rumpf nach dem Vorlauf: erst die gelesenen Bytes, dann der Rest.
type gelesenerRumpf struct {
	io.Reader
	io.Closer
}

// nullZeichenLeser liest einen JSON-Rumpf und scheitert an einem Null-Zeichen, ob roh oder als
// \u0000 geschrieben. Ein Backslash vor dem Backslash hebt ihn auf: `\\u0000` ist Text. Der
// Zustand überdauert die Grenze zwischen zwei Lesevorgängen, und ein Fehler bleibt stehen.
type nullZeichenLeser struct {
	quelle io.Reader
	// zustand: 0 im Text, 1 nach einem Backslash, 2 bis 5 nach \u und so vielen Nullen wie
	// zustand-2.
	zustand int
	fehler  error
}

func (l *nullZeichenLeser) Read(p []byte) (int, error) {
	if l.fehler != nil {
		return 0, l.fehler
	}
	n, err := l.quelle.Read(p)
	for _, b := range p[:n] {
		if l.fehler = l.pruefe(b); l.fehler != nil {
			return 0, l.fehler
		}
	}
	if err != nil {
		l.fehler = err
	}
	return n, err
}

// pruefe schreitet den Zustand um ein Byte fort und scheitert, wenn damit ein Null-Zeichen
// vollständig ist.
func (l *nullZeichenLeser) pruefe(b byte) error {
	if b == 0 {
		return errNullZeichen
	}
	switch l.zustand {
	case 0:
		if b == '\\' {
			l.zustand = 1
		}
	case 1:
		l.zustand = 0
		if b == 'u' {
			l.zustand = 2
		}
	default:
		if b != '0' {
			l.zustand = 0
			return nil
		}
		l.zustand++
		if l.zustand == 6 {
			return errNullZeichen
		}
	}
	return nil
}
