package csvutil

import (
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
)

// ErrKodierung meldet eine Datei, deren Kodierung sich nicht sicher bestimmen lässt.
var ErrKodierung = errors.New("kodierung der Datei nicht eindeutig")

// MeldungKodierung ist der Satz der Türen zu ErrKodierung.
const MeldungKodierung = "Die Umlaute dieser Datei lassen sich nicht sicher lesen. " +
	"Bitte in Excel als „CSV UTF-8“ speichern und erneut hochladen."

// Excel schreibt „CSV“ unter Windows in Windows-1252, „CSV (MS-DOS)“ in Codepage 850 und auf dem
// Mac in jeder Form in Macintosh. Ein ü ist dort 0xFC, 0x81 und 0x9F; die Listen nennen die Bytes,
// an denen sich die Kodierung zeigt.
const (
	umlauteWindows    = "\xE4\xF6\xFC\xDF\xC4\xD6\xDC"         // ä ö ü ß Ä Ö Ü
	amWortCodepage850 = "\x81"                                 // ü, in Windows-1252 nicht belegt
	imWortCodepage850 = "\x84\x94"                             // ä ö, in Windows-1252 „ ”
	amWortMacintosh   = "\x9F"                                 // ü, in Windows-1252 Ÿ
	imWortMacintosh   = "\xA7"                                 // ß, in Windows-1252 §
	mehrdeutig        = "\x80\x85\x86\x8A\x8E\x99\x9A\xA7\xE1" // € … † Š Ž ™ š § á, anderswo ein Umlaut
	unbelegtWindows   = "\x81\x8D\x8F\x90\x9D"
)

// spuren hält, was die Bytes einer Datei über ihre Kodierung verraten.
type spuren struct {
	windows, codepage850, macintosh bool
	mehrdeutig, unbelegt, utf8Folge bool
}

// leseText macht aus den Bytes einer Datei Text. Nullzeichen fallen weg: Die Datenbank lehnte sie
// ab, und die Übernahme endete mit einem Serverfehler. „CSV (Macintosh)“ trennt die Zeilen nur mit
// CR; encoding/csv sähe darin eine einzige Zeile.
func leseText(inhalt []byte) (string, error) {
	text, err := dekodiere(inhalt)
	if err != nil {
		return "", err
	}
	text = strings.ReplaceAll(text, "\x00", "")
	if !strings.Contains(text, "\n") {
		text = strings.ReplaceAll(text, "\r", "\n")
	}
	return text, nil
}

func dekodiere(inhalt []byte) (string, error) {
	if utf8.Valid(inhalt) {
		return strings.TrimPrefix(string(inhalt), "\uFEFF"), nil
	}
	s := sammleSpuren(inhalt)
	kodierung, err := s.kodierung()
	if err != nil {
		return "", err
	}
	var text strings.Builder
	text.Grow(len(inhalt))
	for _, zeichen := range inhalt {
		text.WriteRune(kodierung.DecodeByte(zeichen))
	}
	return text.String(), nil
}

func sammleSpuren(inhalt []byte) spuren {
	var s spuren
	for i, zeichen := range inhalt {
		if zeichen < 0x80 {
			continue
		}
		links := i > 0 && istBuchstabe(inhalt[i-1])
		rechts := i+1 < len(inhalt) && istBuchstabe(inhalt[i+1])
		s.merke(zeichen, links, rechts)
		// „ü“ in UTF-8 steht in Windows-1252 als „Ã¼“ da; das kommt in einem Namen nicht vor.
		if zeichen == 0xC3 && i+1 < len(inhalt) && inhalt[i+1]&0xC0 == 0x80 {
			s.utf8Folge = true
		}
	}
	return s
}

// merke wertet ein Byte mit seinen Nachbarn aus. Die Satzzeichen zählen erst mitten im Wort, denn
// „Harry Potter“ trägt sie auch in Windows-1252 neben einem Buchstaben.
func (s *spuren) merke(zeichen byte, links, rechts bool) {
	amWort, imWort := links || rechts, links && rechts
	s.windows = s.windows || amWort && enthaelt(umlauteWindows, zeichen)
	s.codepage850 = s.codepage850 || amWort && enthaelt(amWortCodepage850, zeichen) ||
		imWort && enthaelt(imWortCodepage850, zeichen)
	s.macintosh = s.macintosh || amWort && enthaelt(amWortMacintosh, zeichen) ||
		imWort && enthaelt(imWortMacintosh, zeichen)
	s.mehrdeutig = s.mehrdeutig || amWort && enthaelt(mehrdeutig, zeichen)
	s.unbelegt = s.unbelegt || enthaelt(unbelegtWindows, zeichen)
}

// kodierung nimmt eine Kodierung nur, wenn genau sie zu den Spuren passt. Falsche Zeichen landeten
// still im Katalog, eine Ablehnung kostet nur ein neues Speichern. Ohne jede Spur gilt Windows-1252.
func (s *spuren) kodierung() (*charmap.Charmap, error) {
	switch {
	case s.utf8Folge:
		return nil, ErrKodierung
	case s.windows && !s.codepage850 && !s.macintosh && !s.unbelegt:
		return charmap.Windows1252, nil
	case s.codepage850 && !s.windows && !s.macintosh:
		return charmap.CodePage850, nil
	case s.macintosh && !s.windows && !s.codepage850:
		return charmap.Macintosh, nil
	case !s.windows && !s.codepage850 && !s.macintosh && !s.mehrdeutig && !s.unbelegt:
		return charmap.Windows1252, nil
	}
	return nil, ErrKodierung
}

func istBuchstabe(zeichen byte) bool {
	return unicode.IsLetter(charmap.Windows1252.DecodeByte(zeichen))
}

func enthaelt(menge string, zeichen byte) bool {
	return strings.IndexByte(menge, zeichen) >= 0
}
